package reference

import (
	"strings"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
)

// formatOutcome is the result of a format program.
type formatOutcome struct {
	ok         bool
	reason     irv1.ReasonCode
	messageKey *string
}

// RunFormat executes a format program at top level.
func (m *machine) RunFormat(program *irv1.Program, base *frame) (formatOutcome, error) {
	if err := m.tick(); err != nil {
		return formatOutcome{}, err
	}
	f := *base
	f.program = program
	f.hasSubject = false
	return m.evalAssertion(&f, program.GetRootNode())
}

// evalAssertion evaluates an assertion node in declaration order.
func (m *machine) evalAssertion(f *frame, index uint32) (formatOutcome, error) {
	if err := m.tick(); err != nil {
		return formatOutcome{}, err
	}
	node, err := m.node(f, index)
	if err != nil {
		return formatOutcome{}, err
	}
	if call := node.GetCallOperation(); call != nil {
		return m.callFormat(f, node, call)
	}
	op := node.GetAssertionOperation()
	if op == nil {
		return formatOutcome{}, enginef("node %d is not an assertion", index)
	}
	switch op.GetKind() {
	case irv1.AssertionOpKind_ASSERTION_OP_KIND_SEQUENCE:
		for _, in := range node.GetInputNodes() {
			outcome, err := m.evalAssertion(f, in)
			if err != nil || !outcome.ok {
				return outcome, err
			}
		}
		return formatOutcome{ok: true, reason: irv1.ReasonCode_REASON_CODE_OK}, nil
	case irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE:
		first, err := operand(node, 0)
		if err != nil {
			return formatOutcome{}, err
		}
		ok, err := m.evalPredicate(f, first)
		if err != nil {
			return formatOutcome{}, err
		}
		if ok {
			return formatOutcome{ok: true, reason: irv1.ReasonCode_REASON_CODE_OK}, nil
		}
		return formatOutcome{reason: op.GetReasonCode(), messageKey: op.MessageKey}, nil
	default:
		return formatOutcome{}, enginef("unsupported assertion %v", op.GetKind())
	}
}

func (m *machine) callFormat(f *frame, node *irv1.Node, call *irv1.CallOperation) (formatOutcome, error) {
	if f.depth >= 32 {
		return formatOutcome{}, enginef("maximum call depth reached")
	}
	first, err := operand(node, 0)
	if err != nil {
		return formatOutcome{}, err
	}
	view, err := m.evalString(f, first)
	if err != nil {
		return formatOutcome{}, err
	}
	callee, err := m.program(call.GetProgramId())
	if err != nil {
		return formatOutcome{}, err
	}
	sub := &frame{
		program:    callee,
		value:      f.value,
		subject:    view,
		hasSubject: true,
		country:    f.country,
		profile:    f.profile,
		target:     f.target,
		depth:      f.depth + 1,
	}
	return m.evalAssertion(sub, callee.GetRootNode())
}

// RunCanonicalization applies a canonicalization program to a value.
func (m *machine) RunCanonicalization(program *irv1.Program, base *frame, value string) (string, error) {
	if err := m.tick(); err != nil {
		return "", err
	}
	f := *base
	f.program = program
	f.value = value
	if err := m.applyStep(&f, program.GetRootNode()); err != nil {
		return "", err
	}
	return f.value, nil
}

// applyStep applies one canonicalization step to the current value of the
// frame. Steps are sequential, so nested expressions are always evaluated
// against the value current at that point and nothing is memoized.
func (m *machine) applyStep(f *frame, index uint32) error {
	if err := m.tick(); err != nil {
		return err
	}
	node, err := m.node(f, index)
	if err != nil {
		return err
	}
	op := node.GetCanonicalizationOperation()
	if op == nil {
		return enginef("node %d is not a canonicalization step", index)
	}
	if err := m.applyCanonicalizationStep(f, node, op); err != nil {
		return err
	}
	// The produced value is charged against the budget, so a graph that keeps
	// growing the value cannot allocate more than the budget allows.
	return m.charge(len([]rune(f.value)))
}

//nolint:gocyclo // one exhaustive evaluation of the closed step set.
func (m *machine) applyCanonicalizationStep(f *frame, node *irv1.Node,
	op *irv1.CanonicalizationOperation,
) error {
	switch op.GetKind() {
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_SEQUENCE:
		for _, in := range node.GetInputNodes() {
			if err := m.applyStep(f, in); err != nil {
				return err
			}
		}
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_WHEN:
		first, err := operand(node, 0)
		if err != nil {
			return err
		}
		ok, err := m.evalPredicate(f, first)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		for _, in := range node.GetInputNodes()[1:] {
			if err := m.applyStep(f, in); err != nil {
				return err
			}
		}
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_TRIM_WHITESPACE:
		f.value = strings.TrimFunc(f.value, IsWhitespaceV1)
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_WHITESPACE:
		f.value = strings.Map(func(r rune) rune {
			if IsWhitespaceV1(r) {
				return -1
			}
			return r
		}, f.value)
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_UPPERCASE_ASCII:
		f.value = upperASCII(f.value)
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_CHARS:
		set := op.GetText()
		f.value = strings.Map(func(r rune) rune {
			if strings.ContainsRune(set, r) {
				return -1
			}
			return r
		}, f.value)
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REPLACE_PREFIX:
		if strings.HasPrefix(f.value, op.GetText()) {
			f.value = op.GetReplacement() + f.value[len(op.GetText()):]
		}
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND:
		f.value = op.GetText() + f.value
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_APPEND:
		f.value += op.GetText()
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_INSERT:
		runes := []rune(f.value)
		if int(op.GetIndex()) <= len(runes) {
			f.value = string(runes[:op.GetIndex()]) + op.GetText() + string(runes[op.GetIndex():])
		}
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD:
		runes := []rune(f.value)
		if len(runes) < int(op.GetLength()) {
			f.value = strings.Repeat(op.GetText(), int(op.GetLength())-len(runes)) + f.value
		}
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND_COUNTRY_IF_MISSING:
		f.value = prependCountry(f.value, f.target, f.country)
	default:
		return enginef("unsupported canonicalization step %v", op.GetKind())
	}
	return nil
}

func prependCountry(value string, target *irv1.DispatchTarget, country *string) string {
	if target == nil {
		return value
	}
	for _, prefix := range target.GetAcceptedPrefixes() {
		if strings.HasPrefix(value, prefix) {
			return value
		}
	}
	switch {
	case target.CanonicalPrefix != nil:
		return target.GetCanonicalPrefix() + value
	case country != nil:
		return *country + value
	default:
		return value
	}
}
