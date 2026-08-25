package artifact

import (
	"strings"
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/features"
)

// ir.md section 9 puts PredicateOperation.values under the normative order --
// ascending, deduplicated -- and says an engine refuses a bundle that does not
// respect it. This loader checked only that no value was empty or over long.
//
// It was invisible while every engine scanned the list. It stopped being
// invisible when section 14 required the lookup not to be linear: a binary
// search over an unsorted list does not answer slowly, it answers wrongly. So a
// bundle with values out of order would be accepted here and then mis-answered
// by a conforming engine. Found by the TypeScript engine, in its own loader.
func TestPrefixInValuesMustBeAscendingAndDeduplicated(t *testing.T) {
	build := func(values []string) *irv1.Program {
		p := &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT}
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
			}},
		})
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
				Kind:   irv1.PredicateOpKind_PREDICATE_OP_KIND_PREFIX_IN,
				Values: values,
			}},
		})
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
			InputNodes: []uint32{1},
			Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{
				Kind:       irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE,
				ReasonCode: irv1.ReasonCode_REASON_CODE_INVALID_FORMAT.Enum(),
			}},
		})
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
			InputNodes: []uint32{2},
			Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{
				Kind: irv1.AssertionOpKind_ASSERTION_OP_KIND_SEQUENCE,
			}},
		})
		p.RootNode = 3
		return p
	}
	load := func(values []string) error {
		v := &validator{bundle: &irv1.RuleBundle{Programs: []*irv1.Program{build(values)}}, used: features.NewSet()}
		return v.validatePrograms()
	}

	if err := load([]string{"AB", "CD", "EF"}); err != nil {
		t.Fatalf("an ascending deduplicated list must load: %v", err)
	}
	for _, bad := range []struct {
		name   string
		values []string
	}{
		{"descending", []string{"CD", "AB"}},
		{"duplicated", []string{"AB", "AB"}},
		{"equal keys out of order", []string{"AB", "AA"}},
	} {
		err := load(bad.values)
		if err == nil {
			t.Errorf("%s: %v must be refused", bad.name, bad.values)
			continue
		}
		if !strings.Contains(err.Error(), "ascending") {
			t.Errorf("%s: the refusal must name the order, got %v", bad.name, err)
		}
	}
}

// A prefix_in never mixes element lengths.
//
// "starts with one of these" over a sorted list of mixed lengths is a trap, and
// two engines fell into different halves of it. With ["AB", "ABA"] and the input
// "ABCD", a search for the greatest element not after the input finds "ABA",
// which is not a prefix; the answer is "AB", which is. The search has to be run
// once per distinct element length, and nothing said so.
//
// The corpus cannot catch it: all four prefix_in nodes of the published bundle
// hold one length each, so a whole-table search passes every case while being
// wrong. The Swift engine stood four hundred random tables in for the cases that
// do not exist, and the TypeScript engine reasoned its way to the same shape.
//
// Rather than leave four engines to re-derive it, the bundle may not carry the
// shape at all. Nothing is lost: mixed lengths are written as one prefix_in per
// length under an any(), which is what the German rule already does.
func TestPrefixInDoesNotMixElementLengths(t *testing.T) {
	build := func(values []string) *irv1.Program {
		p := &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT}
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
			}},
		})
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
				Kind:   irv1.PredicateOpKind_PREDICATE_OP_KIND_PREFIX_IN,
				Values: values,
			}},
		})
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
			InputNodes: []uint32{1},
			Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{
				Kind:       irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE,
				ReasonCode: irv1.ReasonCode_REASON_CODE_INVALID_FORMAT.Enum(),
			}},
		})
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
			InputNodes: []uint32{2},
			Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{
				Kind: irv1.AssertionOpKind_ASSERTION_OP_KIND_SEQUENCE,
			}},
		})
		p.RootNode = 3
		return p
	}
	load := func(values []string) error {
		v := &validator{bundle: &irv1.RuleBundle{Programs: []*irv1.Program{build(values)}}, used: features.NewSet()}
		return v.validatePrograms()
	}

	if err := load([]string{"AB", "CD"}); err != nil {
		t.Fatalf("one length must load: %v", err)
	}
	err := load([]string{"AB", "ABA"})
	if err == nil {
		t.Fatal(`["AB", "ABA"] mixes two and three, and must be refused`)
	}
	if !strings.Contains(err.Error(), "length") {
		t.Fatalf("the refusal must name the lengths, got %v", err)
	}
}
