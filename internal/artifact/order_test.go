package artifact

import (
	"strings"
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/features"
)

// ir.md section 10 runs the arithmetic bounds at check 13 and the program shape
// at check 16, and says the checks run in that order. The rule that an operation
// category must suit the program kind is part of the shape: ir.md section 2
// states it as a table, one row per kind.
//
// This loader enforced it in the per node pass instead, so a bundle carrying
// both faults was refused for the misplaced category rather than for the bound.
// That is how one fixture looked refused by check 13 to one engine and by check
// 16 to another, and why nobody could say which fault the case was proving.
func TestTheArithmeticBoundsAreCheckedBeforeTheProgramShape(t *testing.T) {
	// A format program holding a canonicalization operation - foreign to its
	// kind - whose length also exceeds the index bound. Two faults, and ir.md
	// says the bound speaks first.
	program := &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT}
	program.Nodes = append(program.Nodes, &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
			Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
		}},
	})
	length := uint32(4097)
	pad := "0"
	program.Nodes = append(program.Nodes, &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
		Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
			Kind:   irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD,
			Text:   &pad,
			Length: &length,
		}},
	})
	program.RootNode = 1

	v := &validator{bundle: &irv1.RuleBundle{Programs: []*irv1.Program{program}}, used: features.NewSet()}
	err := v.validatePrograms()
	if err == nil {
		t.Fatal("a program with both faults must be refused")
	}
	if !strings.Contains(err.Error(), "4097") {
		t.Fatalf("check 13 runs before check 16, so the bound must speak first; got %v", err)
	}
}
