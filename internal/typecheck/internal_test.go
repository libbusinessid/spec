package typecheck

import (
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/ast"
	"github.com/entid-org/spec/internal/diagnostics"
	"github.com/entid-org/spec/internal/features"
)

// These white-box tests exercise the defensive branches of the parameter
// binder. They are unreachable through the surface language, which is exactly
// why a future signature table mistake must be reported instead of silently
// dropping a parameter.

func TestSetUintRejectsAnUnsupportedParameter(t *testing.T) {
	c := &checker{bag: diagnostics.New()}
	node := &Node{}
	if c.setUint(node, features.ParamModulus, 1, &ast.IntLit{}) {
		t.Fatal("an unsupported integer parameter must be refused")
	}
	if !c.bag.HasErrors() {
		t.Fatal("the refusal must be reported")
	}
}

func TestSetEnumRejectsAnUnsupportedParameter(t *testing.T) {
	c := &checker{bag: diagnostics.New()}
	node := &Node{}
	if c.setEnum(node, features.ParamText, "left", &ast.StringLit{}) {
		t.Fatal("an unsupported enum parameter must be refused")
	}
	if !c.bag.HasErrors() {
		t.Fatal("the refusal must be reported")
	}
}

func TestBindArgRejectsAnUnknownSlotAndParameter(t *testing.T) {
	c := &checker{bag: diagnostics.New()}
	op, _ := features.LookupOp(features.CategoryChecksum, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN))
	node := &Node{Op: op}
	call := &ast.CallExpr{Name: "luhn"}
	cx := &ctx{kind: irv1.ProgramKind_PROGRAM_KIND_CHECKSUM}

	if c.bindArg(cx, node, op, argSlot{kind: argKind(99)}, &ast.StringLit{}, call) {
		t.Fatal("an unknown slot kind must be refused")
	}
	if c.bindArg(cx, node, op, argSlot{kind: argString, param: features.ParamModulus}, &ast.StringLit{}, call) {
		t.Fatal("a string bound to a non string parameter must be refused")
	}
	if c.bindArg(cx, node, op, argSlot{kind: argIntList, param: features.ParamText},
		&ast.ListExpr{Items: []ast.Expr{&ast.IntLit{Value: 1}}}, call) {
		t.Fatal("an integer list bound to a text parameter must be refused")
	}
	if !c.bag.HasErrors() {
		t.Fatal("every refusal must be reported")
	}
}

func TestBindArgVariadicSlotIsANoOp(t *testing.T) {
	c := &checker{bag: diagnostics.New()}
	op, _ := features.LookupOp(features.CategoryChecksum, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_CHOOSE))
	node := &Node{Op: op}
	if !c.bindArg(&ctx{}, node, op, argSlot{kind: argVariadicOperand}, &ast.StringLit{}, &ast.CallExpr{}) {
		t.Fatal("the variadic slot is consumed by the caller and must succeed")
	}
}

func TestOperandRejectsAnExtraPosition(t *testing.T) {
	c := &checker{bag: diagnostics.New()}
	op, _ := features.LookupOp(features.CategoryPredicate, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_NOT))
	if got := c.operand(&ctx{}, op, 5, &ast.CallExpr{}); got != nil {
		t.Fatal("a position beyond the arity must be refused")
	}
	if !c.bag.HasErrors() {
		t.Fatal("the refusal must be reported")
	}
}

func TestTypeNameFallback(t *testing.T) {
	if got := typeName(irv1.ValueType(99)); got != "type(99)" {
		t.Fatalf("unexpected name %q", got)
	}
	if got := typeName(irv1.ValueType_VALUE_TYPE_INTEGER); got != "IntExpr" {
		t.Fatalf("unexpected name %q", got)
	}
}

func TestExprRejectsAnUnknownNode(t *testing.T) {
	c := &checker{bag: diagnostics.New()}
	if got := c.expr(&ctx{}, nil, irv1.ValueType_VALUE_TYPE_STRING); got != nil {
		t.Fatal("a nil expression produces no node")
	}
}

func TestFinishNodeRejectsAWrongArity(t *testing.T) {
	c := &checker{bag: diagnostics.New()}
	op, _ := features.LookupOp(features.CategoryPredicate, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_NOT))
	node := &Node{Op: op}
	if c.finishNode(node, &ast.CallExpr{Name: "not"}) {
		t.Fatal("a missing operand must be refused")
	}
	node.Inputs = []*Node{{Op: op}, {Op: op}}
	if c.finishNode(node, &ast.CallExpr{Name: "not"}) {
		t.Fatal("an extra operand must be refused")
	}
	if !c.bag.HasErrors() {
		t.Fatal("every refusal must be reported")
	}
}

func TestMaxIntAndAbs(t *testing.T) {
	if maxInt(2, 5) != 5 || maxInt(5, 2) != 5 {
		t.Fatal("maxInt is wrong")
	}
	if abs64(-3) != 3 || abs64(3) != 3 {
		t.Fatal("abs64 is wrong")
	}
}
