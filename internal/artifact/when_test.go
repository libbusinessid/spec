package artifact

import (
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/features"
)

// A WHEN branch nobody references is still not inside a CHOOSE.
//
// Check 16 accepts WHEN only inside CHOOSE, and this loader enforced it by
// looking at each node's parents. A node with no parent at all has none to look
// at, so a dead WHEN passed here and was refused by the Kotlin engine, which
// read the rule as written. Section 2 permits unreachable nodes, so the bundle
// is otherwise well formed and nothing else would catch it.
func TestADeadWhenBranchIsRefused(t *testing.T) {
	build := func(dead bool) *irv1.Program {
		p := &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_CHECKSUM}
		// 0: the subject, which the checksum reads.
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
			}},
		})
		// 1: an unsupported checksum, a legal root.
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
			Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
				Kind:       irv1.ChecksumOpKind_CHECKSUM_OP_KIND_UNSUPPORTED,
				ReasonCode: irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED.Enum(),
			}},
		})
		p.RootNode = 1
		if dead {
			// 2: a predicate, the condition the WHEN branch tests.
			p.Nodes = append(p.Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
					Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS,
				}},
			})
			// 3: a WHEN nobody references. No parent, so no parent to refuse it.
			p.Nodes = append(p.Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				InputNodes: []uint32{2, 1},
				Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
					Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN,
				}},
			})
		}
		return p
	}

	sound := &validator{bundle: &irv1.RuleBundle{Programs: []*irv1.Program{build(false)}}, used: features.NewSet()}
	if err := sound.validatePrograms(); err != nil {
		t.Fatalf("the program without the dead branch must load: %v", err)
	}

	v := &validator{bundle: &irv1.RuleBundle{Programs: []*irv1.Program{build(true)}}, used: features.NewSet()}
	err := v.validatePrograms()
	if err == nil {
		t.Fatal("a WHEN branch that is not a direct operand of a CHOOSE must be refused, referenced or not")
	}
	if !strings.Contains(err.Error(), "CHOOSE") {
		t.Fatalf("the refusal must name the rule, got %v", err)
	}
}
