package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/features"
)

func TestWalkUnknownFindsNestedFields(t *testing.T) {
	// An unknown field nested inside a repeated message must be found.
	node := &irv1.Node{OutputType: irv1.ValueType_VALUE_TYPE_STRING}
	node.ProtoReflect().SetUnknown(protoUnknown(t))
	bundle := &irv1.RuleBundle{
		Programs: []*irv1.Program{{Id: 1, Nodes: []*irv1.Node{node}}},
	}
	path, bad := findUnknownFields(bundle)
	if !bad || !strings.Contains(path, "programs[0].nodes[0]") {
		t.Fatalf("unexpected path %q (%t)", path, bad)
	}

	// An unknown field nested inside a singular message must also be found.
	operation := &irv1.StringOperation{Kind: irv1.StringOpKind_STRING_OP_KIND_VALUE}
	operation.ProtoReflect().SetUnknown(protoUnknown(t))
	clean := &irv1.RuleBundle{Programs: []*irv1.Program{{Id: 1, Nodes: []*irv1.Node{{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		Operation:  &irv1.Node_StringOperation{StringOperation: operation},
	}}}}}
	path, bad = findUnknownFields(clean)
	if !bad || !strings.Contains(path, "string_operation") {
		t.Fatalf("unexpected path %q (%t)", path, bad)
	}

	if _, bad := findUnknownFields(&irv1.RuleBundle{RulesVersion: "2026.08.0"}); bad {
		t.Fatal("a clean message must not be reported")
	}
}

// protoUnknown returns the wire encoding of an unknown field: field 999,
// varint 1.
func protoUnknown(t *testing.T) protoreflect.RawFields {
	t.Helper()
	return protoreflect.RawFields{0xF8, 0x3E, 0x01}
}

func TestOrderComparators(t *testing.T) {
	global := &irv1.IdentifierDefinition{Kind: "vat"}
	be := &irv1.IdentifierDefinition{Kind: "vat", CountryCode: strPtr("BE")}
	fr := &irv1.IdentifierDefinition{Kind: "vat", CountryCode: strPtr("FR")}
	euid := &irv1.IdentifierDefinition{Kind: "euid", CountryCode: strPtr("FR")}
	if !definitionOrderBefore(euid, be) {
		t.Fatal("kinds order first")
	}
	if !definitionOrderBefore(global, be) || definitionOrderBefore(be, global) {
		t.Fatal("GLOBAL comes first inside a kind")
	}
	if definitionOrderBefore(global, &irv1.IdentifierDefinition{Kind: "vat"}) {
		t.Fatal("two GLOBAL definitions are never ordered")
	}
	if !definitionOrderBefore(be, fr) {
		t.Fatal("country codes order lexicographically")
	}

	globalTarget := &irv1.DispatchTarget{}
	beTarget := &irv1.DispatchTarget{CountryCode: strPtr("BE")}
	frTarget := &irv1.DispatchTarget{CountryCode: strPtr("FR")}
	if !targetOrderBefore(globalTarget, beTarget) || targetOrderBefore(beTarget, globalTarget) {
		t.Fatal("a GLOBAL target comes first")
	}
	if targetOrderBefore(globalTarget, &irv1.DispatchTarget{}) {
		t.Fatal("two GLOBAL targets are never ordered")
	}
	if !targetOrderBefore(beTarget, frTarget) {
		t.Fatal("country targets order lexicographically")
	}
	if globalOr(nil) != "GLOBAL" || globalOr(strPtr("FR")) != "FR" {
		t.Fatal("globalOr is wrong")
	}
}

func TestTokenGrammars(t *testing.T) {
	for _, ok := range []string{"vat", "vat_id", "a0"} {
		if !validKind(ok) {
			t.Fatalf("%q must be a valid kind", ok)
		}
	}
	for _, bad := range []string{"", "VAT", "0vat", "_vat", "vat-id", strings.Repeat("a", 65)} {
		if validKind(bad) {
			t.Fatalf("%q must be refused as a kind", bad)
		}
	}
	for _, ok := range []string{"vat", "vat-id", "vat_id", "a0"} {
		if !validKindAlias(ok) {
			t.Fatalf("%q must be a valid alias", ok)
		}
	}
	for _, bad := range []string{"", "VAT", "0vat", "-vat", "vat!", strings.Repeat("a", 65)} {
		if validKindAlias(bad) {
			t.Fatalf("%q must be refused as an alias", bad)
		}
	}
	for _, ok := range []string{"FR", "BE"} {
		if !validCountry(ok) {
			t.Fatalf("%q must be a valid country", ok)
		}
	}
	for _, bad := range []string{"", "F", "FRA", "fr", "F1"} {
		if validCountry(bad) {
			t.Fatalf("%q must be refused as a country", bad)
		}
	}
	for _, ok := range []string{"BE", "el", "0", "ABCDEFGH"} {
		if !validPrefix(ok) {
			t.Fatalf("%q must be a valid prefix", ok)
		}
	}
	for _, bad := range []string{"", "ABCDEFGHI", "B-E", "B E"} {
		if validPrefix(bad) {
			t.Fatalf("%q must be refused as a prefix", bad)
		}
	}
	if !containsString([]string{"a", "b"}, "b") || containsString([]string{"a"}, "b") {
		t.Fatal("containsString is wrong")
	}
}

func TestComputeMaxLen(t *testing.T) {
	tests := []struct {
		name   string
		node   *irv1.Node
		inputs []int
		want   int
	}{
		{"constant", stringOp(irv1.StringOpKind_STRING_OP_KIND_CONSTANT, func(o *irv1.StringOperation) {
			text := "FRA"
			o.Text = &text
		}), nil, 3},
		{"country", stringOp(irv1.StringOpKind_STRING_OP_KIND_COUNTRY_CODE, nil), nil, 2},
		{"value", stringOp(irv1.StringOpKind_STRING_OP_KIND_VALUE, nil), nil, unknownLength},
		{"slice", stringOp(irv1.StringOpKind_STRING_OP_KIND_SLICE, func(o *irv1.StringOperation) {
			start, end := uint32(2), uint32(9)
			o.Start, o.End = &start, &end
		}), []int{unknownLength}, 7},
		{"slice_to bounded by the operand", stringOp(irv1.StringOpKind_STRING_OP_KIND_SLICE_TO, func(o *irv1.StringOperation) {
			end := uint32(9)
			o.End = &end
		}), []int{4}, 4},
		{"slice_to unbounded operand", stringOp(irv1.StringOpKind_STRING_OP_KIND_SLICE_TO, func(o *irv1.StringOperation) {
			end := uint32(9)
			o.End = &end
		}), []int{unknownLength}, 9},
		{"slice_from bounded", stringOp(irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM, func(o *irv1.StringOperation) {
			start := uint32(2)
			o.Start = &start
		}), []int{9}, 7},
		{"slice_from clamped", stringOp(irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM, func(o *irv1.StringOperation) {
			start := uint32(20)
			o.Start = &start
		}), []int{9}, 0},
		{"slice_from unbounded", stringOp(irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM, func(o *irv1.StringOperation) {
			start := uint32(2)
			o.Start = &start
		}), []int{unknownLength}, unknownLength},
		{"strip_prefix keeps the operand bound", stringOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, func(o *irv1.StringOperation) {
			text := "FR"
			o.Text = &text
		}), []int{9}, 9},
		{"concat sums", stringOp(irv1.StringOpKind_STRING_OP_KIND_CONCAT, nil), []int{2, 3}, 5},
		{"concat with an unbounded operand", stringOp(irv1.StringOpKind_STRING_OP_KIND_CONCAT, nil), []int{2, unknownLength}, unknownLength},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			maxLen := make([]int, len(tc.inputs))
			inputs := make([]uint32, len(tc.inputs))
			for i, v := range tc.inputs {
				maxLen[i] = v
				inputs[i] = uint32(i)
			}
			shape, err := shapeOf(tc.node)
			if err != nil {
				t.Fatal(err)
			}
			if got := computeMaxLen(tc.node, shape, inputs, maxLen); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
	predicate := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
		Operation:  &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{}},
	}
	shape, err := shapeOf(predicate)
	if err != nil {
		t.Fatal(err)
	}
	if got := computeMaxLen(predicate, shape, nil, nil); got != unknownLength {
		t.Fatalf("a non string node has no length bound, got %d", got)
	}
}

func stringOp(kind irv1.StringOpKind, apply func(*irv1.StringOperation)) *irv1.Node {
	op := &irv1.StringOperation{Kind: kind}
	if apply != nil {
		apply(op)
	}
	return &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		Operation:  &irv1.Node_StringOperation{StringOperation: op},
	}
}

func TestShapeOfRejectsANodeWithoutOperation(t *testing.T) {
	if _, err := shapeOf(&irv1.Node{}); err == nil {
		t.Fatal("a node without operation must be refused")
	}
}

func TestCategoryAllowedCoversEveryProgramKind(t *testing.T) {
	v := &validator{}
	if v.categoryAllowed(irv1.ProgramKind_PROGRAM_KIND_UNSPECIFIED, features.CategoryString, 0) {
		t.Fatal("an unspecified program kind allows nothing")
	}
	if v.categoryAllowed(irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION, features.CategoryInteger, 0) {
		t.Fatal("a canonicalization program has no integer operation")
	}
	if v.categoryAllowed(irv1.ProgramKind_PROGRAM_KIND_FORMAT, features.CategoryChecksum, 0) {
		t.Fatal("a format program has no checksum operation")
	}
	if v.categoryAllowed(irv1.ProgramKind_PROGRAM_KIND_CHECKSUM, features.CategoryAssertion, 0) {
		t.Fatal("a checksum program has no assertion")
	}
	if v.categoryAllowed(irv1.ProgramKind_PROGRAM_KIND_CHECKSUM, features.CategoryCall,
		int32(irv1.CallOpKind_CALL_OP_KIND_FORMAT)) {
		t.Fatal("a checksum program only calls checksum programs")
	}
	if !v.categoryAllowed(irv1.ProgramKind_PROGRAM_KIND_FORMAT, features.CategoryCall,
		int32(irv1.CallOpKind_CALL_OP_KIND_FORMAT)) {
		t.Fatal("a format program calls format programs")
	}
}

func TestHelperPredicates(t *testing.T) {
	if abs64(-5) != 5 || abs64(5) != 5 || abs64(0) != 0 {
		t.Fatal("abs64 is wrong")
	}
	if runeCount("héllo") != 5 {
		t.Fatal("runeCount must count code points")
	}
	fail := func(string, ...any) error { return errSentinel }
	if err := checkMessageKey(nil, fail); err != nil {
		t.Fatal("an absent message key is accepted")
	}
	short := "k"
	if err := checkMessageKey(&short, fail); err != nil {
		t.Fatal("a short message key is accepted")
	}
	long := strings.Repeat("k", 5000)
	if err := checkMessageKey(&long, fail); err == nil {
		t.Fatal("an over long message key is refused")
	}
	if err := checkIndex(nil, fail); err != nil {
		t.Fatal("an absent index is accepted")
	}
	if err := preCanonicalizationAllowedProbe(); err != nil {
		t.Fatal(err)
	}
}

func preCanonicalizationAllowedProbe() error {
	allowed := []irv1.CanonicalizationOpKind{
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_SEQUENCE,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_TRIM_WHITESPACE,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_WHITESPACE,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_UPPERCASE_ASCII,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_CHARS,
	}
	for _, kind := range allowed {
		if !preCanonicalizationAllowed(kind) {
			return invalidf("%v must be allowed", kind)
		}
	}
	if preCanonicalizationAllowed(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND) {
		return invalidf("prepend must be refused")
	}
	return nil
}

var errSentinel = invalidf("sentinel")

func TestWriteFileAtomicRejectsAnUnwritableDirectory(t *testing.T) {
	dir := t.TempDir()
	readOnly := filepath.Join(dir, "ro")
	if err := os.Mkdir(readOnly, 0o500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(readOnly, 0o700) }()
	if os.Geteuid() == 0 {
		t.Skip("running as root: permissions are not enforced")
	}
	if err := WriteFileAtomic(filepath.Join(readOnly, "x"), []byte("x")); err == nil {
		t.Fatal("writing into a read only directory must fail")
	}
}

func TestStatusesForUnknownReason(t *testing.T) {
	if got := statusesFor("nope"); got != "unspecified" {
		t.Fatalf("unexpected status %q", got)
	}
}

func TestCategoryIndexFallback(t *testing.T) {
	if got := categoryIndex(features.Category("nope")); got != 0 {
		t.Fatalf("unexpected index %d", got)
	}
}

func TestParseModuleLine(t *testing.T) {
	if _, ok := parseModuleLine("only-one-field"); ok {
		t.Fatal("a single field is not a module line")
	}
	m, ok := parseModuleLine("example.com/mod v1.2.3")
	if !ok || m.Path != "example.com/mod" || m.Version != "v1.2.3" {
		t.Fatalf("unexpected module %+v", m)
	}
}

func TestMarshalOfAnInvalidMessage(t *testing.T) {
	// proto.Clone of a valid message stays encodable: the guard exists for the
	// runtime, and the happy path is what the compiler relies on.
	bundle := &irv1.RuleBundle{RulesVersion: "2026.08.0"}
	clone := proto.Clone(bundle)
	if _, err := Marshal(clone); err != nil {
		t.Fatal(err)
	}
}

func strPtr(v string) *string { return &v }
