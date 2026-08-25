package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
)

func minimalBundle(t *testing.T) *irv1.RuleBundle {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "testdata", "bundles", "minimal_valid.binpb"))
	if err != nil {
		t.Fatal(err)
	}
	bundle := &irv1.RuleBundle{}
	if err := proto.Unmarshal(raw, bundle); err != nil {
		t.Fatal(err)
	}
	return bundle
}

func classesOf(changes []Change) map[ChangeClass][]string {
	out := map[ChangeClass][]string{}
	for _, c := range changes {
		out[c.Class] = append(out[c.Class], c.Subject+": "+c.Detail)
	}
	return out
}

func TestDiffDetectsNoChange(t *testing.T) {
	base := minimalBundle(t)
	if changes := DiffBundles(base, proto.Clone(base).(*irv1.RuleBundle)); len(changes) != 0 {
		t.Fatalf("expected no change, got %+v", changes)
	}
}

func TestDiffClassifiesChanges(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(b *irv1.RuleBundle)
		class  ChangeClass
		detail string
	}{
		{"format version", func(b *irv1.RuleBundle) { b.FormatVersion = 2 },
			ClassPotentialIncompatibility, "becomes 2"},
		{"rules version", func(b *irv1.RuleBundle) { b.RulesVersion = "2026.09.0" },
			ClassMetadata, "becomes"},
		{"new capability", func(b *irv1.RuleBundle) { b.RequiredFeatureIds = append(b.RequiredFeatureIds, 32) },
			ClassIRFeature, "is now required"},
		{"dropped capability", func(b *irv1.RuleBundle) { b.RequiredFeatureIds = b.RequiredFeatureIds[:2] },
			ClassIRFeature, "no longer required"},
		{"new definition", func(b *irv1.RuleBundle) {
			extra := proto.Clone(b.Identifiers[0]).(*irv1.IdentifierDefinition)
			extra.Id = 2
			country := "FR"
			extra.CountryCode = &country
			b.Identifiers = append(b.Identifiers, extra)
		}, ClassWidening, "definition is new"},
		{"removed definition", func(b *irv1.RuleBundle) { b.Identifiers = nil },
			ClassRestriction, "definition disappears"},
		{"profile change", func(b *irv1.RuleBundle) { b.Identifiers[0].DefaultProfile = "strict_current" },
			ClassPotentialIncompatibility, "default profile becomes"},
		{"checksum added", func(b *irv1.RuleBundle) { b.Identifiers[0].ChecksumProgram = nil },
			ClassRestriction, "checksum is now applied"},
		{"sources changed", func(b *irv1.RuleBundle) { b.Identifiers[0].Sources = nil },
			ClassMetadata, "sources become"},
		{"format program changed", func(b *irv1.RuleBundle) {
			b.Programs[1].Nodes = b.Programs[1].Nodes[:3]
			b.Programs[1].RootNode = 2
		}, ClassPotentialIncompatibility, "format program changed"},
		{"canonicalization changed", func(b *irv1.RuleBundle) {
			b.Programs[0].Nodes[2].InputNodes = []uint32{0}
		}, ClassPotentialIncompatibility, "canonicalization program changed"},
		{"checksum program changed", func(b *irv1.RuleBundle) {
			b.Programs[2].Nodes[1].GetChecksumOperation().Kind =
				irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ISO7064_MOD97_10
		}, ClassPotentialIncompatibility, "checksum program changed"},
		{"new routing entry", func(b *irv1.RuleBundle) {
			b.Dispatchers[0].KindAliases = []string{"demo_alias"}
		}, ClassWidening, "routing entry is new"},
		{"removed routing entry", func(b *irv1.RuleBundle) { b.Dispatchers = nil },
			ClassRestriction, "routing entry disappears"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			before := minimalBundle(t)
			after := minimalBundle(t)
			if tc.name == "checksum added" {
				// The direction is reversed: the old bundle has no checksum.
				tc.mutate(before)
			} else {
				tc.mutate(after)
			}
			changes := DiffBundles(before, after)
			byClass := classesOf(changes)
			details, ok := byClass[tc.class]
			if !ok {
				t.Fatalf("expected the class %q, got %+v", tc.class, changes)
			}
			if !strings.Contains(strings.Join(details, "\n"), tc.detail) {
				t.Fatalf("expected %q in %v", tc.detail, details)
			}
		})
	}
}

func TestDiffReportsChecksumRemovalAsWidening(t *testing.T) {
	before := minimalBundle(t)
	after := minimalBundle(t)
	after.Identifiers[0].ChecksumProgram = nil
	changes := DiffBundles(before, after)
	byClass := classesOf(changes)
	if len(byClass[ClassWidening]) == 0 {
		t.Fatalf("removing a checksum widens the rule, got %+v", changes)
	}
}

func TestProgramFingerprintOfAMissingProgram(t *testing.T) {
	base := minimalBundle(t)
	if got := programFingerprint(base, 999); got != "missing" {
		t.Fatalf("unexpected fingerprint %q", got)
	}
}

func TestCapabilityNameFallback(t *testing.T) {
	if got := capabilityName(9999); got != "capability-9999" {
		t.Fatalf("unexpected name %q", got)
	}
	if got := capabilityName(1); got != "CORE_GRAPH_V1" {
		t.Fatalf("unexpected name %q", got)
	}
}

func TestOperationNameCoversEveryBranch(t *testing.T) {
	nodes := []*irv1.Node{
		{Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{}}},
		{Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{}}},
		{Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{}}},
		{Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{}}},
		{Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{}}},
		{Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{}}},
		{Operation: &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{}}},
		{},
	}
	for i, n := range nodes {
		if operationName(n) == "" {
			t.Fatalf("node %d produced an empty name", i)
		}
	}
	if operationName(nodes[len(nodes)-1]) != "UNKNOWN" {
		t.Fatal("a node without operation must be reported as UNKNOWN")
	}
}
