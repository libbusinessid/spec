package main

import (
	"strings"
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/diagnostics"
)

// A canonicalizer that removes what its own format requires is a contradiction
// no conformance case will catch unless that country happens to have one. Three
// EUID rules shipped with it, copied from a rule whose separator is noise.
func TestLintStructuralSeparators(t *testing.T) {
	canon := func(text string) *irv1.Program {
		return &irv1.Program{Id: 1, Nodes: []*irv1.Node{{
			Operation: &irv1.Node_CanonicalizationOperation{
				CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_CHARS,
					Text: &text,
				}}}}}
	}
	format := func(text string) *irv1.Program {
		return &irv1.Program{Id: 2, Nodes: []*irv1.Node{{
			Operation: &irv1.Node_PredicateOperation{
				PredicateOperation: &irv1.PredicateOperation{
					Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_CONTAINS,
					Text: &text,
				}}}}}
	}
	country := "BE"

	for name, tc := range map[string]struct {
		removed, needed string
		wantErr         bool
	}{
		"removes the separator it requires": {".-", ".", true},
		"removes something else":            {"-/", ".", false},
		"removes nothing it needs":          {"", ".", false},
	} {
		t.Run(name, func(t *testing.T) {
			bundle := &irv1.RuleBundle{
				Programs: []*irv1.Program{canon(tc.removed), format(tc.needed)},
				Identifiers: []*irv1.IdentifierDefinition{{
					Kind:                    "euid",
					CountryCode:             &country,
					CanonicalizationProgram: 1,
					FormatProgram:           2,
				}},
			}
			bag := diagnostics.New()
			lintStructuralSeparators(bag, &buildResult{rules: &artifact.CompileResult{Bundle: bundle}})
			got := bag.HasErrors()
			if got != tc.wantErr {
				t.Fatalf("errors=%v, want %v: %v", got, tc.wantErr, bag.Sorted())
			}
			if tc.wantErr && !strings.Contains(bag.Sorted()[0].String(), "LINT005") {
				t.Fatalf("expected LINT005, got %v", bag.Sorted()[0])
			}
		})
	}
}
