package ast_test

import (
	"testing"

	"github.com/libbusinessid/spec/internal/ast"
	"github.com/libbusinessid/spec/internal/diagnostics"
)

func pos(line int) diagnostics.Position {
	return diagnostics.Position{File: "rules.hcl", Line: line, Column: 1}
}

func TestExpressionPositions(t *testing.T) {
	exprs := []ast.Expr{
		&ast.CallExpr{Name: "luhn", Position: pos(1)},
		&ast.StringLit{Value: "FR", Position: pos(2)},
		&ast.IntLit{Value: 9, Position: pos(3)},
		&ast.BoolLit{Value: true, Position: pos(4)},
		&ast.ListExpr{Position: pos(5)},
		&ast.RefExpr{Parts: []string{"format", "fr", "siren"}, Position: pos(6)},
	}
	for i, e := range exprs {
		if e.Pos().Line != i+1 {
			t.Fatalf("expression %d reports the wrong position %+v", i, e.Pos())
		}
	}
	ref := exprs[5].(*ast.RefExpr)
	if ref.String() != "format.fr.siren" {
		t.Fatalf("unexpected reference rendering %q", ref.String())
	}
	if (&ast.RefExpr{}).String() != "" {
		t.Fatal("an empty reference renders as an empty string")
	}
}

func TestWalkVisitsEverySubExpression(t *testing.T) {
	tree := &ast.CallExpr{
		Name: "require",
		Args: []ast.Expr{
			&ast.CallExpr{Name: "length_in", Args: []ast.Expr{
				&ast.CallExpr{Name: "subject"},
				&ast.ListExpr{Items: []ast.Expr{&ast.IntLit{Value: 9}, &ast.IntLit{Value: 10}}},
			}},
			&ast.StringLit{Value: "invalid_length"},
		},
	}
	var seen int
	ast.Walk(tree, func(ast.Expr) { seen++ })
	if seen != 7 {
		t.Fatalf("expected seven nodes, visited %d", seen)
	}
	ast.Walk(nil, func(ast.Expr) { t.Fatal("a nil expression must not be visited") })
}

func TestSymbolsAndUnitAccessors(t *testing.T) {
	unit := &ast.Unit{Files: []*ast.File{
		{
			Path:           "a.hcl",
			Canonicalizers: []*ast.Canonicalizer{{Namespace: "vat", Name: "common"}},
			Formats:        []*ast.Format{{Namespace: "fr", Name: "siren"}},
			Checksums:      []*ast.Checksum{{Namespace: "fr", Name: "siren"}},
			Dispatchers:    []*ast.Dispatcher{{Kind: "vat"}},
			Identifiers: []*ast.Identifier{
				{Kind: "vat", CountryCode: "BE"},
				{Kind: "lei", Global: true},
			},
		},
	}}
	if unit.Canonicalizers()[0].Symbol() != "canonicalizer.vat.common" {
		t.Fatal("wrong canonicalizer symbol")
	}
	if unit.Formats()[0].Symbol() != "format.fr.siren" {
		t.Fatal("wrong format symbol")
	}
	if unit.Checksums()[0].Symbol() != "checksum.fr.siren" {
		t.Fatal("wrong checksum symbol")
	}
	if unit.Dispatchers()[0].Symbol() != "dispatcher.vat" {
		t.Fatal("wrong dispatcher symbol")
	}
	ids := unit.Identifiers()
	if ids[0].Symbol() != "identifier.vat.BE" || ids[1].Symbol() != "identifier.lei.GLOBAL" {
		t.Fatalf("wrong identifier symbols: %q %q", ids[0].Symbol(), ids[1].Symbol())
	}
}
