package conformance_test

import (
	"testing"

	"github.com/libbusinessid/spec/internal/conformance"
)

// FuzzReadJSONL feeds partial, duplicated and contradictory corpora to the
// reader. Only diagnostics are acceptable, never a panic.
func FuzzReadJSONL(f *testing.F) {
	seeds := []string{
		"",
		"{}",
		"not json",
		`{"id":"a","operation":"validate","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
		`{"id":"a","kind":"vat","input":"BE0","profile":"compatible","operation":"validate","expected":{"canonicalValue":"BE0","format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"}},"tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}` + "\n" +
			`{"id":"a","kind":"vat","input":"BE0","profile":"compatible","operation":"validate","expected":{"canonicalValue":"BE0","format":{"status":"valid","reasonCode":"ok"},"checksum":{"status":"valid","reasonCode":"ok"}},"tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
		`{"id":"a","operation":"load_ruleset","fixture":"../../etc/passwd","expectedEngineError":"invalid_ruleset","tags":["x"],"dataClassification":"synthetic","redistributionBasis":"b"}`,
		`{"id":"a","operation":"canonicalize","kind":"vat","input":"x","profile":"nope","expected":{"canonicalValue":"x","status":"weird","reasonCode":"nope"},"tags":["b","a"],"dataClassification":"other","redistributionBasis":""}`,
		"{\"id\":\"a\"} {\"id\":\"b\"}",
		"\xff\xfe",
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip("oversized input")
		}
		cases, bag := conformance.Read("fuzz.jsonl", data)
		conformance.Validate(cases)
		conformance.SortCases(cases)
		if _, err := conformance.WriteCanonicalJSONL(cases); err != nil {
			t.Fatalf("the canonical writer must never fail on parsed cases: %v", err)
		}
		if cases == nil && bag == nil {
			t.Fatal("the reader must always return a diagnostic bag")
		}
	})
}
