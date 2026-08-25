package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libbusinessid/spec/internal/diagnostics"
)

// repoRoot locates the repository root from the test working directory.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("cannot locate the repository root")
	return ""
}

// exec runs the CLI from the repository root and captures both streams.
func exec(t *testing.T, args ...string) (int, string, string) {
	t.Helper()
	root := repoRoot(t)
	previous, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(previous) }()
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestUsage(t *testing.T) {
	code, stdout, _ := exec(t, "--help")
	if code != exitOK || !strings.Contains(stdout, "check-generated") {
		t.Fatalf("unexpected help: code=%d out=%q", code, stdout)
	}
	code, _, stderr := exec(t)
	if code != exitUsage || !strings.Contains(stderr, "Usage") {
		t.Fatalf("an empty command line must print the usage, got code=%d", code)
	}
	code, _, stderr = exec(t, "nope")
	if code != exitUsage || !strings.Contains(stderr, `unknown command "nope"`) {
		t.Fatalf("unexpected output for an unknown command: %q", stderr)
	}
}

func TestVersionCommand(t *testing.T) {
	code, stdout, _ := exec(t, "version")
	if code != exitOK || !strings.HasPrefix(stdout, "businessidc ") {
		t.Fatalf("unexpected version output: code=%d out=%q", code, stdout)
	}
	if code, _, _ := exec(t, "version", "--nope"); code != exitUsage {
		t.Fatalf("an unknown flag must be a usage error, got %d", code)
	}
}

func TestVerifyCommand(t *testing.T) {
	code, stdout, stderr := exec(t, "verify")
	if code != exitOK {
		t.Fatalf("verify failed: %s%s", stdout, stderr)
	}
	if !strings.Contains(stdout, "conformance cases passed") {
		t.Fatalf("unexpected verify output: %q", stdout)
	}
}

func TestVerifyRejectsBrokenRules(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.hcl"), []byte("format \"a\" \"b\" {"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := exec(t, "verify", "--rules", dir)
	if code != exitRejected || !strings.Contains(stderr, "HCL001") {
		t.Fatalf("expected a rejection, got code=%d stderr=%q", code, stderr)
	}
}

func TestLintCommand(t *testing.T) {
	code, stdout, stderr := exec(t, "lint")
	if code != exitOK {
		t.Fatalf("lint failed: %s%s", stdout, stderr)
	}
}

func TestFmtCheckIsClean(t *testing.T) {
	code, stdout, stderr := exec(t, "fmt", "--check")
	if code != exitOK {
		t.Fatalf("the sources are not in their canonical form:\n%s%s", stdout, stderr)
	}
}

func TestFmtRewritesACopy(t *testing.T) {
	root := repoRoot(t)
	dir := t.TempDir()
	source, err := os.ReadFile(filepath.Join(root, "rules", "national", "fr_siren.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	messy := append([]byte("\n\n"), bytes.ReplaceAll(source, []byte("  steps"), []byte("      steps"))...)
	target := filepath.Join(dir, "messy.hcl")
	if err := os.WriteFile(target, messy, 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := exec(t, "fmt", "--check", dir)
	if code != exitRejected || !strings.Contains(stdout, "messy.hcl") {
		t.Fatalf("fmt --check must report the file, got code=%d out=%q", code, stdout)
	}
	if code, _, stderr := exec(t, "fmt", dir); code != exitOK {
		t.Fatalf("fmt failed: %s", stderr)
	}
	if code, _, _ := exec(t, "fmt", "--check", dir); code != exitOK {
		t.Fatal("fmt must be idempotent")
	}
	rewritten, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(rewritten, messy) {
		t.Fatal("fmt did not rewrite the file")
	}
}

func TestFmtReportsUnreadableRoot(t *testing.T) {
	code, _, stderr := exec(t, "fmt", filepath.Join(t.TempDir(), "missing"))
	if code != exitRejected || !strings.Contains(stderr, "FMT001") {
		t.Fatalf("expected a walk error, got code=%d stderr=%q", code, stderr)
	}
}

func TestCompileInspectDiffAndCheckGenerated(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	out := t.TempDir()
	code, stdout, stderr := exec(t, "compile", "--out", out)
	if code != exitOK {
		t.Fatalf("compile failed: %s%s", stdout, stderr)
	}
	if !strings.Contains(stdout, "reproducible=true") {
		t.Fatalf("unexpected compile output: %q", stdout)
	}
	version := strings.TrimSpace(readFile(t, filepath.Join(repoRoot(t), "RULES_VERSION")))
	bundle := filepath.Join(out, "businessid-rules-"+version+".binpb")

	code, stdout, stderr = exec(t, "inspect", bundle)
	if code != exitOK || !strings.Contains(stdout, "rules version") {
		t.Fatalf("inspect failed: code=%d %s%s", code, stdout, stderr)
	}
	code, stdout, stderr = exec(t, "inspect", "--json", bundle)
	if code != exitOK {
		t.Fatalf("inspect --json failed: %s%s", stdout, stderr)
	}
	var doc inspection
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid inspection JSON: %v", err)
	}
	if doc.RulesVersion != version || len(doc.Identifiers) == 0 {
		t.Fatalf("unexpected inspection: %+v", doc)
	}

	code, stdout, stderr = exec(t, "diff", bundle, bundle)
	if code != exitOK || !strings.Contains(stdout, "semantically identical") {
		t.Fatalf("diff failed: code=%d %s%s", code, stdout, stderr)
	}
	code, stdout, stderr = exec(t, "diff", "--json", bundle, bundle)
	if code != exitOK || !strings.Contains(stdout, `"changes"`) {
		t.Fatalf("diff --json failed: code=%d %s%s", code, stdout, stderr)
	}

	code, stdout, stderr = exec(t, "check-generated")
	if code != exitOK {
		t.Fatalf("check-generated failed: %s%s", stdout, stderr)
	}
}

func TestCheckGeneratedRequiresSourceDateEpoch(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "")
	if err := os.Unsetenv("SOURCE_DATE_EPOCH"); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := exec(t, "check-generated")
	if code != exitInternal || !strings.Contains(stderr, "SOURCE_DATE_EPOCH") {
		t.Fatalf("expected a mandatory SOURCE_DATE_EPOCH, got code=%d stderr=%q", code, stderr)
	}
}

func TestCompileRejectsAReleaseWithoutEpoch(t *testing.T) {
	if err := os.Unsetenv("SOURCE_DATE_EPOCH"); err != nil {
		t.Fatal(err)
	}
	code, _, stderr := exec(t, "compile", "--out", t.TempDir(), "--release")
	if code != exitRejected || !strings.Contains(stderr, "SOURCE_DATE_EPOCH is mandatory") {
		t.Fatalf("expected a release rejection, got code=%d stderr=%q", code, stderr)
	}
}

func TestCompileRejectsAnInvalidRulesVersion(t *testing.T) {
	code, _, stderr := exec(t, "compile", "--out", t.TempDir(), "--rules-version", "2026.8.0")
	if code != exitRejected || !strings.Contains(stderr, "invalid rules version") {
		t.Fatalf("expected a version rejection, got code=%d stderr=%q", code, stderr)
	}
}

func TestCompileRejectsABadEpoch(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "not-a-number")
	code, _, stderr := exec(t, "compile", "--out", t.TempDir())
	if code != exitRejected || !strings.Contains(stderr, "invalid SOURCE_DATE_EPOCH") {
		t.Fatalf("expected an epoch rejection, got code=%d stderr=%q", code, stderr)
	}
	t.Setenv("SOURCE_DATE_EPOCH", "-1")
	code, _, stderr = exec(t, "compile", "--out", t.TempDir())
	if code != exitRejected || !strings.Contains(stderr, "must not be negative") {
		t.Fatalf("expected a negative epoch rejection, got code=%d stderr=%q", code, stderr)
	}
}

func TestJSONDiagnostics(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.hcl"), []byte("oops {"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := exec(t, "verify", "--rules", dir, "--json")
	if code != exitRejected {
		t.Fatalf("expected a rejection, got %d", code)
	}
	var doc struct {
		Diagnostics []struct {
			Code string `json:"code"`
		} `json:"diagnostics"`
	}
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("invalid diagnostics JSON: %v\n%s", err, stdout)
	}
	if len(doc.Diagnostics) == 0 {
		t.Fatal("expected at least one diagnostic")
	}
}

func TestInspectAndDiffRejectBadInputs(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.binpb")
	if err := os.WriteFile(broken, []byte{0xff, 0xff}, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := exec(t, "inspect"); code != exitUsage {
		t.Fatal("inspect requires exactly one argument")
	}
	if code, _, _ := exec(t, "inspect", filepath.Join(dir, "missing")); code != exitInternal {
		t.Fatal("a missing bundle is an internal error")
	}
	if code, _, _ := exec(t, "inspect", broken); code != exitRejected {
		t.Fatal("a broken bundle must be rejected")
	}
	if code, _, _ := exec(t, "diff", broken); code != exitUsage {
		t.Fatal("diff requires two arguments")
	}
	if code, _, _ := exec(t, "diff", broken, broken); code != exitRejected {
		t.Fatal("diff must reject a broken bundle")
	}
	if code, _, _ := exec(t, "diff", filepath.Join(dir, "missing"), broken); code != exitInternal {
		t.Fatal("a missing bundle is an internal error")
	}
}

func TestCompileRejectsAMissingRulesVersionFile(t *testing.T) {
	code, _, stderr := exec(t, "compile", "--out", t.TempDir(), "--module-root", t.TempDir())
	if code != exitRejected || !strings.Contains(stderr, "RULES_VERSION") {
		t.Fatalf("expected a missing RULES_VERSION, got code=%d stderr=%q", code, stderr)
	}
}

func TestValidRulesVersion(t *testing.T) {
	valid := []string{"2026.08.0", "2026.12.15", "1999.01.0"}
	invalid := []string{"", "2026.8.0", "2026.08", "2026.08.0.1", "abcd.08.0", "2026.08.x", "2026.08."}
	for _, v := range valid {
		if !validRulesVersion(v) {
			t.Errorf("%q must be valid", v)
		}
	}
	for _, v := range invalid {
		if validRulesVersion(v) {
			t.Errorf("%q must be invalid", v)
		}
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestCompileReportsMissingInputs(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	tests := []struct {
		name  string
		setup func(root string)
		msg   string
	}{
		{"missing rules.proto", func(root string) {}, "cannot read rules.proto"},
		{"missing conformance.proto", func(root string) {
			mkdirAll(t, filepath.Join(root, "proto", "libbusinessid", "ir", "v1"))
			writeFile(t, filepath.Join(root, "proto", "libbusinessid", "ir", "v1", "rules.proto"), "x")
		}, "cannot read conformance.proto"},
		{"missing testee.proto", func(root string) {
			mkdirAll(t, filepath.Join(root, "proto", "libbusinessid", "ir", "v1"))
			writeFile(t, filepath.Join(root, "proto", "libbusinessid", "ir", "v1", "rules.proto"), "x")
			mkdirAll(t, filepath.Join(root, "proto", "libbusinessid", "conformance", "v1"))
			writeFile(t, filepath.Join(root, "proto", "libbusinessid", "conformance", "v1", "conformance.proto"), "x")
		}, "cannot read testee.proto"},
		{"missing go.mod", func(root string) {
			mkdirAll(t, filepath.Join(root, "proto", "libbusinessid", "ir", "v1"))
			writeFile(t, filepath.Join(root, "proto", "libbusinessid", "ir", "v1", "rules.proto"), "x")
			mkdirAll(t, filepath.Join(root, "proto", "libbusinessid", "conformance", "v1"))
			writeFile(t, filepath.Join(root, "proto", "libbusinessid", "conformance", "v1", "conformance.proto"), "x")
			mkdirAll(t, filepath.Join(root, "proto", "libbusinessid", "testee", "v1"))
			writeFile(t, filepath.Join(root, "proto", "libbusinessid", "testee", "v1", "testee.proto"), "x")
		}, "cannot read go.mod"},
		{"unparseable go.mod", func(root string) {
			mkdirAll(t, filepath.Join(root, "proto", "libbusinessid", "ir", "v1"))
			writeFile(t, filepath.Join(root, "proto", "libbusinessid", "ir", "v1", "rules.proto"), "x")
			mkdirAll(t, filepath.Join(root, "proto", "libbusinessid", "conformance", "v1"))
			writeFile(t, filepath.Join(root, "proto", "libbusinessid", "conformance", "v1", "conformance.proto"), "x")
			mkdirAll(t, filepath.Join(root, "proto", "libbusinessid", "testee", "v1"))
			writeFile(t, filepath.Join(root, "proto", "libbusinessid", "testee", "v1", "testee.proto"), "x")
			writeFile(t, filepath.Join(root, "go.mod"), "go 1.24.0\n")
			// The prose contracts travel with the release since they are what an
			// engine is written against, so they are read before go.mod is
			// parsed and this case has to supply them to reach the parse.
			mkdirAll(t, filepath.Join(root, "docs", "spec"))
			for _, name := range []string{"spec.md", "engine.md", "engine-go.md",
				"engine-swift.md", "engine-kotlin.md", "engine-typescript.md"} {
				writeFile(t, filepath.Join(root, "docs", "spec", name), "x")
			}
			// The provenance note is assembled here too, so that an engine which
			// has verified a release has nothing left to clone.
			mkdirAll(t, filepath.Join(root, "docs", "spec", "provenance"))
			for _, name := range []string{"body.md", "go.md", "swift.md",
				"kotlin.md", "typescript.md"} {
				writeFile(t, filepath.Join(root, "docs", "spec", "provenance", name), "x")
			}
		}, "cannot parse go.mod"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, "RULES_VERSION"), "2026.08.0\n")
			writeFile(t, filepath.Join(root, "RULES_STABILITY"), "alpha\n")
			tc.setup(root)
			repo := repoRoot(t)
			code, _, stderr := exec(t, "compile",
				"--out", filepath.Join(root, "dist"),
				"--module-root", root,
				"--rules", filepath.Join(repo, "rules"),
				"--cases", filepath.Join(repo, "conformance"),
				"--fixtures", filepath.Join(repo, "testdata"))
			if code != exitRejected || !strings.Contains(stderr, tc.msg) {
				t.Fatalf("expected %q, got code=%d stderr=%q", tc.msg, code, stderr)
			}
		})
	}
}

func TestCompileRejectsABrokenCorpus(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	cases := t.TempDir()
	writeFile(t, filepath.Join(cases, "broken.jsonl"), "{\n")
	code, _, stderr := exec(t, "compile", "--out", t.TempDir(), "--cases", cases)
	if code != exitRejected || !strings.Contains(stderr, "CONF001") {
		t.Fatalf("expected a corpus rejection, got code=%d stderr=%q", code, stderr)
	}
}

func TestCompileRefusesToWriteIntoAFile(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	blocked := filepath.Join(t.TempDir(), "blocked")
	writeFile(t, blocked, "not a directory")
	code, _, stderr := exec(t, "compile", "--out", blocked)
	if code != exitInternal || !strings.Contains(stderr, "cannot write") {
		t.Fatalf("expected a write failure, got code=%d stderr=%q", code, stderr)
	}
}

func TestReportRendersJSONAndPropagatesWriterErrors(t *testing.T) {
	bag := diagnostics.New()
	bag.Errorf(diagnostics.Position{File: "a.hcl", Line: 1}, "E1", "boom")
	var stdout, stderr strings.Builder
	if code := report(bag, true, &stdout, &stderr); code != exitRejected {
		t.Fatalf("unexpected code %d", code)
	}
	if !strings.Contains(stdout.String(), `"diagnostics"`) {
		t.Fatalf("unexpected JSON output %q", stdout.String())
	}
	if code := report(bag, true, brokenWriter{}, &stderr); code != exitInternal {
		t.Fatal("a failing JSON writer must be an internal error")
	}
	if code := report(bag, false, &stdout, brokenWriter{}); code != exitInternal {
		t.Fatal("a failing text writer must be an internal error")
	}
	if code := report(nil, false, &stdout, &stderr); code != exitOK {
		t.Fatal("an empty bag is a success")
	}
	warnings := diagnostics.New()
	warnings.Warnf(diagnostics.Position{File: "a.hcl"}, "W1", "careful")
	if code := report(warnings, false, &stdout, &stderr); code != exitOK {
		t.Fatal("a warning is not a failure")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, errBrokenWriter }

var errBrokenWriter = &writerError{}

type writerError struct{}

func (*writerError) Error() string { return "broken writer" }

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o750); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestFmtNormalizesAJsonlCorpus(t *testing.T) {
	dir := t.TempDir()
	// Keys out of schema order and extra whitespace.
	line := `{ "operation":"validate" , "id":"a-001", "kind":"vat", "input":"x", "profile":"compatible",` +
		`"expected":{"format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"},` +
		`"canonicalValue":"X"},"tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}` + "\n"
	target := filepath.Join(dir, "cases.jsonl")
	writeFile(t, target, line)
	if code, stdout, _ := exec(t, "fmt", "--check", dir); code != exitRejected || !strings.Contains(stdout, "cases.jsonl") {
		t.Fatalf("fmt --check must report the corpus, got code=%d out=%q", code, stdout)
	}
	if code, _, stderr := exec(t, "fmt", dir); code != exitOK {
		t.Fatalf("fmt failed: %s", stderr)
	}
	rewritten := readFile(t, target)
	if !strings.HasPrefix(rewritten, `{"id":"a-001","kind":"vat"`) {
		t.Fatalf("unexpected canonical form: %s", rewritten)
	}
	if code, _, _ := exec(t, "fmt", "--check", dir); code != exitOK {
		t.Fatal("the canonical form must be a fixed point")
	}
}

func TestFmtReportsABrokenCorpus(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "cases.jsonl"), "{\n")
	code, _, stderr := exec(t, "fmt", dir)
	if code != exitRejected || !strings.Contains(stderr, "CONF001") {
		t.Fatalf("expected a corpus rejection, got code=%d stderr=%q", code, stderr)
	}
}

func TestDiffPrintsEveryChange(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	out := t.TempDir()
	if code, _, stderr := exec(t, "compile", "--out", out); code != exitOK {
		t.Fatalf("compile failed: %s", stderr)
	}
	version := strings.TrimSpace(readFile(t, filepath.Join(repoRoot(t), "RULES_VERSION")))
	current := filepath.Join(out, "businessid-rules-"+version+".binpb")
	minimal := filepath.Join(repoRoot(t), "testdata", "bundles", "minimal_valid.binpb")
	code, stdout, stderr := exec(t, "diff", minimal, current)
	if code != exitOK {
		t.Fatalf("diff failed: %s%s", stdout, stderr)
	}
	for _, class := range []string{string(ClassWidening), string(ClassRestriction), string(ClassIRFeature)} {
		if !strings.Contains(stdout, class) {
			t.Fatalf("the diff does not report %q:\n%s", class, stdout)
		}
	}
}

func TestCompileRecordsTheStabilityLevel(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	out := t.TempDir()
	if code, _, stderr := exec(t, "compile", "--out", out); code != exitOK {
		t.Fatalf("compile failed: %s", stderr)
	}
	version := strings.TrimSpace(readFile(t, filepath.Join(repoRoot(t), "RULES_VERSION")))
	declared := strings.TrimSpace(readFile(t, filepath.Join(repoRoot(t), "RULES_STABILITY")))
	var manifest struct {
		Stability string `json:"stability"`
	}
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(out, "businessid-manifest-"+version+".json"))), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Stability != declared {
		t.Fatalf("the manifest declares %q, RULES_STABILITY says %q", manifest.Stability, declared)
	}
}

func TestCompileRejectsAnUnknownStabilityLevel(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	code, _, stderr := exec(t, "compile", "--out", t.TempDir(), "--stability", "rc")
	if code != exitRejected || !strings.Contains(stderr, "invalid stability level") {
		t.Fatalf("expected a rejection, got code=%d stderr=%q", code, stderr)
	}
}

func TestCompileRejectsAMissingStabilityFile(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "RULES_VERSION"), "2026.08.0\n")
	repo := repoRoot(t)
	code, _, stderr := exec(t, "compile",
		"--out", filepath.Join(root, "dist"), "--module-root", root,
		"--rules", filepath.Join(repo, "rules"),
		"--cases", filepath.Join(repo, "conformance"),
		"--fixtures", filepath.Join(repo, "testdata"))
	if code != exitRejected || !strings.Contains(stderr, "RULES_STABILITY") {
		t.Fatalf("expected a missing RULES_STABILITY, got code=%d stderr=%q", code, stderr)
	}
}
