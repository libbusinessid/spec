package reference_test

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/reference"
)

// The conformance corpus is JSONL, and JSON cannot carry bytes that are not
// valid UTF-8, so this rule cannot be expressed as a conformance case. It is
// pinned here instead.
func engineForEncoding(t *testing.T) *reference.Engine {
	t.Helper()
	raw, err := os.ReadFile("../runner/testdata/rules.binpb")
	if err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	e, err := reference.NewEngine(raw)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	return e
}

var malformed = map[string]string{
	"lone continuation byte": "\xbf",
	"truncated sequence":     "\xef\xbb",
	"invalid inside digits":  "12\xff34",
	"fragments that rejoin":  "\xef\xbb\xef\xbb \xbf\xbf",
}

func TestInvalidEncodingIsRefusedByValidate(t *testing.T) {
	e := engineForEncoding(t)
	for name, in := range malformed {
		t.Run(name, func(t *testing.T) {
			if utf8.ValidString(in) {
				t.Fatal("the fixture must not be valid UTF-8")
			}
			report, err := e.Validate(reference.Input{Kind: "siren", Value: in}, reference.Options{})
			if err != nil {
				t.Fatalf("validate: %v", err)
			}
			if report.Format.Status != reference.StatusUnsupported {
				t.Fatalf("status: got %v, want unsupported", report.Format.Status)
			}
			if report.Format.ReasonCode != irv1.ReasonCode_REASON_CODE_INVALID_ENCODING {
				t.Fatalf("reason: got %v", report.Format.ReasonCode)
			}
			if report.Checksum.Status != reference.StatusNotRun {
				t.Fatalf("the checksum must not run: got %v", report.Checksum.Status)
			}
		})
	}
}

func TestInvalidEncodingIsRefusedByCanonicalize(t *testing.T) {
	e := engineForEncoding(t)
	for name, in := range malformed {
		t.Run(name, func(t *testing.T) {
			got, err := e.Canonicalize(reference.Input{Kind: "siren", Value: in}, reference.Options{})
			if err != nil {
				t.Fatalf("canonicalize: %v", err)
			}
			if got.Status != reference.StatusUnsupported ||
				got.ReasonCode != irv1.ReasonCode_REASON_CODE_INVALID_ENCODING {
				t.Fatalf("got %v/%v", got.Status, got.ReasonCode)
			}
			// The value is reported unchanged: nothing was substituted for the
			// malformed bytes.
			if got.CanonicalValue != in {
				t.Fatalf("the input must be reported verbatim, got %q", got.CanonicalValue)
			}
		})
	}
}

// Before the refusal, filtering malformed bytes replaced each with U+FFFD and
// tripled the value. Refusing upstream is what keeps every canonicalization
// step non growing.
func TestCanonicalizationNeverGrowsAnAcceptedValue(t *testing.T) {
	e := engineForEncoding(t)
	for _, in := range []string{"552 100 554", "  552100554  ", strings.Repeat("1", 900)} {
		got, err := e.Canonicalize(reference.Input{Kind: "siren", Value: in}, reference.Options{})
		if err != nil {
			t.Fatalf("canonicalize: %v", err)
		}
		if len(got.CanonicalValue) > len(in) {
			t.Fatalf("%q grew to %q", in, got.CanonicalValue)
		}
	}
}
