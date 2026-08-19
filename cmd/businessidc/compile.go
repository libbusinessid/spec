package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/libbusinessid/spec/internal/artifact"
	"github.com/libbusinessid/spec/internal/conformance"
	"github.com/libbusinessid/spec/internal/diagnostics"
	"github.com/libbusinessid/spec/internal/features"
	"github.com/libbusinessid/spec/internal/reference"
	"github.com/libbusinessid/spec/internal/version"
)

// buildOptions are the inputs shared by compile, verify and check-generated.
type buildOptions struct {
	rules        string
	cases        string
	fixtures     string
	moduleRoot   string
	rulesVersion string
	sourceCommit string
	optimize     bool
	release      bool
	json         bool
}

func (o *buildOptions) bind(fs *flag.FlagSet) {
	fs.StringVar(&o.rules, "rules", "rules", "directory holding the `*.hcl` rule sources")
	fs.StringVar(&o.cases, "cases", "conformance", "directory holding the `*.jsonl` conformance corpus")
	fs.StringVar(&o.fixtures, "fixtures", "testdata", "root of the `load_ruleset` fixtures")
	fs.StringVar(&o.moduleRoot, "module-root", ".", "directory holding go.mod, used to build the SBOM")
	fs.StringVar(&o.rulesVersion, "rules-version", "", "business version; defaults to the RULES_VERSION file")
	fs.StringVar(&o.sourceCommit, "source-commit", "", "commit recorded in the manifest")
	fs.BoolVar(&o.optimize, "optimize", true, "deduplicate identical sub-graphs")
	fs.BoolVar(&o.release, "release", false, "require a reproducible build with SOURCE_DATE_EPOCH")
	fs.BoolVar(&o.json, "json", false, "render the diagnostics as JSON on stdout")
}

// build runs the whole pipeline and returns every artifact.
type buildResult struct {
	rules        *artifact.CompileResult
	suite        *conformance.CompileResult
	manifest     *artifact.Manifest
	files        map[string][]byte
	generatedAt  string
	reproducible bool
}

//nolint:funlen // the build is one linear, reviewable sequence.
func build(opts buildOptions) (*buildResult, *diagnostics.Bag) {
	bag := diagnostics.New()
	rulesVersion, ok := resolveRulesVersion(bag, opts)
	if !ok {
		return nil, bag
	}
	epoch, reproducible, err := sourceDateEpoch(opts.release)
	if err != nil {
		bag.Errorf(diagnostics.Position{}, "CLI012", "%v", err)
		return nil, bag
	}

	rules, rulesBag := artifact.CompileRules(opts.rules, artifact.CompileOptions{
		RulesVersion: rulesVersion,
		Optimize:     opts.optimize,
	})
	bag.Extend(rulesBag)
	if bag.HasErrors() {
		return nil, bag
	}
	suite, suiteBag := conformance.Compile(opts.cases, conformance.CompileOptions{
		RulesVersion:  rulesVersion,
		FormatVersion: rules.Bundle.GetFormatVersion(),
		FixtureRoot:   opts.fixtures,
	})
	bag.Extend(suiteBag)
	if bag.HasErrors() {
		return nil, bag
	}

	inputs, ok := readBuildInputs(bag, opts)
	if !ok {
		return nil, bag
	}

	jsonl, err := conformance.WriteCanonicalJSONL(suite.Cases)
	if err != nil {
		bag.Errorf(diagnostics.Position{}, "CLI013", "cannot render the canonical corpus: %v", err)
		return nil, bag
	}
	jsonlGz, err := artifact.Gzip(jsonl, epoch)
	if err != nil {
		bag.Errorf(diagnostics.Position{}, "CLI014", "cannot compress the corpus: %v", err)
		return nil, bag
	}

	irDoc := artifact.RenderIRDoc()
	featuresDoc := artifact.RenderFeaturesDoc()
	coverageInput := artifact.CoverageInput{Ruleset: rules.Ruleset, Conformance: suite.Bundle}
	coverageDoc := artifact.RenderCoverageDoc(coverageInput)
	generatedAt := artifact.FormatEpoch(epoch)

	sbom, err := artifact.RenderSBOM(inputs.modulePath, inputs.modules, rulesVersion,
		version.Compiler, generatedAt)
	if err != nil {
		bag.Errorf(diagnostics.Position{}, "CLI019", "cannot render the SBOM: %v", err)
		return nil, bag
	}

	manifest := buildManifest(manifestInput{
		opts:         opts,
		rulesVersion: rulesVersion,
		generatedAt:  generatedAt,
		reproducible: reproducible,
		rules:        rules,
		suite:        suite,
		inputs:       inputs,
		jsonlGz:      jsonlGz,
		irDoc:        irDoc,
		featuresDoc:  featuresDoc,
		coverage:     coverageInput,
	})
	manifestBytes, err := manifest.Encode()
	if err != nil {
		bag.Errorf(diagnostics.Position{}, "CLI020", "cannot render the manifest: %v", err)
		return nil, bag
	}

	suffix := "-" + rulesVersion
	files := map[string][]byte{
		"businessid-rules" + suffix + ".binpb":          rules.Bytes,
		"businessid-conformance" + suffix + ".binpb":    suite.Bytes,
		"businessid-conformance" + suffix + ".jsonl.gz": jsonlGz,
		"businessid-manifest" + suffix + ".json":        manifestBytes,
		"rules.proto":                                   inputs.rulesProto,
		"conformance.proto":                             inputs.conformanceProto,
		"ir.md":                                         irDoc,
		"features.md":                                   featuresDoc,
		"coverage.md":                                   coverageDoc,
		"SBOM.spdx.json":                                sbom,
		// The minimal reference bundle and its suite are published with the
		// documents so that an engine can start before any official rule
		// exists (spec.md section 14).
		"reference-bundle.binpb":      inputs.referenceBundle,
		"reference-conformance.binpb": inputs.referenceSuite,
	}
	files["SHA256SUMS"] = renderSums(files)
	return &buildResult{
		rules:        rules,
		suite:        suite,
		manifest:     manifest,
		files:        files,
		generatedAt:  generatedAt,
		reproducible: reproducible,
	}, bag
}

// resolveRulesVersion reads and validates the business version.
func resolveRulesVersion(bag *diagnostics.Bag, opts buildOptions) (string, bool) {
	rulesVersion := opts.rulesVersion
	if rulesVersion == "" {
		raw, err := os.ReadFile(filepath.Join(opts.moduleRoot, "RULES_VERSION"))
		if err != nil {
			bag.Errorf(diagnostics.Position{}, "CLI010", "cannot read RULES_VERSION: %v", err)
			return "", false
		}
		rulesVersion = strings.TrimSpace(string(raw))
	}
	if !validRulesVersion(rulesVersion) {
		bag.Suggestf(diagnostics.Position{File: "RULES_VERSION"}, "CLI011",
			"use the YYYY.MM.PATCH form, for example 2026.08.0",
			"invalid rules version %q", rulesVersion)
		return "", false
	}
	return rulesVersion, true
}

// buildInputs are the files read from the repository, outside the compilers.
type buildInputs struct {
	rulesProto       []byte
	conformanceProto []byte
	referenceBundle  []byte
	referenceSuite   []byte
	modulePath       string
	modules          []artifact.Module
}

func readBuildInputs(bag *diagnostics.Bag, opts buildOptions) (buildInputs, bool) {
	var out buildInputs
	read := func(code, label string, parts ...string) []byte {
		data, err := os.ReadFile(filepath.Join(parts...)) //nolint:gosec // repository path supplied by the operator
		if err != nil {
			bag.Errorf(diagnostics.Position{}, code, "cannot read %s: %v", label, err)
			return nil
		}
		return data
	}
	out.rulesProto = read("CLI015", "rules.proto",
		opts.moduleRoot, "proto", "libbusinessid", "ir", "v1", "rules.proto")
	out.conformanceProto = read("CLI016", "conformance.proto",
		opts.moduleRoot, "proto", "libbusinessid", "conformance", "v1", "conformance.proto")
	out.referenceBundle = read("CLI021", "the reference bundle",
		opts.fixtures, "bundles", "minimal_valid.binpb")
	out.referenceSuite = read("CLI022", "the reference conformance suite",
		opts.fixtures, "bundles", "minimal_conformance.binpb")
	goMod := read("CLI017", "go.mod", opts.moduleRoot, "go.mod")
	if bag.HasErrors() {
		return out, false
	}
	modulePath, modules, err := artifact.ParseGoMod(string(goMod))
	if err != nil {
		bag.Errorf(diagnostics.Position{}, "CLI018", "cannot parse go.mod: %v", err)
		return out, false
	}
	out.modulePath, out.modules = modulePath, modules
	return out, true
}

// manifestInput groups everything the manifest describes.
type manifestInput struct {
	opts         buildOptions
	rulesVersion string
	generatedAt  string
	reproducible bool
	rules        *artifact.CompileResult
	suite        *conformance.CompileResult
	inputs       buildInputs
	jsonlGz      []byte
	irDoc        []byte
	featuresDoc  []byte
	coverage     artifact.CoverageInput
}

func buildManifest(in manifestInput) *artifact.Manifest {
	countries := map[string]bool{}
	for _, d := range in.rules.Bundle.GetIdentifiers() {
		if d.CountryCode != nil {
			countries[d.GetCountryCode()] = true
		}
	}
	declared := make([]artifact.Feature, 0, len(in.rules.Bundle.GetRequiredFeatureIds()))
	for _, id := range in.rules.Bundle.GetRequiredFeatureIds() {
		c, ok := features.Lookup(id)
		if !ok {
			continue
		}
		declared = append(declared, artifact.Feature{ID: c.ID, Name: c.Name})
	}
	return &artifact.Manifest{
		RulesVersion:              in.rulesVersion,
		FormatVersion:             in.rules.Bundle.GetFormatVersion(),
		RequiredFeatures:          declared,
		SourceDigest:              artifact.HexDigest(in.rules.SourceDigest),
		ConformanceSourceDigest:   artifact.HexDigest(in.suite.SourceDigest),
		ArtifactSha256:            artifact.SHA256Hex(in.rules.Bytes),
		ConformanceSha256:         artifact.SHA256Hex(in.suite.Bytes),
		ConformanceJsonlGzSha256:  artifact.SHA256Hex(in.jsonlGz),
		CompilerVersion:           version.Compiler,
		SourceCommit:              in.opts.sourceCommit,
		GeneratedAt:               in.generatedAt,
		IdentifierCount:           len(in.rules.Bundle.GetIdentifiers()),
		CountryCount:              len(countries),
		CoverageByKind:            artifact.KindCoverageOf(in.coverage),
		MinimumEngineCapabilities: in.rules.Bundle.GetRequiredFeatureIds(),
		RulesProtoSha256:          artifact.SHA256Hex(in.inputs.rulesProto),
		ConformanceProtoSha256:    artifact.SHA256Hex(in.inputs.conformanceProto),
		IrDocSha256:               artifact.SHA256Hex(in.irDoc),
		FeaturesDocSha256:         artifact.SHA256Hex(in.featuresDoc),
		Reproducible:              in.reproducible,
		ConformanceCaseCount:      len(in.suite.Bundle.GetCases()),
		ConformanceTagCounts:      artifact.TagCounts(in.suite.Bundle),
	}
}

func renderSums(files map[string][]byte) []byte {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, name := range names {
		_, _ = fmt.Fprintf(&b, "%s  %s\n", artifact.SHA256Hex(files[name]), name)
	}
	return []byte(b.String())
}

func sourceDateEpoch(release bool) (int64, bool, error) {
	raw, ok := os.LookupEnv("SOURCE_DATE_EPOCH")
	if !ok || strings.TrimSpace(raw) == "" {
		if release {
			return 0, false, errors.New("SOURCE_DATE_EPOCH is mandatory in release mode")
		}
		// A local build that cannot be reproduced is explicitly marked.
		return 0, false, nil
	}
	epoch, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("invalid SOURCE_DATE_EPOCH: %w", err)
	}
	if epoch < 0 {
		return 0, false, errors.New("SOURCE_DATE_EPOCH must not be negative")
	}
	return epoch, true, nil
}

func validRulesVersion(v string) bool {
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return false
	}
	if len(parts[0]) != 4 || len(parts[1]) != 2 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		for i := 0; i < len(part); i++ {
			if part[i] < '0' || part[i] > '9' {
				return false
			}
		}
	}
	return true
}

func runCompile(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("compile", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts buildOptions
	opts.bind(fs)
	out := fs.String("out", "dist", "directory receiving the artifacts")
	writeDocs := fs.Bool("write-docs", false, "also refresh the generated documents under docs/")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	result, bag := build(opts)
	if code := report(bag, opts.json, stdout, stderr); code != exitOK || result == nil {
		if result == nil && code == exitOK {
			return exitRejected
		}
		return code
	}
	if opts.release && !result.reproducible {
		return fail(stderr, "a release build must be reproducible")
	}
	// Nothing is written before every artifact is known: a failure never leaves
	// a partial output directory.
	names := make([]string, 0, len(result.files))
	for name := range result.files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := artifact.WriteFileAtomic(filepath.Join(*out, name), result.files[name]); err != nil {
			return fail(stderr, "cannot write %s: %v", name, err)
		}
	}
	if *writeDocs {
		for path, content := range generatedDocs(result) {
			if err := artifact.WriteFileAtomic(filepath.Join(opts.moduleRoot, path), content); err != nil {
				return fail(stderr, "cannot write %s: %v", path, err)
			}
		}
	}
	_, _ = fmt.Fprintf(stdout, "rules %s, %d identifiers, %d conformance cases, reproducible=%t\n",
		result.manifest.RulesVersion, result.manifest.IdentifierCount,
		result.manifest.ConformanceCaseCount, result.reproducible)
	return exitOK
}

// generatedDocs maps the repository documents to their generated content.
func generatedDocs(result *buildResult) map[string][]byte {
	return map[string][]byte{
		filepath.Join("docs", "ir.md"):                    result.files["ir.md"],
		filepath.Join("docs", "features.md"):              result.files["features.md"],
		filepath.Join("docs", "generated", "coverage.md"): result.files["coverage.md"],
	}
}

func runVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts buildOptions
	opts.bind(fs)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	result, bag := build(opts)
	if code := report(bag, opts.json, stdout, stderr); code != exitOK || result == nil {
		if result == nil && code == exitOK {
			return exitRejected
		}
		return code
	}
	engine := reference.NewEngineFromRuleset(result.rules.Ruleset)
	run := conformance.Run(engine, result.suite.Bundle)
	for _, f := range run.Failures {
		_, _ = fmt.Fprintln(stderr, f.String())
	}
	if len(run.Failures) > 0 {
		_, _ = fmt.Fprintf(stderr, "%d/%d conformance cases passed\n", run.Passed, run.Total)
		return exitRejected
	}
	_, _ = fmt.Fprintf(stdout, "%d conformance cases passed\n", run.Total)
	return exitOK
}
