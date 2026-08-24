package reference_test

import (
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/reference"
)

type step struct {
	status reference.StepStatus
	reason irv1.ReasonCode
	key    string
}

func checkStep(t *testing.T, label string, got reference.StepResult, want step, level reference.Level) {
	t.Helper()
	if got.Level != level {
		t.Errorf("%s level: got %s, want %s", label, got.Level, level)
	}
	if got.Status != want.status {
		t.Errorf("%s status: got %s, want %s", label, got.Status, want.status)
	}
	if got.ReasonCode != want.reason {
		t.Errorf("%s reason: got %v, want %v", label, got.ReasonCode, want.reason)
	}
	switch {
	case want.key == "" && got.MessageKey != nil:
		t.Errorf("%s message key: got %q, want none", label, *got.MessageKey)
	case want.key != "" && (got.MessageKey == nil || *got.MessageKey != want.key):
		t.Errorf("%s message key: got %v, want %q", label, show(got.MessageKey), want.key)
	}
}

const (
	ok               = irv1.ReasonCode_REASON_CODE_OK
	empty            = irv1.ReasonCode_REASON_CODE_EMPTY
	badLength        = irv1.ReasonCode_REASON_CODE_INVALID_LENGTH
	badChars         = irv1.ReasonCode_REASON_CODE_INVALID_CHARACTERS
	badFormat        = irv1.ReasonCode_REASON_CODE_INVALID_FORMAT
	badChecksum      = irv1.ReasonCode_REASON_CODE_INVALID_CHECKSUM
	notRequested     = irv1.ReasonCode_REASON_CODE_NOT_REQUESTED
	notRunInvalid    = irv1.ReasonCode_REASON_CODE_NOT_RUN_FORMAT_INVALID
	notRunUnsup      = irv1.ReasonCode_REASON_CODE_NOT_RUN_FORMAT_UNSUPPORTED
	unsupChecksum    = irv1.ReasonCode_REASON_CODE_UNSUPPORTED_CHECKSUM
	checksumNotPub   = irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED
	countryMismatch  = irv1.ReasonCode_REASON_CODE_COUNTRY_MISMATCH
	unsupportedKind  = irv1.ReasonCode_REASON_CODE_UNSUPPORTED_KIND
	missingCountry   = irv1.ReasonCode_REASON_CODE_MISSING_COUNTRY_CODE
	inputTooLong     = irv1.ReasonCode_REASON_CODE_INPUT_TOO_LONG
	unsupCountryCode = irv1.ReasonCode_REASON_CODE_UNSUPPORTED_COUNTRY
)

func TestPilotValidate(t *testing.T) {
	e := engine(t)
	tests := []struct {
		name     string
		in       reference.Input
		profile  reference.Profile
		value    string
		format   step
		checksum step
	}{
		{"siren valid", reference.Input{Kind: "siren", Value: "012 345 674"}, "",
			"012345674", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"siren second valid", reference.Input{Kind: "siren", Value: "123456782"}, "",
			"123456782", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"siren empty", reference.Input{Kind: "siren", Value: ""}, "",
			"", step{reference.StatusInvalid, empty, "fr.siren.empty"}, step{reference.StatusNotRun, notRunInvalid, ""}},
		{"siren too short", reference.Input{Kind: "siren", Value: "01234567"}, "",
			"01234567", step{reference.StatusInvalid, badLength, "fr.siren.length"}, step{reference.StatusNotRun, notRunInvalid, ""}},
		{"siren too long", reference.Input{Kind: "siren", Value: "0123456740"}, "",
			"0123456740", step{reference.StatusInvalid, badLength, "fr.siren.length"}, step{reference.StatusNotRun, notRunInvalid, ""}},
		{"siren letters", reference.Input{Kind: "siren", Value: "01234567A"}, "",
			"01234567A", step{reference.StatusInvalid, badChars, "fr.siren.characters"}, step{reference.StatusNotRun, notRunInvalid, ""}},
		{"siren checksum mutation", reference.Input{Kind: "siren", Value: "012345675"}, "",
			"012345675", step{reference.StatusValid, ok, ""}, step{reference.StatusInvalid, badChecksum, ""}},

		{"vat be valid", reference.Input{Kind: "vat", Value: "BE 0123.456.749"}, "",
			"BE0123456749", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"vat be legacy", reference.Input{Kind: "vat", Value: "BE123456749"}, "",
			"BE0123456749", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"vat be second valid", reference.Input{Kind: "vat", Value: "BE1000000021"}, "",
			"BE1000000021", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"vat be checksum mutation", reference.Input{Kind: "vat", Value: "BE0123456748"}, "",
			"BE0123456748", step{reference.StatusValid, ok, ""}, step{reference.StatusInvalid, badChecksum, ""}},
		{"vat be bad enterprise prefix", reference.Input{Kind: "vat", Value: "BE2123456749"}, "",
			"BE2123456749", step{reference.StatusInvalid, badFormat, "vat.be.enterprise_prefix"}, step{reference.StatusNotRun, notRunInvalid, ""}},
		{"vat be letters", reference.Input{Kind: "vat", Value: "BE012345674A"}, "",
			"BE012345674A", step{reference.StatusInvalid, badChars, "vat.be.characters"}, step{reference.StatusNotRun, notRunInvalid, ""}},

		{"vat fr valid", reference.Input{Kind: "vat", Value: "FR 09 012345674"}, "",
			"FR09012345674", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"vat fr second valid", reference.Input{Kind: "vat", Value: "FR11123456782"}, "",
			"FR11123456782", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"vat fr broken embedded siren", reference.Input{Kind: "vat", Value: "FR12012345675"}, "",
			"FR12012345675", step{reference.StatusValid, ok, ""}, step{reference.StatusInvalid, badChecksum, ""}},
		{"vat fr checksum mutation", reference.Input{Kind: "vat", Value: "FR10012345674"}, "",
			"FR10012345674", step{reference.StatusValid, ok, ""}, step{reference.StatusInvalid, badChecksum, ""}},
		{"vat fr alphanumeric key stays unsupported", reference.Input{Kind: "vat", Value: "FRK7012345674"}, "",
			"FRK7012345674", step{reference.StatusValid, ok, ""}, step{reference.StatusUnsupported, checksumNotPub, ""}},
		{"vat fr alphanumeric key with a broken siren is invalid", reference.Input{Kind: "vat", Value: "FRK7012345675"}, "",
			"FRK7012345675", step{reference.StatusValid, ok, ""}, step{reference.StatusInvalid, badChecksum, ""}},
		{"vat fr alphanumeric key refused by strict", reference.Input{Kind: "vat", Value: "FRK7012345674"}, reference.ProfileStrictCurrent,
			"FRK7012345674", step{reference.StatusInvalid, badChars, "vat.fr.key_characters"}, step{reference.StatusNotRun, notRunInvalid, ""}},

		{"vat de format only", reference.Input{Kind: "vat", Value: "DE123456789"}, "",
			"DE123456789", step{reference.StatusValid, ok, ""}, step{reference.StatusUnsupported, unsupChecksum, ""}},
		{"vat de bad length", reference.Input{Kind: "vat", Value: "DE12345678"}, "",
			"DE12345678", step{reference.StatusInvalid, badLength, "vat.de.length"}, step{reference.StatusNotRun, notRunInvalid, ""}},

		{"vat gr valid el", reference.Input{Kind: "vat", Value: "EL012345670"}, "",
			"EL012345670", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"vat gr valid gr prefix", reference.Input{Kind: "vat", Value: "GR 012 345 670"}, "",
			"EL012345670", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"vat gr second valid", reference.Input{Kind: "vat", Value: "EL099999999"}, "",
			"EL099999999", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"vat gr checksum mutation", reference.Input{Kind: "vat", Value: "EL012345671"}, "",
			"EL012345671", step{reference.StatusValid, ok, ""}, step{reference.StatusInvalid, badChecksum, ""}},

		{"euid valid", reference.Input{Kind: "euid", Value: "FR7501.012345674"}, "",
			"FR7501.012345674", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"euid bad siren length", reference.Input{Kind: "euid", Value: "FR7501.01234567"}, "",
			"FR7501.01234567", step{reference.StatusInvalid, badLength, "fr.siren.length"}, step{reference.StatusNotRun, notRunInvalid, ""}},
		{"euid bad siren checksum", reference.Input{Kind: "euid", Value: "FR7501.012345675"}, "",
			"FR7501.012345675", step{reference.StatusValid, ok, ""}, step{reference.StatusInvalid, badChecksum, ""}},
		{"euid missing separator", reference.Input{Kind: "euid", Value: "FRTVX012345674"}, "",
			"FRTVX012345674", step{reference.StatusInvalid, badFormat, "euid.fr.separator"}, step{reference.StatusNotRun, notRunInvalid, ""}},

		{"lei valid", reference.Input{Kind: "lei", Value: "0000-0000-0000-0000-0098"}, "",
			"00000000000000000098", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"lei alphanumeric valid", reference.Input{Kind: "lei", Value: "000000ABCDEF12345670"}, "",
			"000000ABCDEF12345670", step{reference.StatusValid, ok, ""}, step{reference.StatusValid, ok, ""}},
		{"lei checksum mutation", reference.Input{Kind: "lei", Value: "00000000000000000097"}, "",
			"00000000000000000097", step{reference.StatusValid, ok, ""}, step{reference.StatusInvalid, badChecksum, ""}},
		{"lei bad check digits", reference.Input{Kind: "lei", Value: "0000000000000000009A"}, "",
			"0000000000000000009A", step{reference.StatusInvalid, badChars, "lei.check_digits"}, step{reference.StatusNotRun, notRunInvalid, ""}},

		{"unknown kind", reference.Input{Kind: "nope", Value: "X"}, "",
			"X", step{reference.StatusUnsupported, unsupportedKind, ""}, step{reference.StatusNotRun, notRunUnsup, ""}},
		{"country mismatch", reference.Input{Kind: "vat", Value: "BE0123456749", CountryCode: ptr("FR")}, "",
			"BE0123456749", step{reference.StatusInvalid, countryMismatch, ""}, step{reference.StatusNotRun, notRunInvalid, ""}},
		{"missing country", reference.Input{Kind: "vat", Value: "0123456749"}, "",
			"0123456749", step{reference.StatusUnsupported, missingCountry, ""}, step{reference.StatusNotRun, notRunUnsup, ""}},
		{"unsupported country", reference.Input{Kind: "vat", Value: "0123456749", CountryCode: ptr("JP")}, "",
			"0123456749", step{reference.StatusUnsupported, unsupCountryCode, ""}, step{reference.StatusNotRun, notRunUnsup, ""}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report, err := e.Validate(tc.in, reference.Options{Profile: tc.profile})
			if err != nil {
				t.Fatal(err)
			}
			if report.CanonicalValue != tc.value {
				t.Errorf("canonical value: got %q, want %q", report.CanonicalValue, tc.value)
			}
			if report.InputValue != tc.in.Value {
				t.Errorf("raw input must be preserved")
			}
			checkStep(t, "format", report.Format, tc.format, reference.LevelFormat)
			checkStep(t, "checksum", report.Checksum, tc.checksum, reference.LevelChecksum)

			same, err := e.ValidateChecksum(tc.in, reference.Options{Profile: tc.profile})
			if err != nil {
				t.Fatal(err)
			}
			if !sameReport(same, report) {
				t.Errorf("validateChecksum must return the same report as validate")
			}
		})
	}
}

func TestValidateFormatStopsBeforeChecksum(t *testing.T) {
	e := engine(t)
	report, err := e.ValidateFormat(reference.Input{Kind: "siren", Value: "012345675"}, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	checkStep(t, "format", report.Format, step{reference.StatusValid, ok, ""}, reference.LevelFormat)
	checkStep(t, "checksum", report.Checksum, step{reference.StatusNotRun, notRequested, ""}, reference.LevelChecksum)

	invalid, err := e.ValidateFormat(reference.Input{Kind: "siren", Value: "1"}, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	checkStep(t, "checksum", invalid.Checksum, step{reference.StatusNotRun, notRunInvalid, ""}, reference.LevelChecksum)
}

func TestInputTooLong(t *testing.T) {
	e := engine(t)
	long := make([]byte, 1025)
	for i := range long {
		long[i] = '1'
	}
	in := reference.Input{Kind: "siren", Value: string(long), CountryCode: ptr("fr")}
	report, err := e.Validate(in, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	checkStep(t, "format", report.Format, step{reference.StatusUnsupported, inputTooLong, ""}, reference.LevelFormat)
	checkStep(t, "checksum", report.Checksum, step{reference.StatusNotRun, notRunUnsup, ""}, reference.LevelChecksum)
	if report.CanonicalValue != in.Value {
		t.Error("an over long input keeps the raw value")
	}
	if report.CountryCode == nil || *report.CountryCode != "fr" {
		t.Error("an over long input keeps the raw country context")
	}

	canon, err := e.Canonicalize(in, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if canon.Status != reference.StatusUnsupported || canon.ReasonCode != inputTooLong {
		t.Errorf("unexpected canonicalization result %v/%v", canon.Status, canon.ReasonCode)
	}
}

func TestRegistryInterfaceHasNoProvider(t *testing.T) {
	e := engine(t)
	got, err := e.RegistryLookup(reference.Input{Kind: "siren", Value: "012345674"}, nil, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != reference.RegistryUnsupported ||
		got.ReasonCode != irv1.ReasonCode_REASON_CODE_REGISTRY_NOT_CONFIGURED {
		t.Fatalf("unexpected registry result %+v", got)
	}
}

// sameReport compares two reports semantically: pointer identity is never part
// of the normative contract.
func sameReport(a, b reference.ValidationReport) bool {
	return a.Kind == b.Kind && a.InputValue == b.InputValue &&
		a.CanonicalValue == b.CanonicalValue && equalPtr(a.CountryCode, b.CountryCode) &&
		a.Profile == b.Profile && a.RulesVersion == b.RulesVersion &&
		a.FormatVersion == b.FormatVersion &&
		sameStep(a.Format, b.Format) && sameStep(a.Checksum, b.Checksum)
}

func sameStep(a, b reference.StepResult) bool {
	return a.Level == b.Level && a.Status == b.Status &&
		a.ReasonCode == b.ReasonCode && equalPtr(a.MessageKey, b.MessageKey)
}
