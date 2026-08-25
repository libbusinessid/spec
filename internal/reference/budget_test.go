package reference

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/limits"
)

// probeRuleset compiles the synthetic rule set exercising every operation.
func probeRuleset(t *testing.T) *artifact.Ruleset {
	t.Helper()
	dir := t.TempDir()
	source, err := os.ReadFile(filepath.Join("testdata", "opcodes.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "opcodes.hcl"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	result, bag := artifact.CompileRules(dir, artifact.CompileOptions{RulesVersion: "2026.08.0"})
	if bag.HasErrors() {
		for _, d := range bag.Sorted() {
			t.Error(d.String())
		}
		t.FailNow()
	}
	return result.Ruleset
}

// TestBudgetExhaustionIsReportedAtEveryDepth runs the probe definition with
// every possible remaining budget. Every evaluation must either complete or
// report an engine error: a truncated evaluation must never produce a business
// result, and no propagation path may swallow the error.
func TestBudgetExhaustionIsReportedAtEveryDepth(t *testing.T) {
	rules := probeRuleset(t)
	var definition *irv1.IdentifierDefinition
	for _, d := range rules.Bundle.GetIdentifiers() {
		if d.GetKind() == "probe" {
			definition = d
		}
	}
	if definition == nil {
		t.Fatal("the probe definition is missing")
	}
	target := rules.DispatcherByKind["probe"].GetTargets()[0]

	run := func(remaining int) error {
		m := &machine{rules: rules, steps: limits.MaxStepsPerValidation - remaining}
		base := &frame{profile: ProfileCompatible, country: definition.CountryCode, target: target}
		canonical, err := m.RunCanonicalization(rules.ProgramByID[definition.GetCanonicalizationProgram()],
			base, "XX01234567")
		if err != nil {
			return err
		}
		base.value = canonical
		if _, err := m.RunFormat(rules.ProgramByID[definition.GetFormatProgram()], base); err != nil {
			return err
		}
		_, err = m.RunChecksum(rules.ProgramByID[definition.GetChecksumProgram()], base)
		return err
	}

	total := 0
	for remaining := 1; remaining <= 4000; remaining++ {
		if err := run(remaining); err == nil {
			total = remaining
			break
		}
	}
	if total == 0 {
		t.Fatal("the probe evaluation never completed within the budget")
	}
	t.Logf("the probe definition needs %d evaluation steps", total)
	for remaining := 0; remaining < total; remaining++ {
		err := run(remaining)
		if err == nil {
			t.Fatalf("a budget of %d steps must not complete an evaluation needing %d", remaining, total)
		}
		if !strings.Contains(err.Error(), "budget") {
			t.Fatalf("budget %d produced an unexpected error: %v", remaining, err)
		}
	}
}

func checksumNode(kind irv1.ChecksumOpKind, inputs ...uint32) *irv1.Node {
	return &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
		InputNodes: inputs,
		Operation:  &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{Kind: kind}},
	}
}

func unsupportedNode(reason irv1.ReasonCode) *irv1.Node {
	return &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
		Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
			Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_UNSUPPORTED, ReasonCode: &reason,
		}},
	}
}

// validNode and invalidNode are Luhn checks over constants, so their outcome is
// fixed and the combinators can be tested in isolation.
func constantNode(text string) *irv1.Node {
	return &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
			Kind: irv1.StringOpKind_STRING_OP_KIND_CONSTANT, Text: &text,
		}},
	}
}

func TestCombinatorsFollowTheNormativeOrder(t *testing.T) {
	// 0 valid Luhn, 1 invalid Luhn, 2 unsupported, 3 letters (indeterminate).
	nodes := []*irv1.Node{
		constantNode("79927398713"),
		constantNode("79927398710"),
		constantNode("ABC"),
		checksumNode(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN, 0),
		checksumNode(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN, 1),
		unsupportedNode(irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED),
		checksumNode(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN, 2),
	}
	const (
		validIdx       = 3
		invalidIdx     = 4
		unsupportedIdx = 5
		indeterminate  = 6
	)
	tests := []struct {
		name   string
		kind   irv1.ChecksumOpKind
		inputs []uint32
		status StepStatus
		reason irv1.ReasonCode
	}{
		{"all_checks all valid", irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ALL_CHECKS,
			[]uint32{validIdx, validIdx}, StatusValid, irv1.ReasonCode_REASON_CODE_OK},
		{"all_checks one invalid", irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ALL_CHECKS,
			[]uint32{unsupportedIdx, invalidIdx}, StatusInvalid, irv1.ReasonCode_REASON_CODE_INVALID_CHECKSUM},
		{"all_checks one unsupported", irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ALL_CHECKS,
			[]uint32{validIdx, unsupportedIdx}, StatusUnsupported, irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED},
		{"any_check one valid", irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ANY_CHECK,
			[]uint32{invalidIdx, validIdx}, StatusValid, irv1.ReasonCode_REASON_CODE_OK},
		{"any_check unsupported wins over invalid", irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ANY_CHECK,
			[]uint32{invalidIdx, unsupportedIdx}, StatusUnsupported, irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED},
		{"any_check all invalid", irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ANY_CHECK,
			[]uint32{invalidIdx, invalidIdx}, StatusInvalid, irv1.ReasonCode_REASON_CODE_INVALID_CHECKSUM},
		{"indeterminate stays unsupported", irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ALL_CHECKS,
			[]uint32{indeterminate}, StatusUnsupported, irv1.ReasonCode_REASON_CODE_UNSUPPORTED_CHECKSUM},
		{"choose falls through", irv1.ChecksumOpKind_CHECKSUM_OP_KIND_CHOOSE,
			[]uint32{unsupportedIdx}, StatusUnsupported, irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			program := append(append([]*irv1.Node(nil), nodes...), checksumNode(tc.kind, tc.inputs...))
			m, f := machineWith(program, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
			got, err := m.evalChecksum(f, uint32(len(program)-1))
			if err != nil {
				t.Fatal(err)
			}
			if got.status != tc.status || got.reason != tc.reason {
				t.Fatalf("got %s/%v, want %s/%v", got.status, got.reason, tc.status, tc.reason)
			}
		})
	}
}

func TestAnyCheckWithoutOperandIsUnsupported(t *testing.T) {
	nodes := []*irv1.Node{
		{
			OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
			Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
				Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ANY_CHECK,
			}},
		},
	}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	got, err := m.evalChecksum(f, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got.status != StatusUnsupported {
		t.Fatalf("an empty any_check must be unsupported, got %s", got.status)
	}
}

func TestCompareEdgeCases(t *testing.T) {
	build := func(kind irv1.ChecksumOpKind, apply func(*irv1.ChecksumOperation), text string, modulus int64) (*machine, *frame, uint32) {
		op := &irv1.ChecksumOperation{Kind: kind}
		apply(op)
		nodes := []*irv1.Node{
			constantNode(text),
			{
				OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
				InputNodes: []uint32{0},
				Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
					Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS, Modulus: &modulus,
				}},
			},
			{
				OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
				InputNodes: []uint32{1, 0},
				Operation:  &irv1.Node_ChecksumOperation{ChecksumOperation: op},
			},
		}
		m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
		return m, f, 2
	}
	index := func(v uint32) func(*irv1.ChecksumOperation) {
		return func(op *irv1.ChecksumOperation) { op.Index = &v }
	}
	slice := func(start, end uint32) func(*irv1.ChecksumOperation) {
		return func(op *irv1.ChecksumOperation) { op.Start, op.End = &start, &end }
	}

	m, f, root := build(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT, index(9), "123", 7)
	got, err := m.evalChecksum(f, root)
	if err != nil || got.status != StatusUnsupported {
		t.Fatalf("an out of range digit index must be unsupported: %v %v", got.status, err)
	}
	// The index is zero based: the position equal to the length is already out
	// of range.
	m, f, root = build(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT, index(3), "123", 7)
	got, err = m.evalChecksum(f, root)
	if err != nil || got.status != StatusUnsupported {
		t.Fatalf("the position equal to the length must be unsupported: %v %v", got.status, err)
	}
	m, f, root = build(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT, index(2), "123", 7)
	got, err = m.evalChecksum(f, root)
	if err != nil || got.status == StatusUnsupported {
		t.Fatalf("the last position must be readable: %v %v", got.status, err)
	}
	m, f, root = build(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT, index(0), "123", 7)
	got, err = m.evalChecksum(f, root)
	if err != nil || got.status != StatusInvalid {
		t.Fatalf("a mismatching digit must be invalid: %v %v", got.status, err)
	}
	m, f, root = build(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE, slice(0, 9), "123", 7)
	got, err = m.evalChecksum(f, root)
	if err != nil || got.status != StatusUnsupported {
		t.Fatalf("an out of range slice must be unsupported: %v %v", got.status, err)
	}
	m, f, root = build(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE, slice(0, 3), "123", 1000)
	got, err = m.evalChecksum(f, root)
	if err != nil || got.status != StatusValid {
		t.Fatalf("123 mod 1000 must equal the slice 123: %v %v", got.status, err)
	}
	// A non digit string makes the integer indeterminate, so the comparison is
	// unsupported and never invalid.
	nodes := []*irv1.Node{
		constantNode("12A"),
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS, Modulus: int64Pointer(97),
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME,
			InputNodes: []uint32{1, 0},
			Operation: &irv1.Node_ChecksumOperation{ChecksumOperation: &irv1.ChecksumOperation{
				Kind: irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT, Index: u32Ptr(0),
			}},
		},
	}
	m, f = machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	got, err = m.evalChecksum(f, 2)
	if err != nil || got.status != StatusUnsupported {
		t.Fatalf("an indeterminate integer must be unsupported: %v %v", got.status, err)
	}
}

func int64Pointer(v int64) *int64 { return &v }

func TestIntegerEdgeCases(t *testing.T) {
	nodes := []*irv1.Node{
		constantNode("100"),
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_DIGITS_TO_INTEGER,
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{1},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_COMPLEMENT, Modulus: int64Pointer(97),
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{1},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_REMAINDER_MAP, RemainderValues: []int64{1, 2},
			}},
		},
	}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	// 100 is outside [0, 97], so the complement is indeterminate.
	got, err := m.evalInteger(f, 2)
	if err != nil || !got.indeterminate {
		t.Fatalf("the complement must be indeterminate: %+v %v", got, err)
	}
	// 100 is outside the remainder table, so the lookup is indeterminate.
	got, err = m.evalInteger(f, 3)
	if err != nil || !got.indeterminate {
		t.Fatalf("the remainder lookup must be indeterminate: %+v %v", got, err)
	}
}

func TestModuloIsEuclidean(t *testing.T) {
	nodes := []*irv1.Node{
		constantNode("5"),
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_DIGITS_TO_INTEGER,
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{1},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind:            irv1.IntegerOpKind_INTEGER_OP_KIND_REMAINDER_MAP,
				RemainderValues: []int64{0, 0, 0, 0, 0, -7},
			}},
		},
		{
			OutputType: irv1.ValueType_VALUE_TYPE_INTEGER,
			InputNodes: []uint32{2},
			Operation: &irv1.Node_IntegerOperation{IntegerOperation: &irv1.IntegerOperation{
				Kind: irv1.IntegerOpKind_INTEGER_OP_KIND_MODULO, Modulus: int64Pointer(3),
			}},
		},
	}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_CHECKSUM)
	got, err := m.evalInteger(f, 3)
	if err != nil {
		t.Fatal(err)
	}
	if got.indeterminate || got.value != 2 {
		t.Fatalf("-7 mod 3 must be 2, got %+v", got)
	}
}

// TestExponentialSharingIsBounded builds a legal DAG whose naive evaluation is
// exponential. The loader accepts it, because every node only references lower
// indices, so the evaluation budget is the only line of defence: the engine
// must report an error instead of hanging or allocating without bound.
func TestExponentialSharingIsBounded(t *testing.T) {
	nodes := []*irv1.Node{constantNode("1")}
	for i := 0; i < 40; i++ {
		nodes = append(nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			InputNodes: []uint32{uint32(i), uint32(i)},
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_CONCAT,
			}},
		})
	}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	done := make(chan error, 1)
	go func() {
		_, err := m.evalString(f, uint32(len(nodes)-1))
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "budget") {
			t.Fatalf("the evaluation budget must stop an exponential graph, got %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("the evaluation did not terminate")
	}
	if m.steps > limits.MaxStepsPerValidation+1 {
		t.Fatalf("the budget was exceeded by %d steps", m.steps-limits.MaxStepsPerValidation)
	}
}

// TestProducedCodePointsAreCharged proves that a graph doubling a long constant
// at every level cannot allocate more than the budget allows: the produced code
// points are billed, so the evaluation stops long before the string explodes.
func TestProducedCodePointsAreCharged(t *testing.T) {
	long := strings.Repeat("X", 4096)
	nodes := []*irv1.Node{constantNode(long)}
	for i := 0; i < 32; i++ {
		nodes = append(nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			InputNodes: []uint32{uint32(i), uint32(i)},
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_CONCAT,
			}},
		})
	}
	m, f := machineWith(nodes, irv1.ProgramKind_PROGRAM_KIND_FORMAT)
	if _, err := m.evalString(f, uint32(len(nodes)-1)); err == nil ||
		!strings.Contains(err.Error(), "budget") {
		t.Fatalf("the budget must stop the doubling graph, got %v", err)
	}
	// The total number of produced code points is bounded by the budget.
	if m.steps > limits.MaxStepsPerValidation+limits.MaxConstantBytes {
		t.Fatalf("the budget overshoot is too large: %d", m.steps)
	}
}

// TestCanonicalizationGrowthIsCharged proves the same for a canonicalization
// sequence that repeats a growing step many times.
func TestCanonicalizationGrowthIsCharged(t *testing.T) {
	long := strings.Repeat("Y", 4096)
	prepend := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
		Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
			Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND, Text: &long,
		}},
	}
	inputs := make([]uint32, 5000)
	sequence := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP,
		InputNodes: inputs,
		Operation: &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: &irv1.CanonicalizationOperation{
			Kind: irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_SEQUENCE,
		}},
	}
	m, f := machineWith([]*irv1.Node{prepend, sequence}, irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION)
	f.program.RootNode = 1
	if _, err := m.RunCanonicalization(f.program, f, "0"); err == nil ||
		!strings.Contains(err.Error(), "budget") {
		t.Fatalf("the budget must stop the growing sequence, got %v", err)
	}
	if len([]rune(f.value)) > limits.MaxStepsPerValidation*limits.CodePointsPerStep {
		t.Fatal("the produced value exceeded the budget bound")
	}
}
