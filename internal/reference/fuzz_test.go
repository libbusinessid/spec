package reference_test

import (
	"strings"
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"

	"github.com/entid-org/spec/internal/reference"
)

// FuzzValidateInput exercises the four public operations with arbitrary user
// input, kinds and country contexts, including unusual Unicode.
func FuzzValidateInput(f *testing.F) {
	seeds := []struct {
		kind, value, country string
		hasCountry           bool
		profile              string
	}{
		{"siren", "012345674", "FR", true, "compatible"},
		{"vat", "BE0123456749", "", false, "compatible"},
		{"vat", "GR012345670", "EL", true, "strict_current"},
		{"euid", "FR7501.012345674", "", false, "compatible"},
		{"lei", "00000000000000000098", "fr", true, "compatible"},
		{"", "", "", false, ""},
		{"vat", " \ufeff BE0123456749", "", false, "compatible"},
		{"vat", "\xff\xfe", "\xff", true, "compatible"},
		{"unknown", "0000000000000000000000000000", "ZZ", true, "compatible"},
	}
	for _, seed := range seeds {
		f.Add(seed.kind, seed.value, seed.country, seed.hasCountry, seed.profile)
	}
	f.Fuzz(func(t *testing.T, kind, value, country string, hasCountry bool, profile string) {
		if len(value) > 4096 || len(kind) > 512 || len(country) > 512 {
			t.Skip("oversized input")
		}
		e := engine(t)
		in := reference.Input{Kind: kind, Value: value}
		if hasCountry {
			in.CountryCode = &country
		}
		opts := reference.Options{Profile: reference.Profile(profile)}
		canon, err := e.Canonicalize(in, opts)
		if err != nil {
			t.Fatalf("canonicalize must never fail on user input: %v", err)
		}
		if canon.InputValue != value {
			t.Fatal("the raw input must be preserved")
		}
		if canon.Status == reference.StatusNotRun {
			t.Fatal("not_run is never a final canonicalization status")
		}
		for _, op := range []func(reference.Input, reference.Options) (reference.ValidationReport, error){
			e.Validate, e.ValidateFormat, e.ValidateChecksum,
		} {
			report, err := op(in, opts)
			if err != nil {
				t.Fatalf("validation must never fail on user input: %v", err)
			}
			if report.InputValue != value {
				t.Fatal("the raw input must be preserved")
			}
			checkStatusReason(t, report.Format)
			checkStatusReason(t, report.Checksum)
		}
	})
}

func reasonName(code irv1.ReasonCode) string {
	return strings.ToLower(strings.TrimPrefix(irv1.ReasonCode_name[int32(code)], "REASON_CODE_"))
}

func checkStatusReason(t *testing.T, step reference.StepResult) {
	t.Helper()
	name := reasonName(step.ReasonCode)
	switch step.Status {
	case reference.StatusValid:
		if name != "ok" {
			t.Fatalf("a valid step must use ok, got %q", name)
		}
	case reference.StatusNotRun:
		switch name {
		case "not_requested", "not_run_format_invalid", "not_run_format_unsupported":
		default:
			t.Fatalf("a not_run step cannot use %q", name)
		}
	case reference.StatusInvalid:
		switch name {
		case "empty", "invalid_length", "invalid_characters", "invalid_format",
			"invalid_checksum", "country_mismatch":
		default:
			t.Fatalf("an invalid step must prove the invalidity, got %q", name)
		}
	case reference.StatusUnsupported:
		if name == "ok" {
			t.Fatal("an unsupported step cannot use ok")
		}
	default:
		t.Fatalf("unknown status %q", step.Status)
	}
	if step.Status != reference.StatusValid && step.MessageKey != nil && *step.MessageKey == "" {
		t.Fatal("an empty message key is never emitted")
	}
}
