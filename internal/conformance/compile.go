package conformance

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	conformancev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/conformance/v1"
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/artifact"
	"github.com/libbusinessid/spec/internal/diagnostics"
	"github.com/libbusinessid/spec/internal/hcllang"
	"github.com/libbusinessid/spec/internal/limits"
)

// CompileOptions controls a conformance compilation.
type CompileOptions struct {
	// RulesVersion must match the rule bundle exactly.
	RulesVersion string
	// FormatVersion must match the rule bundle exactly.
	FormatVersion uint32
	// FixtureRoot is the directory holding the `load_ruleset` fixtures.
	FixtureRoot string
}

// CompileResult holds every artifact produced by a conformance compilation.
type CompileResult struct {
	Cases        []*Case
	Bundle       *conformancev1.ConformanceBundle
	Bytes        []byte
	SourceDigest [32]byte
	Files        []hcllang.SourceFile
	// Fixtures lists the embedded fixture paths, sorted.
	Fixtures []string
}

// Compile reads, validates and compiles the whole JSONL corpus.
func Compile(casesDir string, opts CompileOptions) (*CompileResult, *diagnostics.Bag) {
	bag := diagnostics.New()
	files, err := hcllang.DiscoverExt(casesDir, ".jsonl")
	if err != nil {
		bag.Errorf(diagnostics.Position{File: casesDir}, CodeJSON, "cannot read the corpus directory: %v", err)
		return nil, bag
	}
	if len(files) == 0 {
		bag.Errorf(diagnostics.Position{File: casesDir}, CodeJSON, "no conformance file found")
		return nil, bag
	}
	var all []*Case
	entries := make([]artifact.SourceEntry, 0, len(files))
	for _, f := range files {
		data, readErr := os.ReadFile(f.AbsPath)
		if readErr != nil {
			bag.Errorf(diagnostics.Position{File: f.RelPath}, CodeJSON, "cannot read the corpus: %v", readErr)
			continue
		}
		entries = append(entries, artifact.SourceEntry{Path: "conformance/" + f.RelPath, Content: data})
		cases, readBag := Read(f.RelPath, data)
		bag.Extend(readBag)
		all = append(all, cases...)
	}
	bag.Extend(Validate(all))
	if bag.HasErrors() {
		return nil, bag
	}
	SortCases(all)

	fixtures := map[string][]byte{}
	for _, c := range all {
		if c.Fixture == nil {
			continue
		}
		data, ok := readFixture(bag, opts.FixtureRoot, c)
		if !ok {
			continue
		}
		fixtures[*c.Fixture] = data
	}
	if bag.HasErrors() {
		return nil, bag
	}
	fixturePaths := make([]string, 0, len(fixtures))
	for path := range fixtures {
		fixturePaths = append(fixturePaths, path)
		entries = append(entries, artifact.SourceEntry{Path: "fixtures/" + path, Content: fixtures[path], Binary: true})
	}
	sort.Strings(fixturePaths)

	digest, err := artifact.SourceDigest(artifact.ConformanceDomain, entries)
	if err != nil {
		bag.Errorf(diagnostics.Position{}, CodeJSON, "cannot compute the conformance digest: %v", err)
		return nil, bag
	}

	bundle := &conformancev1.ConformanceBundle{
		FormatVersion: opts.FormatVersion,
		RulesVersion:  opts.RulesVersion,
		SourceDigest:  digest[:],
	}
	for _, c := range all {
		pb, ok := toProto(bag, c, fixtures, opts)
		if !ok {
			continue
		}
		bundle.Cases = append(bundle.Cases, pb)
	}
	if bag.HasErrors() {
		return nil, bag
	}
	raw, err := artifact.Marshal(bundle)
	if err != nil {
		bag.Errorf(diagnostics.Position{}, CodeJSON, "cannot serialize the conformance bundle: %v", err)
		return nil, bag
	}
	if len(raw) > limits.MaxConformanceBundleBytes {
		bag.Errorf(diagnostics.Position{}, CodeLimit,
			"the conformance bundle holds %d bytes, the limit is %d", len(raw), limits.MaxConformanceBundleBytes)
		return nil, bag
	}
	return &CompileResult{
		Cases:        all,
		Bundle:       bundle,
		Bytes:        raw,
		SourceDigest: digest,
		Files:        files,
		Fixtures:     fixturePaths,
	}, bag
}

func readFixture(bag *diagnostics.Bag, root string, c *Case) ([]byte, bool) {
	pos := diagnostics.Position{File: c.File, Line: c.Line}
	rel := *c.Fixture
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "/") {
		bag.Errorf(pos, CodeFixture, "a fixture path must be relative to the fixture root")
		return nil, false
	}
	clean := filepath.ToSlash(filepath.Clean(rel))
	if clean != rel || strings.HasPrefix(clean, "..") {
		bag.Errorf(pos, CodeFixture, "a fixture path must stay inside the fixture root")
		return nil, false
	}
	full := filepath.Join(root, filepath.FromSlash(clean))
	info, err := os.Lstat(full)
	if err != nil {
		bag.Errorf(pos, CodeFixture, "cannot read the fixture %q: %v", rel, err)
		return nil, false
	}
	if !info.Mode().IsRegular() {
		bag.Errorf(pos, CodeFixture, "the fixture %q is not a regular file", rel)
		return nil, false
	}
	data, err := os.ReadFile(full) //nolint:gosec // the path was validated above
	if err != nil {
		bag.Errorf(pos, CodeFixture, "cannot read the fixture %q: %v", rel, err)
		return nil, false
	}
	return data, true
}

//nolint:funlen // one flat, mechanical translation of the reviewed case.
func toProto(bag *diagnostics.Bag, c *Case, fixtures map[string][]byte,
	opts CompileOptions,
) (*conformancev1.ConformanceCase, bool) {
	pos := diagnostics.Position{File: c.File, Line: c.Line}
	out := &conformancev1.ConformanceCase{
		Id:                  c.ID,
		Description:         c.Description,
		Tags:                append([]string(nil), c.Tags...),
		SourceIds:           append([]string(nil), c.SourceIDs...),
		Generated:           c.Generated,
		DataClassification:  c.DataClassification,
		RedistributionBasis: c.RedistributionBasis,
	}
	switch c.Operation {
	case OpCanonicalize:
		out.Operation = conformancev1.Operation_OPERATION_CANONICALIZE
	case OpValidateFormat:
		out.Operation = conformancev1.Operation_OPERATION_VALIDATE_FORMAT
	case OpValidateChecksum:
		out.Operation = conformancev1.Operation_OPERATION_VALIDATE_CHECKSUM
	case OpValidate:
		out.Operation = conformancev1.Operation_OPERATION_VALIDATE
	case OpLoadRuleset:
		out.Operation = conformancev1.Operation_OPERATION_LOAD_RULESET
		payload := fixtures[*c.Fixture]
		out.RulesPayload = payload
		expected := *c.ExpectedEngineError
		out.ExpectedEngineError = &expected
		return out, true
	default:
		return nil, false
	}

	out.Kind = *c.Kind
	out.Input = *c.Input
	out.Profile = *c.Profile
	if c.CountryCode != nil {
		country := *c.CountryCode
		out.CountryCode = &country
	}
	expectedKind := out.GetKind()
	if c.Expected.Kind != nil {
		expectedKind = *c.Expected.Kind
	}
	if c.Operation == OpCanonicalize {
		status, okStatus := statusValue(c.Expected.Status)
		reason, okReason := reasonValue(c.Expected.ReasonCode)
		if !okStatus || !okReason {
			bag.Errorf(pos, CodeBadValue, "unknown status or reason code")
			return nil, false
		}
		expected := &conformancev1.ExpectedCanonicalization{
			Kind:           expectedKind,
			InputValue:     out.GetInput(),
			CanonicalValue: *c.Expected.CanonicalValue,
			Profile:        out.GetProfile(),
			RulesVersion:   opts.RulesVersion,
			FormatVersion:  opts.FormatVersion,
			Status:         status,
			ReasonCode:     reason,
			MessageKey:     c.Expected.MessageKey,
		}
		if c.Expected.CountryCode != nil {
			country := *c.Expected.CountryCode
			expected.CountryCode = &country
		}
		out.Expected = &conformancev1.ExpectedOutcome{
			Value: &conformancev1.ExpectedOutcome_Canonicalization{Canonicalization: expected},
		}
		return out, true
	}

	formatStep, okFormat := stepValue(c.Expected.Format, conformancev1.ValidationLevel_VALIDATION_LEVEL_FORMAT)
	checksumStep, okChecksum := stepValue(c.Expected.Checksum, conformancev1.ValidationLevel_VALIDATION_LEVEL_CHECKSUM)
	if !okFormat || !okChecksum {
		bag.Errorf(pos, CodeBadValue, "unknown status or reason code in a step")
		return nil, false
	}
	expected := &conformancev1.ExpectedValidationReport{
		Kind:           expectedKind,
		InputValue:     out.GetInput(),
		CanonicalValue: *c.Expected.CanonicalValue,
		Profile:        out.GetProfile(),
		RulesVersion:   opts.RulesVersion,
		FormatVersion:  opts.FormatVersion,
		Format:         formatStep,
		Checksum:       checksumStep,
	}
	if c.Expected.CountryCode != nil {
		country := *c.Expected.CountryCode
		expected.CountryCode = &country
	}
	out.Expected = &conformancev1.ExpectedOutcome{
		Value: &conformancev1.ExpectedOutcome_ValidationReport{ValidationReport: expected},
	}
	return out, true
}

func statusValue(name *string) (conformancev1.StepStatus, bool) {
	if name == nil {
		return conformancev1.StepStatus_STEP_STATUS_UNSPECIFIED, false
	}
	v, ok := conformancev1.StepStatus_value["STEP_STATUS_"+strings.ToUpper(*name)]
	if !ok || v == 0 {
		return conformancev1.StepStatus_STEP_STATUS_UNSPECIFIED, false
	}
	return conformancev1.StepStatus(v), true
}

func reasonValue(name *string) (irv1.ReasonCode, bool) {
	if name == nil {
		return irv1.ReasonCode_REASON_CODE_UNSPECIFIED, false
	}
	v, ok := irv1.ReasonCode_value["REASON_CODE_"+strings.ToUpper(*name)]
	if !ok || v == 0 {
		return irv1.ReasonCode_REASON_CODE_UNSPECIFIED, false
	}
	return irv1.ReasonCode(v), true
}

func stepValue(s *Step, level conformancev1.ValidationLevel) (*conformancev1.ExpectedStep, bool) {
	if s == nil {
		return nil, false
	}
	status, okStatus := statusValue(&s.Status)
	reason, okReason := reasonValue(&s.ReasonCode)
	if !okStatus || !okReason {
		return nil, false
	}
	return &conformancev1.ExpectedStep{
		Level:      level,
		Status:     status,
		ReasonCode: reason,
		MessageKey: s.MessageKey,
	}, true
}
