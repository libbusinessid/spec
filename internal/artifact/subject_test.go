package artifact

import (
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
)

// A subject node whose subtree reads SUBJECT defines the subject in terms of
// itself. No reading of the IR makes such a bundle usable: a generator emitting
// it recurses forever, an interpreter exhausts its budget.
//
// The Swift engine refused it on its own and asked for it to be written down.
// Nothing in the twenty five checks covered it: the subject node is checked for
// range and for type, and its subtree is never walked.
func TestSubjectNodeMayNotReadTheSubject(t *testing.T) {
	build := func(subjectReadsSubject bool) *irv1.Program {
		p := &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT}
		// 0: SUBJECT
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
			}},
		})
		// 1: a string the subject node can be built from
		inputs := []uint32{}
		if subjectReadsSubject {
			inputs = []uint32{0}
		}
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			InputNodes: inputs,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM,
			}},
		})
		subject := uint32(1)
		p.SubjectNode = &subject
		// 2: an assertion rooting the program
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
				Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS,
			}},
		})
		p.RootNode = 2
		return p
	}

	if err := checkSubjectNode(build(false)); err != nil {
		t.Fatalf("a subject node that does not read the subject is fine: %v", err)
	}
	err := checkSubjectNode(build(true))
	if err == nil {
		t.Fatal("a subject node defined in terms of the subject must be refused")
	}
	if !strings.Contains(err.Error(), "subject") {
		t.Fatalf("the error must name what is circular, got %v", err)
	}
}
