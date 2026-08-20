package artifact

import (
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/features"
)

// PROVENANCE_TIER_V1 exists because tier was added to a frozen capability, and
// the point of giving it its own id is that the two stay independent: a bundle
// that carries no tier must be readable by an engine that never heard of 41.
//
// tier is not optional in the schema, so an absent field and an explicit
// SOURCE_TIER_UNSPECIFIED are the same bytes. Refusing UNSPECIFIED therefore
// makes 41 mandatory the moment 40 is, which is the opposite of independent.
// UNSPECIFIED means the source states no tier; a value outside the enum is
// still a forged bundle.
func TestTierCapabilityStaysIndependentOfProvenance(t *testing.T) {
	source := func(tier irv1.SourceTier) *irv1.Source {
		return &irv1.Source{Id: "s", Url: "u", Authority: "a", Title: "t",
			AccessedAt: "2026-08-20", Jurisdiction: "FR", Language: "fr",
			LicenseOrTerms: "l", Tier: tier}
	}

	for name, tc := range map[string]struct {
		tier     irv1.SourceTier
		wantCap  bool
		wantFail string
	}{
		"no tier stated": {
			irv1.SourceTier_SOURCE_TIER_UNSPECIFIED, false, "",
		},
		"a stated tier needs the capability": {
			irv1.SourceTier_SOURCE_TIER_PRIMARY, true, "",
		},
		"a tier outside the enum is refused": {
			irv1.SourceTier(47), false, "unknown tier",
		},
	} {
		t.Run(name, func(t *testing.T) {
			v := &validator{
				bundle: &irv1.RuleBundle{
					// Declared generously: the question here is what the bundle
					// is found to use, not what it forgot to declare.
					RequiredFeatureIds: []uint32{
						features.CoreGraphV1, features.ProfilesV1,
						features.ProvenanceV1, features.ProvenanceTierV1,
					},
					Identifiers: []*irv1.IdentifierDefinition{{
						Id: 1, Kind: "k", Sources: []*irv1.Source{source(tc.tier)},
					}},
				},
				used: features.NewSet(),
			}
			err := v.validateDeclaredFeatures()
			if tc.wantFail != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantFail) {
					t.Fatalf("expected %q, got %v", tc.wantFail, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			got := v.used.Contains(features.ProvenanceTierV1)
			if got != tc.wantCap {
				t.Fatalf("capability 41 declared=%v, want %v", got, tc.wantCap)
			}
			if !v.used.Contains(features.ProvenanceV1) {
				t.Fatal("a source always needs PROVENANCE_V1")
			}
		})
	}
}
