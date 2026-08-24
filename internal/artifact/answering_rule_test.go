package artifact

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Each hostile fixture is pinned to the rule that refuses it.
//
// Proving a fixture has exactly one fault needs a repair, and a repair needs to
// know which field the case is named for - which is the per fixture knowledge a
// generic method cannot have. The TypeScript engine measured that: a baseline is
// recoverable from the fixtures themselves by component wise majority vote, and
// restoring the differing components makes thirty four of thirty five load. Run
// against the corpus where two fixtures were known to be double faulted, the
// method reported both clean, because both faults lived inside `programs` and
// restoring it wholesale repaired both at once. On left_pad_length both faults
// were properties of one node. Any granularity coarse enough to be generic is
// coarse enough to hide the second fault.
//
// So isolation stays hand written, three cases of it, and this pins the cheap
// half instead: which rule answers. It would not have caught left_pad_length,
// whose bound both named and answered. It catches the dangerous direction - a
// second fault appearing earlier in the order, so the fixture never reaches its
// own rule. That is subject_node_circular answering 25 instead of 15.
func TestEachHostileFixtureIsRefusedByTheSameRuleAsBefore(t *testing.T) {
	// The exact text this loader answers. A change here is not a failure by
	// itself: it means a fixture now stops at a different rule, and somebody has
	// to say whether that is the intended one.
	expected := map[string]string{
		"alphabet_empty.binpb":              "weighted_sum uses custom_alphabet without an alphabet",
		"alphabet_missing.binpb":            "weighted_sum uses custom_alphabet without an alphabet",
		"alphabet_repeated.binpb":           `the alphabet lists "0" twice`,
		"alphabet_too_many.binpb":           "the alphabet holds 257 code points, above the accepted 256",
		"alphabet_unread.binpb":             "weighted_sum carries an alphabet that only custom_alphabet reads",
		"call_cycle.binpb":                  "the call graph holds a cycle through program 3",
		"duplicate_prefix.binpb":            `dispatcher "demo" maps the prefix "XX" to two targets`,
		"empty.binpb":                       "unsupported format_version 0",
		"empty_message_key.binpb":           "message key must not be empty when present",
		"empty_rules_version.binpb":         "rules_version must not be empty",
		"forbidden_reason_code.binpb":       "cannot prove a format invalidity",
		"global_target_with_prefix.binpb":   `the GLOBAL target of "demo" declares prefixes`,
		"left_pad_length.binpb":             "index 4097 exceeds the limit of 4096",
		"minimal_conformance.binpb":         "unknown capability id 0",
		"missing_operation.binpb":           "node carries no operation",
		"modulus_out_of_range.binpb":        "modulus 1 is outside 2..1000000000",
		"node_forward_reference.binpb":      "references node 5 which is not lower",
		"node_out_of_range.binpb":           "has an out of range root node",
		"orphan_definition.binpb":           "identifier 2 is not referenced by any dispatch target",
		"predicate_constant.binpb":          "constant 1000000001 is outside the accepted range",
		"prefix_in_unsorted.binpb":          "values must be ascending and deduplicated",
		"program_expansion.binpb":           "expands to more than 100000 operation instances",
		"rules_version_shape.binpb":         `rules_version holds '\x00'`,
		"short_digest.binpb":                "source_digest must hold exactly 32 bytes, got 16",
		"source_tier_unknown.binpb":         "declares an unknown tier",
		"stray_parameter.binpb":             `must not carry the parameter "text"`,
		"stray_when_branch.binpb":           "must not be rooted in a WHEN branch",
		"subject_node_circular.binpb":       "builds its subject node from the subject it defines",
		"truncated.binpb":                   "protobuf decoding failed",
		"type_mismatch.binpb":               "declares output VALUE_TYPE_STRING but",
		"unbounded_digits_to_integer.binpb": "needs a provable bound of at most 18 digits",
		"undeclared_feature.binpb":          "without declaring it",
		"unknown_call_target.binpb":         "calls the unknown program 42",
		"unknown_feature.binpb":             "unknown capability id 9999",
		"unknown_field_root.binpb":          "unknown Protobuf field at",
		"unspecified_enum.binpb":            "program 1 has an unspecified kind",
		"when_unreferenced.binpb":           "a WHEN branch is only accepted as a direct operand of CHOOSE",
		"unsupported_format_version.binpb":  "unsupported format_version 2",
	}

	dir := filepath.Join("..", "..", "testdata", "bundles")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading the fixtures: %v", err)
	}
	var seen []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".binpb" {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("%s: %v", e.Name(), err)
		}
		want, hostile := expected[e.Name()]
		_, lerr := LoadRuleset(raw)
		if !hostile {
			if lerr != nil {
				t.Errorf("%s is not listed as hostile but is refused: %v", e.Name(), lerr)
			}
			continue
		}
		seen = append(seen, e.Name())
		if lerr == nil {
			t.Errorf("%s loads; it is supposed to be refused by %q", e.Name(), want)
			continue
		}
		if !strings.Contains(lerr.Error(), want) {
			t.Errorf("%s stops at a different rule than before.\n  was: %s\n  now: %v\n"+
				"A fault that answers earlier in the order means the fixture no longer "+
				"reaches the rule its case is named for.", e.Name(), want, lerr)
		}
	}
	if len(seen) != len(expected) {
		sort.Strings(seen)
		t.Errorf("listed %d hostile fixtures, found %d on disk: %s",
			len(expected), len(seen), fmt.Sprint(seen))
	}
}
