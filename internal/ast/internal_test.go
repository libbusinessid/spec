package ast

import "testing"

// TestExpressionInterfaceIsClosed exercises the marker method that keeps the
// Expr interface closed to this package: no foreign type can be smuggled into
// the typed AST.
func TestExpressionInterfaceIsClosed(t *testing.T) {
	want := map[string]Expr{
		"call":      &CallExpr{},
		"string":    &StringLit{},
		"integer":   &IntLit{},
		"boolean":   &BoolLit{},
		"list":      &ListExpr{},
		"reference": &RefExpr{},
	}
	for label, e := range want {
		if got := e.exprKind(); got != label {
			t.Fatalf("got %q, want %q", got, label)
		}
		if got := ExprKind(e); got != label {
			t.Fatalf("ExprKind returned %q, want %q", got, label)
		}
	}
	if got := ExprKind(nil); got != "none" {
		t.Fatalf("a nil expression must be labelled none, got %q", got)
	}
}
