package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/libbusinessid/spec/internal/artifact"
	"github.com/libbusinessid/spec/internal/diagnostics"
)

// runCheckGenerated rebuilds every artifact in two distinct temporary
// directories and compares the produced bytes with the committed documents.
// It proves both reproducibility and the freshness of the generated files.
func runCheckGenerated(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("check-generated", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts buildOptions
	opts.bind(fs)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if _, ok := os.LookupEnv("SOURCE_DATE_EPOCH"); !ok {
		return fail(stderr, "SOURCE_DATE_EPOCH is mandatory for check-generated")
	}
	first, bag := build(opts)
	if code := reportBuild(bag, first == nil, opts.json, stdout, stderr); code != exitOK {
		return code
	}
	second, secondBag := build(opts)
	if code := reportBuild(secondBag, second == nil, opts.json, stdout, stderr); code != exitOK {
		return code
	}

	problems := diagnostics.New()
	names := sortedNames(first.files)
	compareInMemory(problems, names, first, second)
	if code := compareOnDisk(problems, names, first, second, stderr); code != exitOK {
		return code
	}
	compareCommittedDocuments(problems, opts, first)
	if first.manifest.IrDocSha256 != artifact.SHA256Hex(first.files["ir.md"]) ||
		first.manifest.FeaturesDocSha256 != artifact.SHA256Hex(first.files["features.md"]) {
		problems.Errorf(diagnostics.Position{File: "manifest"}, "GEN005",
			"the manifest does not describe the generated documents")
	}
	if problems.HasErrors() {
		return report(problems, opts.json, stdout, stderr)
	}
	_, _ = fmt.Fprintf(stdout, "check-generated: %d artifacts are reproducible and up to date\n", len(names))
	return exitOK
}

// reportBuild renders the diagnostics of one build and maps them to an exit
// code, treating a nil result without diagnostics as a rejection.
func reportBuild(bag *diagnostics.Bag, missing, asJSON bool, stdout, stderr io.Writer) int {
	code := report(bag, asJSON, stdout, stderr)
	if code != exitOK {
		return code
	}
	if missing {
		return exitRejected
	}
	return exitOK
}

func sortedNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// compareInMemory checks that two builds of the same sources produced the same
// bytes.
func compareInMemory(problems *diagnostics.Bag, names []string, first, second *buildResult) {
	for _, name := range names {
		if !bytes.Equal(first.files[name], second.files[name]) {
			problems.Errorf(diagnostics.Position{File: name}, "GEN001",
				"two builds of the same sources produced different bytes")
		}
	}
}

// compareOnDisk writes both builds into two distinct temporary directories and
// compares them, which also exercises the atomic writer.
func compareOnDisk(problems *diagnostics.Bag, names []string, first, second *buildResult, stderr io.Writer) int {
	dirs := make([]string, 2)
	for i := range dirs {
		dir, err := os.MkdirTemp("", fmt.Sprintf("businessidc-check-%d-", i))
		if err != nil {
			return fail(stderr, "cannot create a temporary directory: %v", err)
		}
		defer func(p string) { _ = os.RemoveAll(p) }(dir)
		dirs[i] = dir
	}
	for i, result := range []*buildResult{first, second} {
		for _, name := range names {
			if err := os.WriteFile(filepath.Join(dirs[i], name), result.files[name], 0o600); err != nil {
				return fail(stderr, "cannot write the temporary artifact: %v", err)
			}
		}
	}
	for _, name := range names {
		left, err := os.ReadFile(filepath.Join(dirs[0], name)) //nolint:gosec // temporary directory
		if err != nil {
			return fail(stderr, "cannot read the temporary artifact: %v", err)
		}
		right, err := os.ReadFile(filepath.Join(dirs[1], name)) //nolint:gosec // temporary directory
		if err != nil {
			return fail(stderr, "cannot read the temporary artifact: %v", err)
		}
		if !bytes.Equal(left, right) {
			problems.Errorf(diagnostics.Position{File: name}, "GEN002",
				"the artifact differs between two build directories")
		}
	}
	return exitOK
}

// compareCommittedDocuments checks the generated documents of the repository.
func compareCommittedDocuments(problems *diagnostics.Bag, opts buildOptions, result *buildResult) {
	for path, want := range generatedDocs(result) {
		got, err := os.ReadFile(filepath.Join(opts.moduleRoot, path)) //nolint:gosec // repository path
		if err != nil {
			problems.Suggestf(diagnostics.Position{File: path}, "GEN003",
				"run `businessidc compile --write-docs`", "the generated document is missing: %v", err)
			continue
		}
		if !bytes.Equal(got, want) {
			problems.Suggestf(diagnostics.Position{File: path}, "GEN004",
				"run `businessidc compile --write-docs`", "the generated document is stale")
		}
	}
}
