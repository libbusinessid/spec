package conformance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libbusinessid/spec/internal/conformance"
)

func diagnosticCodes(t *testing.T, jsonl string) string {
	t.Helper()
	cases, bag := conformance.Read("cases.jsonl", []byte(jsonl))
	bag.Extend(conformance.Validate(cases))
	var sb strings.Builder
	for _, d := range bag.Sorted() {
		sb.WriteString(d.Code + ": " + d.Message + " | " + d.Suggestion + "\n")
	}
	return sb.String()
}

const validCase = `{"id":"a-001","description":"d","kind":"vat","countryCode":"BE","input":"BE0123456749","profile":"compatible","operation":"validate","expected":{"canonicalValue":"BE0123456749","countryCode":"BE","format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"}},"tags":["a","b"],"sourceIds":["s1"],"dataClassification":"synthetic","redistributionBasis":"basis"}`

func TestReadAcceptsAValidCase(t *testing.T) {
	if out := diagnosticCodes(t, validCase+"\n"); out != "" {
		t.Fatalf("unexpected diagnostics:\n%s", out)
	}
}

func TestReadAcceptsBlankLines(t *testing.T) {
	if out := diagnosticCodes(t, "\n"+validCase+"\n\n"); out != "" {
		t.Fatalf("unexpected diagnostics:\n%s", out)
	}
}

func TestReadAcceptsAGeneratedFlag(t *testing.T) {
	line := strings.Replace(validCase, `"tags"`, `"generated":true,"tags"`, 1)
	if out := diagnosticCodes(t, line); out != "" {
		t.Fatalf("unexpected diagnostics:\n%s", out)
	}
}

func TestCorpusRejections(t *testing.T) {
	tests := []struct {
		name string
		body string
		code string
		msg  string
	}{
		{"invalid utf8", "\xff\xfe", conformance.CodeJSON, "not valid UTF-8"},
		{"comment", "# a comment", conformance.CodeJSON, "no comment"},
		{"slash comment", "// a comment", conformance.CodeJSON, "no comment"},
		{"invalid json", "{", conformance.CodeJSON, "invalid case"},
		{"unknown field", `{"id":"a","operation":"validate","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b","nope":1}`,
			conformance.CodeJSON, "unknown field"},
		{"two objects", `{"id":"a"} {"id":"b"}`, conformance.CodeJSON, "exactly one JSON object"},
		{"missing id", `{"operation":"validate","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeMissingField, "non empty id"},
		{"duplicate id", validCase + "\n" + validCase, conformance.CodeDuplicateID, "duplicate case id"},
		{"missing operation", `{"id":"a","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeMissingField, "must declare an operation"},
		{"unknown operation", `{"id":"a","operation":"nope","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeBadValue, "unknown operation"},
		{"no tag", `{"id":"a","operation":"validate","tags":[],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeMissingField, "at least one tag"},
		{"empty tag", `{"id":"a","operation":"validate","tags":[""],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeBadValue, "tag must not be empty"},
		{"unsorted tags", strings.Replace(validCase, `["a","b"]`, `["b","a"]`, 1),
			conformance.CodeBadValue, "sorted and unique"},
		{"duplicate tags", strings.Replace(validCase, `["a","b"]`, `["a","a"]`, 1),
			conformance.CodeBadValue, "sorted and unique"},
		{"missing classification", strings.Replace(validCase, `"dataClassification":"synthetic",`, "", 1),
			conformance.CodeMissingField, "dataClassification"},
		{"unknown classification", strings.Replace(validCase, `"synthetic"`, `"other"`, 1),
			conformance.CodeDataPolicy, "unknown dataClassification"},
		{"empty basis", strings.Replace(validCase, `"redistributionBasis":"basis"`, `"redistributionBasis":"  "`, 1),
			conformance.CodeMissingField, "redistributionBasis"},
		{"forbidden phrase", strings.Replace(validCase, `"basis"`, `"a production case"`, 1),
			conformance.CodeDataPolicy, "production case"},
		{"official without source", strings.Replace(strings.Replace(validCase, `"synthetic"`, `"official_public_example"`, 1), `"sourceIds":["s1"],`, "", 1),
			conformance.CodeDataPolicy, "at least one source id"},
		{"empty source id", strings.Replace(validCase, `["s1"]`, `[""]`, 1),
			conformance.CodeBadValue, "source id must not be empty"},
		{"unsorted source ids", strings.Replace(validCase, `["s1"]`, `["s2","s1"]`, 1),
			conformance.CodeBadValue, "sorted and unique"},
		{"long description", `{"id":"a","description":"` + strings.Repeat("x", 5000) + `","operation":"validate","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeLimit, "exceeds"},
		{"missing kind", `{"id":"a","input":"x","profile":"compatible","operation":"validate","expected":{"canonicalValue":"x","format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"}},"tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeMissingField, "requires kind"},
		{"missing input", strings.Replace(validCase, `"input":"BE0123456749",`, "", 1),
			conformance.CodeMissingField, "requires input"},
		{"missing profile", strings.Replace(validCase, `"profile":"compatible",`, "", 1),
			conformance.CodeMissingField, "requires profile"},
		{"unknown profile", strings.Replace(validCase, `"profile":"compatible"`, `"profile":"loose"`, 1),
			conformance.CodeBadValue, "unknown profile"},
		{"missing expectation", strings.Replace(validCase, `"expected":{"canonicalValue":"BE0123456749","countryCode":"BE","format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"}},`, "", 1),
			conformance.CodeMissingField, "requires expected"},
		{"missing canonical value", strings.Replace(validCase, `"canonicalValue":"BE0123456749",`, "", 1),
			conformance.CodeMissingField, "canonicalValue"},
		{"missing steps", strings.Replace(validCase, `"format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"}`, `"kind":"vat"`, 1),
			conformance.CodeMissingField, "expected.format and expected.checksum"},
		{"status on a validation case", strings.Replace(validCase, `"canonicalValue"`, `"status":"valid","reasonCode":"ok","canonicalValue"`, 1),
			conformance.CodeForbiddenField, "not a top level status"},
		{"bad status", strings.Replace(validCase, `"status":"valid","reasonCode":"ok"},"checksum"`, `"status":"weird","reasonCode":"ok"},"checksum"`, 1),
			conformance.CodeBadValue, "unknown status"},
		{"impossible pairing", strings.Replace(validCase, `"format":{"status":"valid","reasonCode":"ok"}`, `"format":{"status":"valid","reasonCode":"empty"}`, 1),
			conformance.CodeBadValue, "cannot carry the reason code"},
		{"empty message key", strings.Replace(validCase, `"format":{"status":"valid","reasonCode":"ok"}`, `"format":{"status":"valid","reasonCode":"ok","messageKey":""}`, 1),
			conformance.CodeBadValue, "must not be empty"},
		{"validate_format after a valid format", strings.Replace(validCase, `"operation":"validate"`, `"operation":"validate_format"`, 1),
			conformance.CodeBadValue, "not_run/not_requested"},
		{"canonicalize with steps", strings.Replace(validCase, `"operation":"validate"`, `"operation":"canonicalize"`, 1),
			conformance.CodeForbiddenField, "not format or checksum steps"},
		{"canonicalize without status", `{"id":"a","kind":"vat","input":"x","profile":"compatible","operation":"canonicalize","expected":{"canonicalValue":"x"},"tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeMissingField, "expected.status"},
		{"load_ruleset with kind", `{"id":"a","kind":"vat","operation":"load_ruleset","fixture":"f","expectedEngineError":"invalid_ruleset","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeForbiddenField, "forbids kind"},
		{"load_ruleset with country", `{"id":"a","countryCode":"BE","operation":"load_ruleset","fixture":"f","expectedEngineError":"invalid_ruleset","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeForbiddenField, "forbids countryCode"},
		{"load_ruleset without fixture", `{"id":"a","operation":"load_ruleset","expectedEngineError":"invalid_ruleset","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeMissingField, "fixture path"},
		{"load_ruleset without error", `{"id":"a","operation":"load_ruleset","fixture":"f","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeMissingField, "expectedEngineError"},
		{"load_ruleset unknown error", `{"id":"a","operation":"load_ruleset","fixture":"f","expectedEngineError":"boom","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
			conformance.CodeBadValue, "unknown expectedEngineError"},
		{"business case with fixture", strings.Replace(validCase, `"tags"`, `"fixture":"f","tags"`, 1),
			conformance.CodeForbiddenField, "load_ruleset cases only"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := diagnosticCodes(t, tc.body)
			if !strings.Contains(out, tc.code) {
				t.Fatalf("expected %s, got:\n%s", tc.code, out)
			}
			if tc.msg != "" && !strings.Contains(out, tc.msg) {
				t.Fatalf("expected %q, got:\n%s", tc.msg, out)
			}
		})
	}
}

func compileCorpus(t *testing.T, jsonl string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cases.jsonl"), []byte(jsonl), 0o600); err != nil {
		t.Fatal(err)
	}
	_, bag := conformance.Compile(dir, conformance.CompileOptions{
		RulesVersion: "2026.08.0", FormatVersion: 1, FixtureRoot: filepath.Join(repoRoot(t), "testdata"),
	})
	var sb strings.Builder
	for _, d := range bag.Sorted() {
		sb.WriteString(d.Code + ": " + d.Message + "\n")
	}
	return sb.String()
}

func TestCompileRejectsBadFixturePaths(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
		msg     string
	}{
		{"absolute", "/etc/passwd", "relative to the fixture root"},
		{"escape", "../../etc/passwd", "inside the fixture root"},
		{"missing", "bundles/nope.binpb", "cannot read the fixture"},
		{"directory", "bundles", "not a regular file"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			body := `{"id":"a","operation":"load_ruleset","fixture":"` + tc.fixture +
				`","expectedEngineError":"invalid_ruleset","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`
			out := compileCorpus(t, body)
			if !strings.Contains(out, conformance.CodeFixture) || !strings.Contains(out, tc.msg) {
				t.Fatalf("expected %q, got:\n%s", tc.msg, out)
			}
		})
	}
}

func TestCompileRejectsAnEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	_, bag := conformance.Compile(dir, conformance.CompileOptions{RulesVersion: "2026.08.0", FormatVersion: 1})
	if !bag.HasErrors() {
		t.Fatal("an empty corpus directory must be refused")
	}
	_, bag = conformance.Compile(filepath.Join(dir, "missing"), conformance.CompileOptions{})
	if !bag.HasErrors() {
		t.Fatal("a missing corpus directory must be refused")
	}
}

func TestCompileEmbedsFixtures(t *testing.T) {
	dir := t.TempDir()
	body := `{"id":"a","operation":"load_ruleset","fixture":"bundles/truncated.binpb","expectedEngineError":"invalid_ruleset","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`
	if err := os.WriteFile(filepath.Join(dir, "cases.jsonl"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	result, bag := conformance.Compile(dir, conformance.CompileOptions{
		RulesVersion: "2026.08.0", FormatVersion: 1, FixtureRoot: filepath.Join(repoRoot(t), "testdata"),
	})
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", bag.Sorted())
	}
	if len(result.Bundle.GetCases()) != 1 || len(result.Bundle.GetCases()[0].GetRulesPayload()) == 0 {
		t.Fatal("the fixture bytes must be embedded")
	}
	if len(result.Fixtures) != 1 || result.Fixtures[0] != "bundles/truncated.binpb" {
		t.Fatalf("unexpected fixtures %v", result.Fixtures)
	}
}

func TestCanonicalJSONLRoundTrip(t *testing.T) {
	corpus := validCase + "\n" +
		`{"id":"b-001","operation":"load_ruleset","tags":["x"],"fixture":"bundles/truncated.binpb","expectedEngineError":"invalid_ruleset","generated":true,"dataClassification":"synthetic","redistributionBasis":"b"}` + "\n" +
		`{"id":"c-001","kind":"vat","input":"x","profile":"compatible","operation":"canonicalize","expected":{"kind":"vat","canonicalValue":"X","status":"valid","reasonCode":"ok","messageKey":"k"},"tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}` + "\n"
	cases, bag := conformance.Read("cases.jsonl", []byte(corpus))
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics %v", bag.Sorted())
	}
	rendered, err := conformance.WriteCanonicalJSONL(cases)
	if err != nil {
		t.Fatal(err)
	}
	again, bag := conformance.Read("cases.jsonl", rendered)
	if bag.HasErrors() {
		t.Fatalf("the canonical form must re-read cleanly: %v", bag.Sorted())
	}
	second, err := conformance.WriteCanonicalJSONL(again)
	if err != nil {
		t.Fatal(err)
	}
	if string(rendered) != string(second) {
		t.Fatal("the canonical form must be a fixed point")
	}
	lines := strings.Split(strings.TrimRight(string(rendered), "\n"), "\n")
	if len(lines) != 3 || !strings.HasPrefix(lines[0], `{"id":"a-001"`) {
		t.Fatalf("unexpected canonical output:\n%s", rendered)
	}
	if !strings.Contains(lines[1], `"generated":true`) {
		t.Fatalf("the generated flag must survive: %s", lines[1])
	}
}

func TestCanonicalLine(t *testing.T) {
	cases, _ := conformance.Read("cases.jsonl", []byte(validCase))
	line, err := conformance.CanonicalLine(cases[0])
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(line), `{"id":"a-001","description":"d","kind":"vat","countryCode":"BE"`) {
		t.Fatalf("unexpected key order: %s", line)
	}
}

func TestCompileReportsAnUnreadableFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root: permissions are not enforced")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "cases.jsonl")
	if err := os.WriteFile(path, []byte(validCase+"\n"), 0o000); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(path, 0o600) }()
	_, bag := conformance.Compile(dir, conformance.CompileOptions{RulesVersion: "2026.08.0", FormatVersion: 1})
	if !bag.HasErrors() {
		t.Fatal("an unreadable corpus file must be reported")
	}
}

func TestCompileProducesADeterministicBundle(t *testing.T) {
	dir := t.TempDir()
	corpus := validCase + "\n" +
		`{"id":"b-001","kind":"vat","input":"x","profile":"compatible","operation":"canonicalize",` +
		`"expected":{"kind":"vat","canonicalValue":"X","countryCode":"BE","status":"valid","reasonCode":"ok","messageKey":"k"},` +
		`"tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "cases.jsonl"), []byte(corpus), 0o600); err != nil {
		t.Fatal(err)
	}
	opts := conformance.CompileOptions{RulesVersion: "2026.08.0", FormatVersion: 1}
	first, bag := conformance.Compile(dir, opts)
	if bag.HasErrors() {
		t.Fatalf("unexpected diagnostics: %v", bag.Sorted())
	}
	second, _ := conformance.Compile(dir, opts)
	if string(first.Bytes) != string(second.Bytes) {
		t.Fatal("two compilations of the same corpus must produce the same bytes")
	}
	if first.Bundle.GetCases()[0].GetId() != "a-001" {
		t.Fatal("the cases must be sorted by id")
	}
	canonical := first.Bundle.GetCases()[1].GetExpected().GetCanonicalization()
	if canonical.GetCanonicalValue() != "X" || canonical.GetCountryCode() != "BE" ||
		canonical.GetMessageKey() != "k" || canonical.GetKind() != "vat" {
		t.Fatalf("the canonicalization expectation is incomplete: %+v", canonical)
	}
	report := first.Bundle.GetCases()[0].GetExpected().GetValidationReport()
	if report.GetCountryCode() != "BE" || report.GetRulesVersion() != "2026.08.0" {
		t.Fatalf("the validation expectation is incomplete: %+v", report)
	}
}
