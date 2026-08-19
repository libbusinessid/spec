package reference

import (
	"fmt"
	"strings"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/artifact"
	"github.com/libbusinessid/spec/internal/limits"
)

// EngineError is a technical failure: it never becomes a business result.
type EngineError struct {
	Detail string
}

// Error implements the error interface.
func (e *EngineError) Error() string { return "engine error: " + e.Detail }

func enginef(format string, args ...any) *EngineError {
	return &EngineError{Detail: fmt.Sprintf(format, args...)}
}

// stringValue is a possibly absent string view.
type stringValue struct {
	text   string
	absent bool
}

func present(s string) stringValue { return stringValue{text: s} }

var absentString = stringValue{absent: true}

// integerValue is a bounded integer that may be indeterminate. An
// indeterminate integer never produces an invalid checksum: it propagates to
// `unsupported`.
type integerValue struct {
	value         int64
	indeterminate bool
}

var indeterminateInteger = integerValue{indeterminate: true}

// checksumOutcome is the tri-state result of a checksum node.
type checksumOutcome struct {
	status        StepStatus
	reason        irv1.ReasonCode
	messageKey    *string
	notApplicable bool
}

var checksumUnsupported = checksumOutcome{
	status: StatusUnsupported,
	reason: irv1.ReasonCode_REASON_CODE_UNSUPPORTED_CHECKSUM,
}

// frame is the execution context of one program invocation.
type frame struct {
	program *irv1.Program
	// value is the canonical value of the identifier under validation, or the
	// value current at that point inside a canonicalization program.
	value string
	// subject is the caller supplied view, if any.
	subject stringValue
	// hasSubject reports whether the caller supplied a subject.
	hasSubject bool
	// country is the canonical country of the selected target.
	country *string
	profile Profile
	// target describes the selected dispatch target, when one exists.
	target *irv1.DispatchTarget
	depth  int
}

// machine executes IR programs with a bounded step budget.
type machine struct {
	rules *artifact.Ruleset
	steps int
}

func (m *machine) tick() error {
	m.steps++
	if m.steps > limits.MaxStepsPerValidation {
		return enginef("evaluation budget of %d steps exhausted", limits.MaxStepsPerValidation)
	}
	return nil
}

// charge bills the code points an operation produced. Without it, a bundle
// whose graph doubles a string at every level would allocate memory
// proportional to the budget times the longest constant; with it, the budget
// bounds the total number of code points a single operation materializes.
func (m *machine) charge(produced int) error {
	if produced <= 0 {
		return nil
	}
	m.steps += (produced + limits.CodePointsPerStep - 1) / limits.CodePointsPerStep
	if m.steps > limits.MaxStepsPerValidation {
		return enginef("evaluation budget of %d steps exhausted", limits.MaxStepsPerValidation)
	}
	return nil
}

func (m *machine) program(id uint32) (*irv1.Program, error) {
	p, ok := m.rules.ProgramByID[id]
	if !ok {
		return nil, enginef("unknown program %d", id)
	}
	return p, nil
}

// evalString evaluates a string node.
func (m *machine) evalString(f *frame, index uint32) (stringValue, error) {
	if err := m.tick(); err != nil {
		return absentString, err
	}
	node, err := m.node(f, index)
	if err != nil {
		return absentString, err
	}
	op := node.GetStringOperation()
	if op == nil {
		return absentString, enginef("node %d is not a string operation", index)
	}
	switch op.GetKind() {
	case irv1.StringOpKind_STRING_OP_KIND_CONSTANT:
		return present(op.GetText()), nil
	case irv1.StringOpKind_STRING_OP_KIND_VALUE:
		return present(f.value), nil
	case irv1.StringOpKind_STRING_OP_KIND_SUBJECT:
		if f.hasSubject {
			return f.subject, nil
		}
		if f.program.SubjectNode != nil {
			return m.evalSubjectNode(f)
		}
		return present(f.value), nil
	case irv1.StringOpKind_STRING_OP_KIND_COUNTRY_CODE:
		if f.country == nil {
			return absentString, nil
		}
		return present(*f.country), nil
	default:
		// Every other constructor reads an operand, handled below.
	}
	first, err := operand(node, 0)
	if err != nil {
		return absentString, err
	}
	value, err := m.evalString(f, first)
	if err != nil {
		return absentString, err
	}
	produced, err := m.produceString(f, node, op, value)
	if err != nil {
		return absentString, err
	}
	if !produced.absent {
		if err := m.charge(len([]rune(produced.text))); err != nil {
			return absentString, err
		}
	}
	return produced, nil
}

// produceString applies the string constructors that read an operand.
func (m *machine) produceString(f *frame, node *irv1.Node, op *irv1.StringOperation,
	value stringValue,
) (stringValue, error) {
	switch op.GetKind() {
	case irv1.StringOpKind_STRING_OP_KIND_SLICE:
		return sliceRunes(value, int(op.GetStart()), int(op.GetEnd())), nil
	case irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM:
		if value.absent {
			return absentString, nil
		}
		runes := []rune(value.text)
		if int(op.GetStart()) > len(runes) {
			return absentString, nil
		}
		return present(string(runes[op.GetStart():])), nil
	case irv1.StringOpKind_STRING_OP_KIND_SLICE_TO:
		return sliceRunes(value, 0, int(op.GetEnd())), nil
	case irv1.StringOpKind_STRING_OP_KIND_BEFORE_FIRST:
		if value.absent {
			return absentString, nil
		}
		i := strings.Index(value.text, op.GetText())
		if i < 0 {
			return absentString, nil
		}
		return present(value.text[:i]), nil
	case irv1.StringOpKind_STRING_OP_KIND_AFTER_FIRST:
		if value.absent {
			return absentString, nil
		}
		i := strings.Index(value.text, op.GetText())
		if i < 0 {
			return absentString, nil
		}
		return present(value.text[i+len(op.GetText()):]), nil
	case irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX:
		if value.absent || !strings.HasPrefix(value.text, op.GetText()) {
			return absentString, nil
		}
		return present(value.text[len(op.GetText()):]), nil
	case irv1.StringOpKind_STRING_OP_KIND_CONCAT:
		var b strings.Builder
		for i, in := range node.GetInputNodes() {
			part := value
			if i > 0 {
				var err error
				part, err = m.evalString(f, in)
				if err != nil {
					return absentString, err
				}
			}
			if part.absent {
				return absentString, nil
			}
			b.WriteString(part.text)
		}
		return present(b.String()), nil
	default:
		return absentString, enginef("unsupported string operation %v", op.GetKind())
	}
}

func (m *machine) evalSubjectNode(f *frame) (stringValue, error) {
	sub := *f
	sub.hasSubject = true
	sub.subject = present(f.value)
	return m.evalString(&sub, f.program.GetSubjectNode())
}

func sliceRunes(v stringValue, start, end int) stringValue {
	if v.absent || start > end {
		return absentString
	}
	runes := []rune(v.text)
	if end > len(runes) {
		return absentString
	}
	return present(string(runes[start:end]))
}

// operand returns the index of the i-th operand of a node. The loader already
// proved the arity, so a missing operand is an internal invariant violation and
// is reported as an engine error rather than dereferenced.
func operand(node *irv1.Node, i int) (uint32, error) {
	inputs := node.GetInputNodes()
	if i >= len(inputs) {
		return 0, enginef("operand %d is missing", i)
	}
	return inputs[i], nil
}

func (m *machine) node(f *frame, index uint32) (*irv1.Node, error) {
	nodes := f.program.GetNodes()
	if int(index) >= len(nodes) {
		return nil, enginef("node index %d out of range", index)
	}
	return nodes[index], nil
}

// evalPredicate evaluates a boolean node.
//
//nolint:gocyclo // one exhaustive evaluation of the closed predicate set.
func (m *machine) evalPredicate(f *frame, index uint32) (bool, error) {
	if err := m.tick(); err != nil {
		return false, err
	}
	node, err := m.node(f, index)
	if err != nil {
		return false, err
	}
	op := node.GetPredicateOperation()
	if op == nil {
		return false, enginef("node %d is not a predicate", index)
	}
	switch op.GetKind() {
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_ALL:
		for _, in := range node.GetInputNodes() {
			ok, err := m.evalPredicate(f, in)
			if err != nil || !ok {
				return false, err
			}
		}
		return true, nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_ANY:
		for _, in := range node.GetInputNodes() {
			ok, err := m.evalPredicate(f, in)
			if err != nil {
				return false, err
			}
			if ok {
				return true, nil
			}
		}
		return false, nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_NOT:
		first, err := operand(node, 0)
		if err != nil {
			return false, err
		}
		ok, err := m.evalPredicate(f, first)
		return !ok, err
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_PROFILE_IS:
		return string(f.profile) == op.GetText(), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_EQUALS:
		leftIndex, err := operand(node, 0)
		if err != nil {
			return false, err
		}
		rightIndex, err := operand(node, 1)
		if err != nil {
			return false, err
		}
		left, err := m.evalString(f, leftIndex)
		if err != nil {
			return false, err
		}
		right, err := m.evalString(f, rightIndex)
		if err != nil {
			return false, err
		}
		if left.absent || right.absent {
			return false, nil
		}
		return left.text == right.text, nil
	default:
		// Every other predicate reads a single string operand, handled below.
	}

	first, err := operand(node, 0)
	if err != nil {
		return false, err
	}
	value, err := m.evalString(f, first)
	if err != nil {
		return false, err
	}
	if op.GetKind() == irv1.PredicateOpKind_PREDICATE_OP_KIND_IS_ABSENT {
		return value.absent, nil
	}
	if value.absent {
		return false, nil
	}
	return evalStringPredicate(op, value.text)
}

// evalStringPredicate evaluates the predicates that read a single present
// string operand.
func evalStringPredicate(op *irv1.PredicateOperation, text string) (bool, error) {
	runes := []rune(text)
	switch op.GetKind() {
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_IS_EMPTY:
		return len(runes) == 0, nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_EQ:
		return len(runes) == int(op.GetLength()), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_IN:
		for _, l := range op.GetLengths() {
			if len(runes) == int(l) {
				return true, nil
			}
		}
		return false, nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_BETWEEN:
		return len(runes) >= int(op.GetMinLength()) && len(runes) <= int(op.GetMaxLength()), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS:
		return allRunes(runes, isASCIIDigit), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_UPPER_LETTERS:
		return allRunes(runes, isASCIIUpper), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_ALPHANUMERIC:
		return allRunes(runes, isASCIIAlnum), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_CHARSET:
		return allRunes(runes, func(r rune) bool { return strings.ContainsRune(op.GetText(), r) }), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_STARTS_WITH:
		return strings.HasPrefix(text, op.GetText()), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_ENDS_WITH:
		return strings.HasSuffix(text, op.GetText()), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_PREFIX_IN:
		for _, prefix := range op.GetValues() {
			if strings.HasPrefix(text, prefix) {
				return true, nil
			}
		}
		return false, nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_CHAR_AT_IN:
		if int(op.GetIndex()) >= len(runes) {
			return false, nil
		}
		return strings.ContainsRune(op.GetText(), runes[op.GetIndex()]), nil
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_CONTAINS:
		return strings.Contains(text, op.GetText()), nil
	default:
		return false, enginef("unsupported predicate %v", op.GetKind())
	}
}

func allRunes(runes []rune, pred func(rune) bool) bool {
	if len(runes) == 0 {
		return false
	}
	for _, r := range runes {
		if !pred(r) {
			return false
		}
	}
	return true
}
