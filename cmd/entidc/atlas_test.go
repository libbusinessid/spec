package main

import (
	"strings"
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/diagnostics"
)

// The atlas is the page a reader lands on to learn what this library covers, so
// a rule that ships without an entry is a rule nobody can find. It is written by
// hand because most of it is about the world, and a hand written page that
// tracks generated artifacts drifts.
func TestLintAtlas(t *testing.T) {
	def := func(kind, country string) *irv1.IdentifierDefinition {
		d := &irv1.IdentifierDefinition{Kind: kind}
		if country != "" {
			d.CountryCode = &country
		}
		return d
	}

	for name, tc := range map[string]struct {
		text    string
		def     *irv1.IdentifierDefinition
		wantErr bool
	}{
		"named with its country": {
			"| France | SIREN | INSEE |", def("siren", "FR"), false,
		},
		"country named by code": {
			"VAT covers Northern Ireland (`XI`) under the Windsor Framework",
			def("vat", "XI"), false,
		},
		"identifier missing": {
			"| France | nothing here |", def("siren", "FR"), true,
		},
		"country missing": {
			"| Spain | SIREN |", def("siren", "FR"), true,
		},
		// The first version of this lint compared a lower case kind against a
		// document that writes VAT, "Company number" and D-U-N-S, and rejected
		// all sixty five definitions it was meant to accept.
		"upper case spelling": {
			"| Belgium | VAT | mod 97 |", def("vat", "BE"), false,
		},
		"spaced spelling": {
			"| United Kingdom | Company number | Companies House |",
			def("company_number", "GB"), false,
		},
		"hyphenated spelling": {
			"| D-U-N-S | Dun & Bradstreet | worldwide |", def("duns", ""), false,
		},
		"a global identifier needs no country": {
			"| LEI | GLEIF |", def("lei", ""), false,
		},
		// A country carrying a definition but absent from the name table cannot
		// be looked for at all, so it must fail rather than pass silently.
		"country outside the name table": {
			"| Japan | everything about Japan |", def("vat", "JP"), true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			bag := diagnostics.New()
			checkAtlas(bag, &buildResult{rules: &artifact.CompileResult{
				Bundle: &irv1.RuleBundle{Identifiers: []*irv1.IdentifierDefinition{tc.def}},
			}}, tc.text)
			if got := bag.HasErrors(); got != tc.wantErr {
				t.Fatalf("errors=%v, want %v: %v", got, tc.wantErr, bag.Sorted())
			}
			if tc.wantErr && !strings.Contains(bag.Sorted()[0].String(), "LINT006") {
				t.Fatalf("expected LINT006, got %v", bag.Sorted()[0])
			}
		})
	}
}
