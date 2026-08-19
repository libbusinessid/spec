package artifact_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libbusinessid/spec/internal/artifact"
)

func compileDiagnostics(t *testing.T, dir string) string {
	t.Helper()
	_, bag := artifact.CompileRules(dir, artifact.CompileOptions{RulesVersion: "2026.08.0", Optimize: true})
	var sb strings.Builder
	for _, d := range bag.Sorted() {
		sb.WriteString(d.Code + ": " + d.Message + "\n")
	}
	return sb.String()
}

func TestCompileRulesRejectsAMissingDirectory(t *testing.T) {
	out := compileDiagnostics(t, filepath.Join(t.TempDir(), "missing"))
	if !strings.Contains(out, "CLI001") {
		t.Fatalf("expected CLI001, got:\n%s", out)
	}
}

func TestCompileRulesRejectsAnEmptyDirectory(t *testing.T) {
	out := compileDiagnostics(t, t.TempDir())
	if !strings.Contains(out, "CLI002") {
		t.Fatalf("expected CLI002, got:\n%s", out)
	}
}

func TestCompileRulesReportsParseErrors(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "broken.hcl"), []byte("format \"a\" \"b\" {"), 0o600); err != nil {
		t.Fatal(err)
	}
	out := compileDiagnostics(t, dir)
	if !strings.Contains(out, "HCL001") {
		t.Fatalf("expected a syntax error, got:\n%s", out)
	}
}

func TestCompileRulesReportsLinkAndTypeErrors(t *testing.T) {
	dir := t.TempDir()
	source := `
canonicalizer "a" "b" {
  steps = [trim_whitespace()]
}

format "a" "b" {
  checks = [require(is_empty(subject()), "empty", "a.b.empty")]
}

identifier "demo" "FR" {
  canonicalizer   = canonicalizer.a.b
  format          = format.a.b
  checksum        = checksum.missing.rule
  default_profile = "compatible"
}
`
	if err := os.WriteFile(filepath.Join(dir, "rules.hcl"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	out := compileDiagnostics(t, dir)
	if !strings.Contains(out, "LINK002") {
		t.Fatalf("expected an unknown symbol, got:\n%s", out)
	}
}

func TestCompileRulesReportsATypeError(t *testing.T) {
	dir := t.TempDir()
	source := `
canonicalizer "a" "b" {
  steps = [luhn(value())]
}
`
	if err := os.WriteFile(filepath.Join(dir, "rules.hcl"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	out := compileDiagnostics(t, dir)
	if !strings.Contains(out, "TYPE") {
		t.Fatalf("expected a type error, got:\n%s", out)
	}
}

func TestCompileRulesSucceedsOnThePilot(t *testing.T) {
	result, bag := artifact.CompileRules(filepath.Join(repoRootFor(t), "rules"),
		artifact.CompileOptions{RulesVersion: "2026.08.0", Optimize: true})
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", bag.Sorted())
	}
	if result.Ruleset == nil || len(result.Bytes) == 0 || len(result.Files) == 0 {
		t.Fatal("the compilation result is incomplete")
	}
	if result.Bundle.GetSourceDigest() == nil || len(result.Bundle.GetSourceDigest()) != 32 {
		t.Fatal("the source digest must be 32 bytes")
	}
	// Compiling twice must produce the same bytes.
	again, bag := artifact.CompileRules(filepath.Join(repoRootFor(t), "rules"),
		artifact.CompileOptions{RulesVersion: "2026.08.0", Optimize: true})
	if bag.HasErrors() {
		t.Fatal("the second compilation failed")
	}
	if string(result.Bytes) != string(again.Bytes) {
		t.Fatal("two compilations of the same sources must produce the same bytes")
	}
	// The unoptimized bundle must still be accepted by the loader.
	plain, bag := artifact.CompileRules(filepath.Join(repoRootFor(t), "rules"),
		artifact.CompileOptions{RulesVersion: "2026.08.0", Optimize: false})
	if bag.HasErrors() {
		t.Fatal("the unoptimized compilation failed")
	}
	if len(plain.Bytes) < len(result.Bytes) {
		t.Fatal("deduplication must not grow the bundle")
	}
}

func TestCoverageDocumentDescribesThePilot(t *testing.T) {
	result, bag := artifact.CompileRules(filepath.Join(repoRootFor(t), "rules"),
		artifact.CompileOptions{RulesVersion: "2026.08.0", Optimize: true})
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", bag.Sorted())
	}
	doc := string(artifact.RenderCoverageDoc(artifact.CoverageInput{Ruleset: result.Ruleset}))
	for _, needle := range []string{
		"Rule coverage", "Country by kind matrix", "Dispatch tables", "Algorithms in use",
		"Rules without a published checksum", "Required capabilities", "Provenance",
		"`vat`", "`GLOBAL`", "unsupported_checksum",
	} {
		if !strings.Contains(doc, needle) {
			t.Fatalf("the coverage document does not mention %q", needle)
		}
	}
	if strings.Contains(doc, "Conformance statistics") {
		t.Fatal("without a suite the document must not claim statistics")
	}
	coverage := artifact.KindCoverageOf(artifact.CoverageInput{Ruleset: result.Ruleset})
	if len(coverage) == 0 {
		t.Fatal("the coverage table is empty")
	}
	for _, entry := range coverage {
		if entry.Kind == "vat" && entry.WithoutChecksum != 1 {
			t.Fatalf("the vat kind must hold exactly one definition without checksum: %+v", entry)
		}
	}
}
