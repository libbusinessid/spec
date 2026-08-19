package main

import (
	"os"
	osexec "os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// copyRepo clones the working tree into a temporary directory so that a test
// can corrupt a generated document without touching the repository.
func copyRepo(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	cmd := osexec.CommandContext(t.Context(), "cp", "-a", repoRoot(t)+"/.", dst)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("cannot copy the repository: %v\n%s", err, out)
	}
	return dst
}

func inCopy(t *testing.T, root string, args ...string) (int, string, string) {
	t.Helper()
	full := append([]string{args[0],
		"--module-root", root,
		"--rules", filepath.Join(root, "rules"),
		"--cases", filepath.Join(root, "conformance"),
		"--fixtures", filepath.Join(root, "testdata"),
	}, args[1:]...)
	return exec(t, full...)
}

func TestCheckGeneratedDetectsAStaleDocument(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	root := copyRepo(t)
	target := filepath.Join(root, "docs", "ir.md")
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if code, _, _ := inCopy(t, root, "check-generated"); code != exitOK {
		t.Fatal("the copy must start clean")
	}
	if err := os.WriteFile(target, append(original, []byte("\nstale\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := inCopy(t, root, "check-generated")
	if code != exitRejected || !strings.Contains(stdout+stderr, "GEN004") {
		t.Fatalf("expected a stale document, got code=%d\n%s%s", code, stdout, stderr)
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = inCopy(t, root, "check-generated")
	if code != exitRejected || !strings.Contains(stdout+stderr, "GEN003") {
		t.Fatalf("expected a missing document, got code=%d\n%s%s", code, stdout, stderr)
	}
}

func TestCheckGeneratedRejectsBrokenSources(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	root := copyRepo(t)
	if err := os.WriteFile(filepath.Join(root, "rules", "broken.hcl"), []byte("format \"a\" \"b\" {"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := inCopy(t, root, "check-generated")
	if code != exitRejected {
		t.Fatalf("expected a rejection, got code=%d\n%s%s", code, stdout, stderr)
	}
}

func TestCompileWritesDocsInACopy(t *testing.T) {
	t.Setenv("SOURCE_DATE_EPOCH", "1755475200")
	root := copyRepo(t)
	if err := os.RemoveAll(filepath.Join(root, "docs", "generated")); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := inCopy(t, root, "compile", "--out", filepath.Join(root, "dist"), "--write-docs")
	if code != exitOK {
		t.Fatalf("compile failed: %s%s", stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "docs", "generated", "coverage.md")); err != nil {
		t.Fatalf("the generated document was not rewritten: %v", err)
	}
	if code, _, _ := inCopy(t, root, "check-generated"); code != exitOK {
		t.Fatal("the rewritten documents must be up to date")
	}
}

func TestLintReportsAnUnknownSourceReference(t *testing.T) {
	root := copyRepo(t)
	path := filepath.Join(root, "conformance", "global", "extra.jsonl")
	line := `{"id":"zz-unknown-source-001","kind":"siren","input":"012345674","profile":"compatible",` +
		`"operation":"validate","expected":{"canonicalValue":"012345674","countryCode":"FR",` +
		`"format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"}},` +
		`"tags":["x"],"sourceIds":["does-not-exist"],"dataClassification":"synthetic","redistributionBasis":"b"}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := inCopy(t, root, "lint")
	if code != exitRejected || !strings.Contains(stdout+stderr, "LINT004") {
		t.Fatalf("expected an unknown source diagnostic, got code=%d\n%s%s", code, stdout, stderr)
	}
}

func TestLintReportsUnformattedSources(t *testing.T) {
	root := copyRepo(t)
	target := filepath.Join(root, "rules", "national", "fr_siren.hcl")
	original, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	messy := strings.Replace(string(original), "  steps = [", "        steps = [", 1)
	if err := os.WriteFile(target, []byte(messy), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := inCopy(t, root, "lint")
	if code != exitRejected || !strings.Contains(stdout+stderr, "LINT003") {
		t.Fatalf("expected a formatting diagnostic, got code=%d\n%s%s", code, stdout, stderr)
	}
}

func TestVerifyDetectsABrokenExpectation(t *testing.T) {
	root := copyRepo(t)
	path := filepath.Join(root, "conformance", "global", "extra.jsonl")
	line := `{"id":"zz-wrong-expectation-001","kind":"siren","input":"012345674","profile":"compatible",` +
		`"operation":"validate","expected":{"canonicalValue":"999999999","countryCode":"FR",` +
		`"format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"}},` +
		`"tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := inCopy(t, root, "verify")
	if code != exitRejected || !strings.Contains(stdout+stderr, "canonicalValue") {
		t.Fatalf("expected a conformance failure, got code=%d\n%s%s", code, stdout, stderr)
	}
}
