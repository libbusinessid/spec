package reference

import (
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/artifact"
	"github.com/libbusinessid/spec/internal/limits"
)

// These white-box tests cover the defensive branches of the interpreter. They
// are unreachable through a validated bundle, which is exactly why they must be
// exercised explicitly: a future change to the loader must never turn one of
// them into a silent wrong answer.

func machineWith(nodes []*irv1.Node, kind irv1.ProgramKind) (*machine, *frame) {
	program := &irv1.Program{Id: 1, Kind: kind, Nodes: nodes}
	m := &machine{rules: &artifact.Ruleset{
		Bundle:      &irv1.RuleBundle{},
		ProgramByID: map[uint32]*irv1.Program{1: program},
	}}
	return m, &frame{program: program, value: "VALUE", profile: ProfileCompatible}
}

func stringNode(kind irv1.StringOpKind, inputs ...uint32) *irv1.Node {
	return &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		InputNodes: inputs,
		Operation:  &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{Kind: kind}},
	}
}

func TestEvalStringRejectsAForeignNode(t *testing.T) {
	m, f := machineWith([]*irv1.Node{{OutputType: irv1.ValueType_VALUE_TYPE_STRING}},
		irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalString(f, 0); err == nil || !strings.Contains(err.Error(), "not a string operation") {
		t.Fatalf("unexpected error %v", err)
	}
	if _, err := m.evalString(f, 9); err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestEvalStringRejectsAnUnknownOpcode(t *testing.T) {
	nodes := []*irv1.Node{
		stringNode(irv1.StringOpKind_STRING_OP_KIND_VALUE),
		stringNode(irv1.StringOpKind(120), 0),
	}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalString(f, 1); err == nil || !strings.Contains(err.Error(), "unsupported string operation") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestMissingOperandIsAnEngineError(t *testing.T) {
	nodes := []*irv1.Node{stringNode(irv1.StringOpKind_STRING_OP_KIND_SLICE)}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalString(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	equals := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
		InputNodes: []uint32{0},
		Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
			Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_EQUALS,
		}},
	}
	m, f = machineWith([]*irv1.Node{stringNode(irv1.StringOpKind_STRING_OP_KIND_VALUE), equals},
		irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalPredicate(f, 1); err == nil || !strings.Contains(err.Error(), "operand 1 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	notNode := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
		Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
			Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_NOT,
		}},
	}
	m, f = machineWith([]*irv1.Node{notNode}, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalPredicate(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	lengthNode := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
		Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
			Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_EQ,
		}},
	}
	m, f = machineWith([]*irv1.Node{lengthNode}, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalPredicate(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	intNode := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
		Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
			Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_DIGITS_TO_INTEGER,
		}},
	}
	m, f = machineWith([]*irv1.Node{intNode}, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	if _, err := m.evalInteger(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	luhnNode := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
		Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
			Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN,
		}},
	}
	m, f = machineWith([]*irv1.Node{luhnNode}, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	if _, err := m.evalChecksum(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	whenNode := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
		Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
			Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN,
		}},
	}
	m, f = machineWith([]*irv1.Node{whenNode}, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	if _, err := m.evalChecksum(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	compareNode := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
		Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
			Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT, Index: u32Ptr(0),
		}},
	}
	m, f = machineWith([]*irv1.Node{compareNode}, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	if _, err := m.evalChecksum(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	requireNode := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
		Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{
			Kind: irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE,
		}},
	}
	m, f = machineWith([]*irv1.Node{requireNode}, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalAssertion(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	whenStep := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
		Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
			Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_WHEN,
		}},
	}
	m, f = machineWith([]*irv1.Node{whenStep}, irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION)
	if err := m.applyStep(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	callNode := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
		Operation: &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{
			Kind: irv1.CallOpKind_CALL_OP_KIND_CHECKSUM, ProgramId: 1,
		}},
	}
	m, f = machineWith([]*irv1.Node{callNode}, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	if _, err := m.evalChecksum(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
	formatCall := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
		Operation: &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{
			Kind: irv1.CallOpKind_CALL_OP_KIND_FORMAT, ProgramId: 1,
		}},
	}
	m, f = machineWith([]*irv1.Node{formatCall}, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalAssertion(f, 0); err == nil || !strings.Contains(err.Error(), "operand 0 is missing") {
		t.Fatalf("unexpected error %v", err)
	}
}

func u32Ptr(v uint32) *uint32 { return &v }

func TestEvalPredicateRejectsAForeignNode(t *testing.T) {
	m, f := machineWith([]*irv1.Node{{OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN}},
		irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalPredicate(f, 0); err == nil || !strings.Contains(err.Error(), "not a predicate") {
		t.Fatalf("unexpected error %v", err)
	}
	if _, err := m.evalPredicate(f, 5); err == nil {
		t.Fatal("an out of range index must fail")
	}
}

func TestEvalPredicateRejectsAnUnknownOpcode(t *testing.T) {
	nodes := []*irv1.Node{
		stringNode(irv1.StringOpKind_STRING_OP_KIND_VALUE),
		{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
				Kind: irv1.PredicateOpKind(120),
			}},
		},
	}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalPredicate(f, 1); err == nil || !strings.Contains(err.Error(), "unsupported predicate") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestEvalIntegerAndChecksumRejectForeignNodes(t *testing.T) {
	m, f := machineWith([]*irv1.Node{{OutputType: irv1.ValueType_VALUE_TYPE_INTEGER}},
		irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	if _, err := m.evalInteger(f, 0); err == nil || !strings.Contains(err.Error(), "not an integer operation") {
		t.Fatalf("unexpected error %v", err)
	}
	if _, err := m.evalInteger(f, 7); err == nil {
		t.Fatal("an out of range index must fail")
	}
	if _, err := m.evalChecksum(f, 0); err == nil || !strings.Contains(err.Error(), "not a checksum operation") {
		t.Fatalf("unexpected error %v", err)
	}
	if _, err := m.evalChecksum(f, 7); err == nil {
		t.Fatal("an out of range index must fail")
	}
}

func TestEvalIntegerAndChecksumRejectUnknownOpcodes(t *testing.T) {
	nodes := []*irv1.Node{
		stringNode(irv1.StringOpKind_STRING_OP_KIND_VALUE),
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind: irv1.IntegerOpKind(120),
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
				Kind: irv1.ChecksumOpKind(120),
			}},
		},
	}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	if _, err := m.evalInteger(f, 1); err == nil || !strings.Contains(err.Error(), "unsupported integer operation") {
		t.Fatalf("unexpected error %v", err)
	}
	if _, err := m.evalChecksum(f, 2); err == nil || !strings.Contains(err.Error(), "unsupported checksum operation") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestAssertionAndCanonicalizationRejectForeignNodes(t *testing.T) {
	m, f := machineWith([]*irv1.Node{{OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION}},
		irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalAssertion(f, 0); err == nil || !strings.Contains(err.Error(), "not an assertion") {
		t.Fatalf("unexpected error %v", err)
	}
	if _, err := m.evalAssertion(f, 4); err == nil {
		t.Fatal("an out of range index must fail")
	}
	f.program.Nodes[0].Operation = &irv1.Node_AssertionOperation{
		AssertionOperation: &irv1.AssertionOperation{Kind: irv1.AssertionOpKind(120)},
	}
	if _, err := m.evalAssertion(f, 0); err == nil || !strings.Contains(err.Error(), "unsupported assertion") {
		t.Fatalf("unexpected error %v", err)
	}

	m, f = machineWith([]*irv1.Node{{OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP}},
		irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION)
	if err := m.applyStep(f, 0); err == nil || !strings.Contains(err.Error(), "not a canonicalization step") {
		t.Fatalf("unexpected error %v", err)
	}
	if err := m.applyStep(f, 3); err == nil {
		t.Fatal("an out of range index must fail")
	}
	f.program.Nodes[0].Operation = &irv1.Node_CanonicalizationOperation{
		CanonicalizationOperation: &irv1.CanonicalizationOperation{Kind: irv1.CanonicalizationOpKind(120)},
	}
	if err := m.applyStep(f, 0); err == nil || !strings.Contains(err.Error(), "unsupported canonicalization step") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestEvaluationBudgetIsEnforced(t *testing.T) {
	m, f := machineWith([]*irv1.Node{stringNode(irv1.StringOpKind_STRING_OP_KIND_VALUE)},
		irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	m.steps = limits.MaxStepsPerValidation
	if _, err := m.evalString(f, 0); err == nil || !strings.Contains(err.Error(), "budget") {
		t.Fatalf("unexpected error %v", err)
	}
	m.steps = limits.MaxStepsPerValidation
	if _, err := m.evalPredicate(f, 0); err == nil {
		t.Fatal("the budget must stop the predicate evaluation")
	}
	m.steps = limits.MaxStepsPerValidation
	if _, err := m.evalInteger(f, 0); err == nil {
		t.Fatal("the budget must stop the integer evaluation")
	}
	m.steps = limits.MaxStepsPerValidation
	if _, err := m.evalChecksum(f, 0); err == nil {
		t.Fatal("the budget must stop the checksum evaluation")
	}
	m.steps = limits.MaxStepsPerValidation
	if _, err := m.evalAssertion(f, 0); err == nil {
		t.Fatal("the budget must stop the assertion evaluation")
	}
	m.steps = limits.MaxStepsPerValidation
	if err := m.applyStep(f, 0); err == nil {
		t.Fatal("the budget must stop the canonicalization")
	}
	m.steps = limits.MaxStepsPerValidation
	if _, err := m.RunFormat(f.program, f); err == nil {
		t.Fatal("the budget must stop a program invocation")
	}
	m.steps = limits.MaxStepsPerValidation
	if _, err := m.RunChecksum(f.program, f); err == nil {
		t.Fatal("the budget must stop a checksum invocation")
	}
	m.steps = limits.MaxStepsPerValidation
	if _, err := m.RunCanonicalization(f.program, f, "x"); err == nil {
		t.Fatal("the budget must stop a canonicalization invocation")
	}
}

func TestCallDepthIsBounded(t *testing.T) {
	nodes := []*irv1.Node{
		stringNode(irv1.StringOpKind_STRING_OP_KIND_VALUE),
		{
			OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{
				Kind: irv1.CallOpKind_CALL_OP_KIND_CHECKSUM, ProgramId: 1,
			}},
		},
	}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	f.depth = 32
	if _, err := m.evalChecksum(f, 1); err == nil || !strings.Contains(err.Error(), "call depth") {
		t.Fatalf("unexpected error %v", err)
	}
	assertNodes := []*irv1.Node{
		stringNode(irv1.StringOpKind_STRING_OP_KIND_VALUE),
		{
			OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{
				Kind: irv1.CallOpKind_CALL_OP_KIND_FORMAT, ProgramId: 1,
			}},
		},
	}
	m, f = machineWith(assertNodes, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	f.depth = 32
	if _, err := m.evalAssertion(f, 1); err == nil || !strings.Contains(err.Error(), "call depth") {
		t.Fatalf("unexpected error %v", err)
	}
	f.depth = 0
	m.rules.ProgramByID = map[uint32]*irv1.Program{}
	if _, err := m.evalAssertion(f, 1); err == nil || !strings.Contains(err.Error(), "unknown program") {
		t.Fatalf("unexpected error %v", err)
	}
	m2, f2 := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	m2.rules.ProgramByID = map[uint32]*irv1.Program{}
	if _, err := m2.evalChecksum(f2, 1); err == nil || !strings.Contains(err.Error(), "unknown program") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestChecksumPrimitives(t *testing.T) {
	if got := luhn(absentString); got.status != StatusUnsupported {
		t.Fatalf("an absent view must be unsupported, got %v", got.status)
	}
	if got := luhn(present("7")); got.status != StatusUnsupported {
		t.Fatalf("a single digit must be unsupported, got %v", got.status)
	}
	if got := luhn(present("12A4")); got.status != StatusUnsupported {
		t.Fatalf("a non digit must be unsupported, got %v", got.status)
	}
	if got := luhn(present("79927398713")); got.status != StatusValid {
		t.Fatalf("the published Luhn vector must be valid, got %v", got.status)
	}
	if got := luhn(present("79927398710")); got.status != StatusInvalid {
		t.Fatalf("a mutated Luhn vector must be invalid, got %v", got.status)
	}
	if got := iso7064Mod9710(absentString); got.status != StatusUnsupported {
		t.Fatal("an absent view must be unsupported")
	}
	if got := iso7064Mod9710(present("12")); got.status != StatusUnsupported {
		t.Fatal("a short view must be unsupported")
	}
	if got := iso7064Mod9710(present("12-4")); got.status != StatusUnsupported {
		t.Fatal("a non alphanumeric view must be unsupported")
	}
	if got := iso7064Mod9710(present("5493001KJTIIGC8Y1R12")); got.status != StatusValid {
		t.Fatalf("the published LEI example must be valid, got %v", got.status)
	}
	if got := iso7064Mod9710(present("5493001KJTIIGC8Y1R13")); got.status != StatusInvalid {
		t.Fatal("a mutated LEI must be invalid")
	}
}

func TestIntegerPrimitives(t *testing.T) {
	if v, _ := digitsToInteger(absentString); !v.indeterminate {
		t.Fatal("an absent view is indeterminate")
	}
	if v, _ := digitsToInteger(present("")); !v.indeterminate {
		t.Fatal("an empty view is indeterminate")
	}
	if v, _ := digitsToInteger(present("12A")); !v.indeterminate {
		t.Fatal("a non digit is indeterminate")
	}
	if v, _ := digitsToInteger(present("00123")); v.value != 123 {
		t.Fatalf("unexpected value %d", v.value)
	}
	if v, _ := modDigits(absentString, 97); !v.indeterminate {
		t.Fatal("an absent view is indeterminate")
	}
	if v, _ := modDigits(present("12A"), 97); !v.indeterminate {
		t.Fatal("a non digit is indeterminate")
	}
	if v, _ := modDigits(present("1234567"), 97); v.value != 1234567%97 {
		t.Fatalf("unexpected remainder %d", v.value)
	}
}

func TestWeightedSumAlignments(t *testing.T) {
	op := func(alignment irv1.WeightAlignment, mapping irv1.CharMapping, weights ...int64) *irv1.IntegerOperation {
		return &irv1.IntegerOperation{
			Kind:    irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM,
			Weights: weights, Alignment: &alignment, Mapping: &mapping,
		}
	}
	digit := irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE
	alnum := irv1.CharMapping_CHAR_MAPPING_ALNUM_BASE36

	got, _ := weightedSum(present("1234"), op(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT, digit, 1, 2, 3, 4))
	if got.value != 1*1+2*2+3*3+4*4 {
		t.Fatalf("left alignment is wrong: %d", got.value)
	}
	got, _ = weightedSum(present("12"), op(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT, digit, 1, 2, 3, 4))
	if got.value != 1*1+2*2 {
		t.Fatalf("a shorter operand must pair only the leading positions: %d", got.value)
	}
	got, _ = weightedSum(present("12"), op(irv1.WeightAlignment_WEIGHT_ALIGNMENT_RIGHT, digit, 1, 2, 3, 4))
	if got.value != 1*3+2*4 {
		t.Fatalf("right alignment is wrong: %d", got.value)
	}
	got, _ = weightedSum(present("1234"), op(irv1.WeightAlignment_WEIGHT_ALIGNMENT_CYCLE, digit, 1, 2))
	if got.value != 1*1+2*2+3*1+4*2 {
		t.Fatalf("cycle alignment is wrong: %d", got.value)
	}
	got, _ = weightedSum(present("Z1"), op(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT, alnum, 1, 1))
	if got.value != 35+1 {
		t.Fatalf("base 36 mapping is wrong: %d", got.value)
	}
	if got, _ = weightedSum(present("z1"), op(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT, alnum, 1, 1)); !got.indeterminate {
		t.Fatal("a lower case letter is outside the mapping domain")
	}
	if got, _ = weightedSum(present("A"), op(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT, digit, 1)); !got.indeterminate {
		t.Fatal("a letter is outside the digit mapping domain")
	}
	if got, _ = weightedSum(absentString, op(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT, digit, 1)); !got.indeterminate {
		t.Fatal("an absent view is indeterminate")
	}
	if got, _ = weightedSum(present("1"), op(irv1.WeightAlignment_WEIGHT_ALIGNMENT_UNSPECIFIED, digit, 1)); !got.indeterminate {
		t.Fatal("an unspecified alignment must not silently compute")
	}
}

func TestPrependCountryWithoutTarget(t *testing.T) {
	if got := prependCountry("X", nil, nil); got != "X" {
		t.Fatalf("without a target the value is unchanged, got %q", got)
	}
	country := "BE"
	target := &irv1.DispatchTarget{}
	if got := prependCountry("X", target, &country); got != "BEX" {
		t.Fatalf("the country must be prepended, got %q", got)
	}
	if got := prependCountry("X", target, nil); got != "X" {
		t.Fatalf("without a country the value is unchanged, got %q", got)
	}
}

func TestWhitespaceTableIsFrozen(t *testing.T) {
	table := WhitespaceV1()
	if len(table) != 26 {
		t.Fatalf("the frozen table must hold 26 code points, got %d", len(table))
	}
	for i := 1; i < len(table); i++ {
		if table[i-1] >= table[i] {
			t.Fatal("the table must be sorted")
		}
	}
	for _, r := range []rune{0x0009, 0x00A0, 0x2028, 0xFEFF} {
		if !IsWhitespaceV1(r) {
			t.Fatalf("U+%04X must belong to the table", r)
		}
	}
	for _, r := range []rune{'a', '0', 0x200B} {
		if IsWhitespaceV1(r) {
			t.Fatalf("U+%04X must not belong to the table", r)
		}
	}
}

func TestASCIIHelpers(t *testing.T) {
	if upperASCII("aéz") != "AéZ" || upperASCII("AZ") != "AZ" {
		t.Fatal("uppercase_ascii must only map a..z")
	}
	if lowerASCII("AéZ") != "aéz" || lowerASCII("az") != "az" {
		t.Fatal("lowercase must only map A..Z")
	}
	if trimASCII(" \t\r\nx  ") != "x " {
		t.Fatalf("trim ASCII must only remove the ASCII blanks, got %q", trimASCII(" \t\r\nx  "))
	}
}

func TestChecksumOutcomeMessageKey(t *testing.T) {
	key := "k"
	valid := withMessageKey(checksumOutcome{status: StatusValid}, &key)
	if valid.messageKey != nil {
		t.Fatal("a valid outcome carries no message key")
	}
	invalid := withMessageKey(checksumOutcome{status: StatusInvalid}, &key)
	if invalid.messageKey == nil || *invalid.messageKey != "k" {
		t.Fatal("a failing outcome keeps its message key")
	}
}

func TestProfileOfFallsBackToTheDefinition(t *testing.T) {
	definition := &irv1.IdentifierDefinition{DefaultProfile: "strict_current"}
	if got := profileOf(Options{}, definition); got != ProfileStrictCurrent {
		t.Fatalf("unexpected profile %q", got)
	}
	if got := profileOf(Options{Profile: ProfileCompatible}, definition); got != ProfileCompatible {
		t.Fatalf("the caller option must win, got %q", got)
	}
	if got := profileOf(Options{}, nil); got != ProfileCompatible {
		t.Fatalf("unexpected fallback %q", got)
	}
	if got := profileOf(Options{}, &irv1.IdentifierDefinition{}); got != ProfileCompatible {
		t.Fatalf("unexpected fallback %q", got)
	}
}

func TestKindAndCountryTokens(t *testing.T) {
	for _, ok := range []string{"vat", "vat_id", "vat-id", "a0"} {
		if !validKindToken(ok) {
			t.Fatalf("%q must be a valid kind token", ok)
		}
	}
	for _, bad := range []string{"", "VAT", "0vat", "_vat", strings.Repeat("a", 65), "vat!"} {
		if validKindToken(bad) {
			t.Fatalf("%q must be refused", bad)
		}
	}
	if !validCountryToken("FR") || validCountryToken("fr") || validCountryToken("FRA") {
		t.Fatal("the country token grammar is wrong")
	}
}

// viewProgram builds a format program returning the emptiness of a string
// expression, so that every absence rule can be observed directly.
func viewProgram(nodes []*irv1.Node) (*machine, *frame) {
	return machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
}

func textOp(kind irv1.StringOpKind, apply func(*irv1.StringOperation), inputs ...uint32) *irv1.Node {
	op := &irv1.StringOperation{Kind: kind}
	if apply != nil {
		apply(op)
	}
	return &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		InputNodes: inputs,
		Operation:  &irv1.Node_StringOperation{StringOperation: op},
	}
}

func text(v string) func(*irv1.StringOperation) {
	return func(op *irv1.StringOperation) { op.Text = &v }
}

func bounds(start, end uint32) func(*irv1.StringOperation) {
	return func(op *irv1.StringOperation) { op.Start, op.End = &start, &end }
}

func TestStringViewAbsence(t *testing.T) {
	constant := func(v string) *irv1.Node {
		return textOp(irv1.StringOpKind_STRING_OP_KIND_CONSTANT, text(v))
	}
	tests := []struct {
		name   string
		nodes  []*irv1.Node
		absent bool
		value  string
	}{
		{"strip_prefix without the prefix",
			[]*irv1.Node{constant("ABC"), textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("Z"), 0)},
			true, ""},
		{"strip_prefix with the prefix",
			[]*irv1.Node{constant("ABC"), textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("A"), 0)},
			false, "BC"},
		{"before_first without the delimiter",
			[]*irv1.Node{constant("ABC"), textOp(irv1.StringOpKind_STRING_OP_KIND_BEFORE_FIRST, text("."), 0)},
			true, ""},
		{"after_first without the delimiter",
			[]*irv1.Node{constant("ABC"), textOp(irv1.StringOpKind_STRING_OP_KIND_AFTER_FIRST, text("."), 0)},
			true, ""},
		{"slice_from beyond the end",
			[]*irv1.Node{constant("AB"), textOp(irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM, func(op *irv1.StringOperation) {
				start := uint32(9)
				op.Start = &start
			}, 0)},
			true, ""},
		{"slice beyond the end",
			[]*irv1.Node{constant("AB"), textOp(irv1.StringOpKind_STRING_OP_KIND_SLICE, bounds(0, 9), 0)},
			true, ""},
		{"slice with an inverted range",
			[]*irv1.Node{constant("ABCD"), textOp(irv1.StringOpKind_STRING_OP_KIND_SLICE, bounds(3, 1), 0)},
			true, ""},
		{"slice_to beyond the end",
			[]*irv1.Node{constant("AB"), textOp(irv1.StringOpKind_STRING_OP_KIND_SLICE_TO, func(op *irv1.StringOperation) {
				end := uint32(9)
				op.End = &end
			}, 0)},
			true, ""},
		{"absence propagates through a view",
			[]*irv1.Node{
				constant("ABC"),
				textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("Z"), 0),
				textOp(irv1.StringOpKind_STRING_OP_KIND_SLICE, bounds(0, 1), 1),
			}, true, ""},
		{"absence propagates through before_first",
			[]*irv1.Node{
				constant("ABC"),
				textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("Z"), 0),
				textOp(irv1.StringOpKind_STRING_OP_KIND_BEFORE_FIRST, text("."), 1),
			}, true, ""},
		{"absence propagates through after_first",
			[]*irv1.Node{
				constant("ABC"),
				textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("Z"), 0),
				textOp(irv1.StringOpKind_STRING_OP_KIND_AFTER_FIRST, text("."), 1),
			}, true, ""},
		{"absence propagates through slice_from",
			[]*irv1.Node{
				constant("ABC"),
				textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("Z"), 0),
				textOp(irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM, func(op *irv1.StringOperation) {
					start := uint32(0)
					op.Start = &start
				}, 1),
			}, true, ""},
		{"absence propagates through strip_prefix",
			[]*irv1.Node{
				constant("ABC"),
				textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("Z"), 0),
				textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("A"), 1),
			}, true, ""},
		{"concat with an absent operand",
			[]*irv1.Node{
				constant("ABC"),
				textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("Z"), 0),
				textOp(irv1.StringOpKind_STRING_OP_KIND_CONCAT, nil, 0, 1),
			}, true, ""},
		{"concat of present operands",
			[]*irv1.Node{
				constant("AB"), constant("CD"),
				textOp(irv1.StringOpKind_STRING_OP_KIND_CONCAT, nil, 0, 1),
			}, false, "ABCD"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, f := viewProgram(tc.nodes)
			got, err := m.evalString(f, uint32(len(tc.nodes)-1))
			if err != nil {
				t.Fatal(err)
			}
			if got.absent != tc.absent {
				t.Fatalf("absence: got %t, want %t", got.absent, tc.absent)
			}
			if !tc.absent && got.text != tc.value {
				t.Fatalf("value: got %q, want %q", got.text, tc.value)
			}
		})
	}
}

func TestPredicatesOnAbsentAndEdgeValues(t *testing.T) {
	nodes := []*irv1.Node{
		textOp(irv1.StringOpKind_STRING_OP_KIND_CONSTANT, text("ABC")),
		textOp(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX, text("Z"), 0),
		textOp(irv1.StringOpKind_STRING_OP_KIND_CONSTANT, text("")),
	}
	const (
		presentIdx = 0
		absentIdx  = 1
		emptyIdx   = 2
	)
	predicate := func(kind irv1.PredicateOpKind, apply func(*irv1.PredicateOperation), inputs ...uint32) *irv1.Node {
		op := &irv1.PredicateOperation{Kind: kind}
		if apply != nil {
			apply(op)
		}
		return &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: inputs,
			Operation:  &irv1.Node_PredicateOperation{PredicateOperation: op},
		}
	}
	index := func(v uint32) func(*irv1.PredicateOperation) {
		return func(op *irv1.PredicateOperation) {
			op.Index = &v
			set := "ABC"
			op.Text = &set
		}
	}
	tests := []struct {
		name string
		node *irv1.Node
		want bool
	}{
		{"is_absent on an absent view",
			predicate(irv1.PredicateOpKind_PREDICATE_OP_KIND_IS_ABSENT, nil, absentIdx), true},
		{"is_empty on an absent view",
			predicate(irv1.PredicateOpKind_PREDICATE_OP_KIND_IS_EMPTY, nil, absentIdx), false},
		{"is_empty on an empty view",
			predicate(irv1.PredicateOpKind_PREDICATE_OP_KIND_IS_EMPTY, nil, emptyIdx), true},
		{"ascii_digits on an empty view",
			predicate(irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS, nil, emptyIdx), false},
		{"equals with an absent operand",
			predicate(irv1.PredicateOpKind_PREDICATE_OP_KIND_EQUALS, nil, presentIdx, absentIdx), false},
		{"equals of identical views",
			predicate(irv1.PredicateOpKind_PREDICATE_OP_KIND_EQUALS, nil, presentIdx, presentIdx), true},
		{"char_at_in beyond the end",
			predicate(irv1.PredicateOpKind_PREDICATE_OP_KIND_CHAR_AT_IN, index(9), presentIdx), false},
		{"char_at_in inside the value",
			predicate(irv1.PredicateOpKind_PREDICATE_OP_KIND_CHAR_AT_IN, index(0), presentIdx), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			program := append(append([]*irv1.Node(nil), nodes...), tc.node)
			m, f := viewProgram(program)
			got, err := m.evalPredicate(f, uint32(len(program)-1))
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestCountryCodeIsAbsentForAGlobalTarget(t *testing.T) {
	nodes := []*irv1.Node{textOp(irv1.StringOpKind_STRING_OP_KIND_COUNTRY_CODE, nil)}
	m, f := viewProgram(nodes)
	got, err := m.evalString(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !got.absent {
		t.Fatal("a GLOBAL target has no country code")
	}
	country := "FR"
	f.country = &country
	got, err = m.evalString(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.absent || got.text != "FR" {
		t.Fatalf("unexpected country %+v", got)
	}
}

func TestSubjectFallsBackToTheProgramSubjectNode(t *testing.T) {
	nodes := []*irv1.Node{
		textOp(irv1.StringOpKind_STRING_OP_KIND_VALUE, nil),
		textOp(irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM, func(op *irv1.StringOperation) {
			start := uint32(2)
			op.Start = &start
		}, 0),
		textOp(irv1.StringOpKind_STRING_OP_KIND_SUBJECT, nil),
	}
	m, f := viewProgram(nodes)
	subject := uint32(1)
	f.program.SubjectNode = &subject
	f.value = "BE0123"
	got, err := m.evalString(f, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got.absent || got.text != "0123" {
		t.Fatalf("unexpected subject %+v", got)
	}
}
