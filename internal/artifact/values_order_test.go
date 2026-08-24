package artifact

import (
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/features"
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
