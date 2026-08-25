package lower

import (
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
)

// The sort comparators must be total: two GLOBAL definitions of the same kind
// cannot coexist in a valid bundle, but the comparator is still required to
// answer for that pair, otherwise the sort is not a strict weak ordering.
func TestSortComparatorsAreTotal(t *testing.T) {
	country := func(v string) *string { return &v }
	global := &irv1.IdentifierDefinition{Kind: "vat"}
	otherGlobal := &irv1.IdentifierDefinition{Kind: "vat"}
	be := &irv1.IdentifierDefinition{Kind: "vat", CountryCode: country("BE")}

	if identifierBefore(global, otherGlobal) || identifierBefore(otherGlobal, global) {
		t.Fatal("two GLOBAL definitions are never ordered")
	}
	if !identifierBefore(global, be) || identifierBefore(be, global) {
		t.Fatal("a GLOBAL definition comes first")
	}

	globalTarget := &irv1.DispatchTarget{}
	otherGlobalTarget := &irv1.DispatchTarget{}
	beTarget := &irv1.DispatchTarget{CountryCode: country("BE")}
	if targetBefore(globalTarget, otherGlobalTarget) || targetBefore(otherGlobalTarget, globalTarget) {
		t.Fatal("two GLOBAL targets are never ordered")
	}
	if !targetBefore(globalTarget, beTarget) || targetBefore(beTarget, globalTarget) {
		t.Fatal("a GLOBAL target comes first")
	}
}

func TestReasonCodeByName(t *testing.T) {
	if code, ok := reasonCodeByName("checksum_not_published"); !ok ||
		code != irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED {
		t.Fatalf("unexpected code %v", code)
	}
	if _, ok := reasonCodeByName("nope"); ok {
		t.Fatal("an unknown reason must not resolve")
	}
	if _, ok := reasonCodeByName("unspecified"); ok {
		t.Fatal("the zero value must never resolve")
	}
}

func TestDuplicateIdentifierSortKeyIsRejected(t *testing.T) {
	country := "BE"
	l := &lowerer{bag: newBag()}
	l.rejectDuplicateIdentifiers([]*irv1.IdentifierDefinition{
		{Id: 1, Kind: "vat", CountryCode: &country},
		{Id: 2, Kind: "vat", CountryCode: &country},
	})
	if !l.bag.HasErrors() {
		t.Fatal("two definitions sharing a sort key must be refused")
	}
}
