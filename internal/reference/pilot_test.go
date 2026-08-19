package reference_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/artifact"
	"github.com/libbusinessid/spec/internal/reference"
)

var (
	pilotOnce   sync.Once
	pilotEngine *reference.Engine
	pilotErr    string
)

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

func engine(t *testing.T) *reference.Engine {
	t.Helper()
	pilotOnce.Do(func() {
		root := repoRoot(t)
		result, bag := artifact.CompileRules(filepath.Join(root, "rules"), artifact.CompileOptions{
			RulesVersion: "2026.08.0",
			Optimize:     true,
		})
		if bag.HasErrors() {
			var sb strings.Builder
			for _, d := range bag.Sorted() {
				sb.WriteString(d.String() + "\n")
			}
			pilotErr = sb.String()
			return
		}
		pilotEngine = reference.NewEngineFromRuleset(result.Ruleset)
	})
	if pilotErr != "" {
		t.Fatalf("the pilot rules do not compile:\n%s", pilotErr)
	}
	return pilotEngine
}

func TestPilotCompiles(t *testing.T) {
	e := engine(t)
	if e.Ruleset().RulesVersion() != "2026.08.0" {
		t.Fatalf("unexpected rules version %q", e.Ruleset().RulesVersion())
	}
}

func ptr(s string) *string { return &s }

func TestPilotCanonicalize(t *testing.T) {
	e := engine(t)
	tests := []struct {
		name    string
		in      reference.Input
		value   string
		country *string
		status  reference.StepStatus
		reason  irv1.ReasonCode
		kind    string
	}{
		{"siren spaced", reference.Input{Kind: "siren", Value: "012 345 674"},
			"012345674", ptr("FR"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "siren"},
		{"siren alias", reference.Input{Kind: " FR_SIREN ", Value: "012.345.674"},
			"012345674", ptr("FR"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "siren"},
		{"vat be prefixed", reference.Input{Kind: "vat", Value: "BE 0123.456.749"},
			"BE0123456749", ptr("BE"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "vat"},
		{"vat be from country", reference.Input{Kind: "vat_id", Value: "0123456749", CountryCode: ptr("be")},
			"BE0123456749", ptr("BE"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "vat"},
		{"vat be legacy nine digits", reference.Input{Kind: "vat", Value: "BE123456749"},
			"BE0123456749", ptr("BE"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "vat"},
		{"vat gr prefix el", reference.Input{Kind: "vat", Value: "EL012345670"},
			"EL012345670", ptr("GR"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "vat"},
		{"vat gr prefix gr", reference.Input{Kind: "vat", Value: "GR012345670"},
			"EL012345670", ptr("GR"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "vat"},
		{"vat gr country alias", reference.Input{Kind: "vat", Value: "012345670", CountryCode: ptr("EL")},
			"EL012345670", ptr("GR"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "vat"},
		{"euid", reference.Input{Kind: "euid", Value: "fr tvx.012345674"},
			"FRTVX.012345674", ptr("FR"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "euid"},
		{"lei global keeps country context", reference.Input{Kind: "lei", Value: "0000-0000-0000-0000-0098", CountryCode: ptr("fr")},
			"00000000000000000098", ptr("FR"), reference.StatusValid, irv1.ReasonCode_REASON_CODE_OK, "lei"},
		{"unknown kind", reference.Input{Kind: "Unknown-Kind", Value: "X"},
			"X", nil, reference.StatusUnsupported, irv1.ReasonCode_REASON_CODE_UNSUPPORTED_KIND, "unknown-kind"},
		{"unsupported country", reference.Input{Kind: "vat", Value: "0123456749", CountryCode: ptr("UK")},
			"0123456749", ptr("GB"), reference.StatusUnsupported, irv1.ReasonCode_REASON_CODE_UNSUPPORTED_COUNTRY, "vat"},
		{"invalid country token", reference.Input{Kind: "vat", Value: "0123456749", CountryCode: ptr("FRA")},
			"0123456749", ptr("FRA"), reference.StatusUnsupported, irv1.ReasonCode_REASON_CODE_UNSUPPORTED_COUNTRY, "vat"},
		{"country mismatch", reference.Input{Kind: "vat", Value: "BE0123456749", CountryCode: ptr("FR")},
			"BE0123456749", ptr("FR"), reference.StatusInvalid, irv1.ReasonCode_REASON_CODE_COUNTRY_MISMATCH, "vat"},
		{"missing country", reference.Input{Kind: "vat", Value: "0123456749"},
			"0123456749", nil, reference.StatusUnsupported, irv1.ReasonCode_REASON_CODE_MISSING_COUNTRY_CODE, "vat"},
		{"siren wrong country", reference.Input{Kind: "siren", Value: "012345674", CountryCode: ptr("DE")},
			"012345674", ptr("DE"), reference.StatusUnsupported, irv1.ReasonCode_REASON_CODE_UNSUPPORTED_COUNTRY, "siren"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.Canonicalize(tc.in, reference.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if got.Kind != tc.kind {
				t.Errorf("kind: got %q, want %q", got.Kind, tc.kind)
			}
			if got.CanonicalValue != tc.value {
				t.Errorf("canonical value: got %q, want %q", got.CanonicalValue, tc.value)
			}
			if got.InputValue != tc.in.Value {
				t.Errorf("input value must be preserved, got %q", got.InputValue)
			}
			if !equalPtr(got.CountryCode, tc.country) {
				t.Errorf("country: got %v, want %v", show(got.CountryCode), show(tc.country))
			}
			if got.Status != tc.status || got.ReasonCode != tc.reason {
				t.Errorf("status/reason: got %s/%v, want %s/%v", got.Status, got.ReasonCode, tc.status, tc.reason)
			}
		})
	}
}

func equalPtr(a, b *string) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}

func show(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func TestPilotCanonicalizationIsIdempotent(t *testing.T) {
	e := engine(t)
	values := []reference.Input{
		{Kind: "siren", Value: "012 345 674"},
		{Kind: "vat", Value: "BE 0123.456.749"},
		{Kind: "vat", Value: "BE123456749"},
		{Kind: "vat", Value: "GR012345670"},
		{Kind: "euid", Value: "FRTVX.012345674"},
		{Kind: "lei", Value: "0000-0000-0000-0000-0098"},
	}
	for _, in := range values {
		first, err := e.Canonicalize(in, reference.Options{})
		if err != nil {
			t.Fatal(err)
		}
		second, err := e.Canonicalize(reference.Input{
			Kind: in.Kind, Value: first.CanonicalValue, CountryCode: first.CountryCode,
		}, reference.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if first.CanonicalValue != second.CanonicalValue {
			t.Fatalf("%q is not idempotent: %q then %q", in.Value, first.CanonicalValue, second.CanonicalValue)
		}
	}
}
