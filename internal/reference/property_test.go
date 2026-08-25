package reference_test

import (
	"math/rand"
	"strings"
	"sync"
	"testing"

	"github.com/entid-org/spec/internal/reference"
)

// The property tests below are the invariants every engine must satisfy. They
// are checked on the official rules with generated inputs.

const separators = ".-/ \t"

func generatedValues(seed int64, n int) []string {
	r := rand.New(rand.NewSource(seed))
	alphabet := []rune("0123456789ABCDEFXYZ.- /abcdef")
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		length := r.Intn(24)
		var b strings.Builder
		for j := 0; j < length; j++ {
			b.WriteRune(alphabet[r.Intn(len(alphabet))])
		}
		out = append(out, b.String())
	}
	return out
}

func allKinds() []string {
	return []string{"siren", "vat", "euid", "lei", "vat_id", "unknown"}
}

func TestPropertyCanonicalizationIsIdempotent(t *testing.T) {
	e := engine(t)
	countries := []*string{nil, ptr("FR"), ptr("BE"), ptr("GR"), ptr("DE"), ptr("ZZ")}
	for _, kind := range allKinds() {
		for _, value := range generatedValues(1, 200) {
			for _, country := range countries {
				first, err := e.Canonicalize(reference.Input{Kind: kind, Value: value, CountryCode: country},
					reference.Options{})
				if err != nil {
					t.Fatalf("canonicalize failed for %q/%q: %v", kind, value, err)
				}
				second, err := e.Canonicalize(reference.Input{
					Kind: kind, Value: first.CanonicalValue, CountryCode: first.CountryCode,
				}, reference.Options{})
				if err != nil {
					t.Fatal(err)
				}
				if first.CanonicalValue != second.CanonicalValue {
					t.Fatalf("canonicalization of %q/%q is not idempotent: %q then %q",
						kind, value, first.CanonicalValue, second.CanonicalValue)
				}
			}
		}
	}
}

func TestPropertyNoUserInputEverFails(t *testing.T) {
	e := engine(t)
	values := append(generatedValues(2, 300),
		"", " ", strings.Repeat("9", 1024), strings.Repeat("9", 1025),
		"  \ufeff", "\u00e9"+strings.Repeat("\u00df", 40), "\x00\x01")
	for _, kind := range allKinds() {
		for _, value := range values {
			in := reference.Input{Kind: kind, Value: value}
			if _, err := e.Validate(in, reference.Options{}); err != nil {
				t.Fatalf("validate failed for %q/%q: %v", kind, value, err)
			}
			if _, err := e.ValidateFormat(in, reference.Options{}); err != nil {
				t.Fatalf("validateFormat failed for %q/%q: %v", kind, value, err)
			}
			if _, err := e.Canonicalize(in, reference.Options{}); err != nil {
				t.Fatalf("canonicalize failed for %q/%q: %v", kind, value, err)
			}
		}
	}
}

func TestPropertySeparatorsAndCaseDoNotChangeTheCanonicalValue(t *testing.T) {
	e := engine(t)
	cases := []struct{ kind, value string }{
		{"siren", "012345674"},
		{"vat", "BE0123456749"},
		{"vat", "FR09012345674"},
		{"vat", "EL012345670"},
		{"vat", "DE123456789"},
		{"lei", "00000000000000000098"},
	}
	r := rand.New(rand.NewSource(3))
	for _, tc := range cases {
		reference0, err := e.Canonicalize(reference.Input{Kind: tc.kind, Value: tc.value}, reference.Options{})
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 50; i++ {
			var b strings.Builder
			for _, c := range tc.value {
				if r.Intn(3) == 0 {
					b.WriteByte(separators[r.Intn(len(separators))])
				}
				if c >= 'A' && c <= 'Z' && r.Intn(2) == 0 {
					b.WriteRune(c + ('a' - 'A'))
					continue
				}
				b.WriteRune(c)
			}
			decorated := b.String()
			got, err := e.Canonicalize(reference.Input{Kind: tc.kind, Value: decorated}, reference.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if got.CanonicalValue != reference0.CanonicalValue {
				t.Fatalf("%q and %q must canonicalize identically: %q vs %q",
					tc.value, decorated, reference0.CanonicalValue, got.CanonicalValue)
			}
		}
	}
}

func TestPropertyCheckDigitMutationInvalidates(t *testing.T) {
	e := engine(t)
	cases := []struct{ kind, value string }{
		{"siren", "012345674"},
		{"vat", "BE0123456749"},
		{"vat", "EL012345670"},
		{"lei", "00000000000000000098"},
	}
	for _, tc := range cases {
		base, err := e.Validate(reference.Input{Kind: tc.kind, Value: tc.value}, reference.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if base.Checksum.Status != reference.StatusValid {
			t.Fatalf("%q must be a valid vector, got %v", tc.value, base.Checksum)
		}
		runes := []rune(tc.value)
		last := len(runes) - 1
		for d := '0'; d <= '9'; d++ {
			if runes[last] == d {
				continue
			}
			mutated := string(append(append([]rune{}, runes[:last]...), d))
			got, err := e.Validate(reference.Input{Kind: tc.kind, Value: mutated}, reference.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if got.Format.Status == reference.StatusValid && got.Checksum.Status == reference.StatusValid {
				t.Fatalf("mutating the last digit of %q to %q must break the checksum", tc.value, string(d))
			}
		}
	}
}

func TestPropertyUnsupportedNeverBecomesInvalid(t *testing.T) {
	e := engine(t)
	// The German definition has no applicable algorithm: no input can ever make
	// its checksum step invalid.
	for _, value := range append(generatedValues(4, 200), "DE123456789", "DE000000000") {
		report, err := e.Validate(reference.Input{Kind: "vat", Value: value, CountryCode: ptr("DE")},
			reference.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if report.Checksum.Status == reference.StatusInvalid {
			t.Fatalf("%q produced an invalid checksum on a definition without algorithm", value)
		}
	}
}

func TestPropertyEngineIsDeterministicAndThreadSafe(t *testing.T) {
	e := engine(t)
	values := generatedValues(5, 60)
	expected := make([]reference.ValidationReport, len(values))
	for i, value := range values {
		report, err := e.Validate(reference.Input{Kind: "vat", Value: value}, reference.Options{})
		if err != nil {
			t.Fatal(err)
		}
		expected[i] = report
	}
	var wg sync.WaitGroup
	errs := make(chan string, len(values)*8)
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i, value := range values {
				got, err := e.Validate(reference.Input{Kind: "vat", Value: value}, reference.Options{})
				if err != nil {
					errs <- "engine error: " + err.Error()
					return
				}
				if !sameReport(got, expected[i]) {
					errs <- "non deterministic result for " + value
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errs)
	for message := range errs {
		t.Fatal(message)
	}
}

func TestPropertyTwoEnginesOnTheSameBytesAgree(t *testing.T) {
	first := engine(t)
	second := engine(t)
	for _, value := range generatedValues(6, 100) {
		a, err := first.Validate(reference.Input{Kind: "vat", Value: value}, reference.Options{})
		if err != nil {
			t.Fatal(err)
		}
		b, err := second.Validate(reference.Input{Kind: "vat", Value: value}, reference.Options{})
		if err != nil {
			t.Fatal(err)
		}
		if !sameReport(a, b) {
			t.Fatalf("two engines disagreed on %q", value)
		}
	}
}
