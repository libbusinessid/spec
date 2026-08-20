package features_test

import (
	"testing"

	"github.com/libbusinessid/spec/internal/features"

	"google.golang.org/protobuf/reflect/protoreflect"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
)

func enumNames(d protoreflect.EnumDescriptor) map[int32]string {
	out := make(map[int32]string)
	vals := d.Values()
	for i := 0; i < vals.Len(); i++ {
		v := vals.Get(i)
		if v.Number() == 0 {
			continue
		}
		out[int32(v.Number())] = string(v.Name())
	}
	return out
}

func TestCatalogCoversEveryOpcodeEnumValue(t *testing.T) {
	cases := []struct {
		category features.Category
		desc     protoreflect.EnumDescriptor
	}{
		{features.CategoryString, irv1.StringOpKind(0).Descriptor()},
		{features.CategoryInteger, irv1.IntegerOpKind(0).Descriptor()},
		{features.CategoryPredicate, irv1.PredicateOpKind(0).Descriptor()},
		{features.CategoryCanonicalization, irv1.CanonicalizationOpKind(0).Descriptor()},
		{features.CategoryAssertion, irv1.AssertionOpKind(0).Descriptor()},
		{features.CategoryChecksum, irv1.ChecksumOpKind(0).Descriptor()},
		{features.CategoryCall, irv1.CallOpKind(0).Descriptor()},
	}
	for _, tc := range cases {
		want := enumNames(tc.desc)
		got := features.OpsByCategory(tc.category)
		if len(got) != len(want) {
			t.Fatalf("category %s: catalog has %d ops, enum has %d", tc.category, len(got), len(want))
		}
		for _, op := range got {
			name, ok := want[op.Code]
			if !ok {
				t.Fatalf("category %s: catalog op code %d is not an enum value", tc.category, op.Code)
			}
			if name != op.Symbol {
				t.Fatalf("category %s code %d: catalog symbol %q, enum %q", tc.category, op.Code, op.Symbol, name)
			}
			if op.Doc == "" {
				t.Fatalf("op %s has no documented semantics", op.Symbol)
			}
			if len(op.Features) == 0 {
				t.Fatalf("op %s declares no capability", op.Symbol)
			}
			for _, id := range op.Features {
				if !features.Known(id) {
					t.Fatalf("op %s references unknown capability %d", op.Symbol, id)
				}
			}
		}
	}
}

func TestCapabilityRegistryIsSortedAndDocumented(t *testing.T) {
	all := features.All()
	if len(all) == 0 {
		t.Fatal("empty capability registry")
	}
	for i, c := range all {
		if i > 0 && all[i-1].ID >= c.ID {
			t.Fatalf("capability registry is not strictly ascending at index %d", i)
		}
		if c.Name == "" || c.Summary == "" || len(c.Content) == 0 {
			t.Fatalf("capability %d is not fully documented", c.ID)
		}
	}
}

func TestEveryCapabilityIsReachableFromAnOperationOrStructure(t *testing.T) {
	used := map[uint32]bool{
		// Structural capabilities are required by bundle level constructs.
		features.IdentifierDispatchV1: true,
		features.ProfilesV1:           true,
		features.ProvenanceV1:         true,
		features.ProvenanceTierV1:     true,
		features.CapturesAndCallsV1:   true,
	}
	for _, op := range features.Ops() {
		for _, id := range op.Features {
			used[id] = true
		}
	}
	for _, c := range features.All() {
		if !used[c.ID] {
			t.Fatalf("capability %s (%d) is unreachable", c.Name, c.ID)
		}
	}
}

func TestSetSortsAndDeduplicates(t *testing.T) {
	s := features.NewSet()
	s.Add(30, 1, 30, 2)
	got := s.Sorted()
	want := []uint32{1, 2, 30}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
	if !s.Contains(1) || s.Contains(99) {
		t.Fatal("Contains is wrong")
	}
}

func TestLookupOpAndCapability(t *testing.T) {
	if _, ok := features.LookupOp(features.CategoryString, int32(irv1.StringOpKind_STRING_OP_KIND_VALUE)); !ok {
		t.Fatal("value() must be in the catalog")
	}
	if _, ok := features.LookupOp(features.CategoryString, 9999); ok {
		t.Fatal("unknown op must not resolve")
	}
	if c, ok := features.Lookup(features.CoreGraphV1); !ok || c.Name != "CORE_GRAPH_V1" {
		t.Fatal("capability lookup is wrong")
	}
	if _, ok := features.Lookup(9999); ok {
		t.Fatal("unknown capability must not resolve")
	}
}

func TestOperandArity(t *testing.T) {
	concat, _ := features.LookupOp(features.CategoryString, int32(irv1.StringOpKind_STRING_OP_KIND_CONCAT))
	if concat.MinOperands() != 1 || concat.MaxOperands() != 256 {
		t.Fatalf("concat arity is wrong: %d..%d", concat.MinOperands(), concat.MaxOperands())
	}
	if _, ok := concat.OperandType(200); !ok {
		t.Fatal("variadic operand type must resolve")
	}
	not, _ := features.LookupOp(features.CategoryPredicate, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_NOT))
	if not.MinOperands() != 1 || not.MaxOperands() != 1 {
		t.Fatal("not arity is wrong")
	}
	if _, ok := not.OperandType(1); ok {
		t.Fatal("fixed arity op must not resolve an extra operand")
	}
	all, _ := features.LookupOp(features.CategoryPredicate, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ALL))
	if all.MaxOperands() != -1 {
		t.Fatal("all() must be unbounded")
	}
	req, _ := features.LookupOp(features.CategoryAssertion, int32(irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE))
	if !req.RequiresParam(features.ParamReasonCode) || !req.AllowsParam(features.ParamMessageKey) {
		t.Fatal("require parameters are wrong")
	}
	if req.RequiresParam(features.ParamMessageKey) || req.AllowsParam(features.ParamModulus) {
		t.Fatal("require must not accept unrelated parameters")
	}
}
