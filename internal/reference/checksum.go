package reference

import (
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
)

// evalInteger evaluates an integer node. Any impossibility yields an
// indeterminate value, which later propagates to `unsupported`, never to
// `invalid`.
//
//nolint:gocyclo // one exhaustive evaluation of the closed integer set.
func (m *machine) evalInteger(f *frame, index uint32) (integerValue, error) {
	if err := m.tick(); err != nil {
		return indeterminateInteger, err
	}
	node, err := m.node(f, index)
	if err != nil {
		return indeterminateInteger, err
	}
	op := node.GetIntegerOperation()
	if op == nil {
		return indeterminateInteger, enginef("node %d is not an integer operation", index)
	}
	first, err := operand(node, 0)
	if err != nil {
		return indeterminateInteger, err
	}
	switch op.GetKind() {
	case irv1.IntegerOpKind_INTEGER_OP_KIND_DIGITS_TO_INTEGER:
		s, err := m.evalString(f, first)
		if err != nil {
			return indeterminateInteger, err
		}
		return digitsToInteger(s)
	case irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS:
		s, err := m.evalString(f, first)
		if err != nil {
			return indeterminateInteger, err
		}
		return modDigits(s, op.GetModulus())
	case irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM:
		s, err := m.evalString(f, first)
		if err != nil {
			return indeterminateInteger, err
		}
		return weightedSum(s, op)
	case irv1.IntegerOpKind_INTEGER_OP_KIND_MODULO:
		v, err := m.evalInteger(f, first)
		if err != nil || v.indeterminate {
			return indeterminateInteger, err
		}
		mod := op.GetModulus()
		r := v.value % mod
		if r < 0 {
			r += mod
		}
		return integerValue{value: r}, nil
	case irv1.IntegerOpKind_INTEGER_OP_KIND_COMPLEMENT:
		v, err := m.evalInteger(f, first)
		if err != nil || v.indeterminate {
			return indeterminateInteger, err
		}
		if v.value < 0 || v.value > op.GetModulus() {
			return indeterminateInteger, nil
		}
		return integerValue{value: op.GetModulus() - v.value}, nil
	case irv1.IntegerOpKind_INTEGER_OP_KIND_REMAINDER_MAP:
		v, err := m.evalInteger(f, first)
		if err != nil || v.indeterminate {
			return indeterminateInteger, err
		}
		values := op.GetRemainderValues()
		if v.value < 0 || v.value >= int64(len(values)) {
			return indeterminateInteger, nil
		}
		return integerValue{value: values[v.value]}, nil
	default:
		return indeterminateInteger, enginef("unsupported integer operation %v", op.GetKind())
	}
}

func digitsToInteger(s stringValue) (integerValue, error) {
	if s.absent || s.text == "" {
		return indeterminateInteger, nil
	}
	var out int64
	for _, r := range s.text {
		if !isASCIIDigit(r) {
			return indeterminateInteger, nil
		}
		out = out*10 + int64(r-'0')
	}
	return integerValue{value: out}, nil
}

func modDigits(s stringValue, modulus int64) (integerValue, error) {
	if s.absent || s.text == "" {
		return indeterminateInteger, nil
	}
	var r int64
	for _, c := range s.text {
		if !isASCIIDigit(c) {
			return indeterminateInteger, nil
		}
		r = (r*10 + int64(c-'0')) % modulus
	}
	return integerValue{value: r}, nil
}

func mapChar(r rune, mapping irv1.CharMapping) (int64, bool) {
	switch mapping {
	case irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE:
		if isASCIIDigit(r) {
			return int64(r - '0'), true
		}
	case irv1.CharMapping_CHAR_MAPPING_ALNUM_BASE36:
		switch {
		case isASCIIDigit(r):
			return int64(r - '0'), true
		case isASCIIUpper(r):
			return int64(r-'A') + 10, true
		}
	default:
	}
	return 0, false
}

func weightedSum(s stringValue, op *irv1.IntegerOperation) (integerValue, error) {
	if s.absent || s.text == "" {
		return indeterminateInteger, nil
	}
	runes := []rune(s.text)
	weights := op.GetWeights()
	values := make([]int64, len(runes))
	for i, r := range runes {
		v, ok := mapChar(r, op.GetMapping())
		if !ok {
			return indeterminateInteger, nil
		}
		values[i] = v
	}
	var sum int64
	switch op.GetAlignment() {
	case irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT:
		n := minInt(len(values), len(weights))
		for i := 0; i < n; i++ {
			sum += values[i] * weights[i]
		}
	case irv1.WeightAlignment_WEIGHT_ALIGNMENT_RIGHT:
		n := minInt(len(values), len(weights))
		for i := 0; i < n; i++ {
			sum += values[len(values)-1-i] * weights[len(weights)-1-i]
		}
	case irv1.WeightAlignment_WEIGHT_ALIGNMENT_CYCLE:
		for i := range values {
			sum += values[i] * weights[i%len(weights)]
		}
	default:
		return indeterminateInteger, enginef("unsupported weight alignment %v", op.GetAlignment())
	}
	return integerValue{value: sum}, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// luhn applies the Luhn algorithm; the rightmost digit is the check digit.
func luhn(s stringValue) checksumOutcome {
	if s.absent {
		return checksumUnsupported
	}
	runes := []rune(s.text)
	if len(runes) < 2 {
		return checksumUnsupported
	}
	var sum int64
	for i := len(runes) - 1; i >= 0; i-- {
		if !isASCIIDigit(runes[i]) {
			return checksumUnsupported
		}
		d := int64(runes[i] - '0')
		if (len(runes)-1-i)%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	if sum%10 == 0 {
		return checksumOutcome{status: StatusValid, reason: irv1.ReasonCode_REASON_CODE_OK}
	}
	return checksumOutcome{status: StatusInvalid, reason: irv1.ReasonCode_REASON_CODE_INVALID_CHECKSUM}
}

// iso7064Mod97_10 expands ASCII letters to their base 36 decimal value and
// requires the resulting number to be congruent to one modulo 97.
func iso7064Mod9710(s stringValue) checksumOutcome {
	if s.absent {
		return checksumUnsupported
	}
	runes := []rune(s.text)
	if len(runes) < 3 {
		return checksumUnsupported
	}
	var r int64
	for _, c := range runes {
		v, ok := mapChar(c, irv1.CharMapping_CHAR_MAPPING_ALNUM_BASE36)
		if !ok {
			return checksumUnsupported
		}
		if v < 10 {
			r = (r*10 + v) % 97
			continue
		}
		r = (r*100 + v) % 97
	}
	if r == 1 {
		return checksumOutcome{status: StatusValid, reason: irv1.ReasonCode_REASON_CODE_OK}
	}
	return checksumOutcome{status: StatusInvalid, reason: irv1.ReasonCode_REASON_CODE_INVALID_CHECKSUM}
}

// evalChecksum evaluates a checksum node.
//
//nolint:gocyclo // one exhaustive evaluation of the closed checksum set.
func (m *machine) evalChecksum(f *frame, index uint32) (checksumOutcome, error) {
	if err := m.tick(); err != nil {
		return checksumUnsupported, err
	}
	node, err := m.node(f, index)
	if err != nil {
		return checksumUnsupported, err
	}
	if call := node.GetCallOperation(); call != nil {
		return m.callChecksum(f, node, call)
	}
	op := node.GetChecksumOperation()
	if op == nil {
		return checksumUnsupported, enginef("node %d is not a checksum operation", index)
	}
	switch op.GetKind() {
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_UNSUPPORTED:
		return checksumOutcome{
			status:     StatusUnsupported,
			reason:     op.GetReasonCode(),
			messageKey: op.MessageKey,
		}, nil
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN, irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ISO7064_MOD97_10:
		first, err := operand(node, 0)
		if err != nil {
			return checksumUnsupported, err
		}
		s, err := m.evalString(f, first)
		if err != nil {
			return checksumUnsupported, err
		}
		var outcome checksumOutcome
		if op.GetKind() == irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN {
			outcome = luhn(s)
		} else {
			outcome = iso7064Mod9710(s)
		}
		return withMessageKey(outcome, op.MessageKey), nil
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT,
		irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE:
		return m.evalCompare(f, node, op)
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_CONSTANT:
		return m.evalCompareConstant(f, node, op)
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN:
		predicate, err := operand(node, 0)
		if err != nil {
			return checksumUnsupported, err
		}
		branch, err := operand(node, 1)
		if err != nil {
			return checksumUnsupported, err
		}
		ok, err := m.evalPredicate(f, predicate)
		if err != nil {
			return checksumUnsupported, err
		}
		if !ok {
			return checksumOutcome{notApplicable: true}, nil
		}
		return m.evalChecksum(f, branch)
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_CHOOSE:
		for _, in := range node.GetInputNodes() {
			outcome, err := m.evalChecksum(f, in)
			if err != nil {
				return checksumUnsupported, err
			}
			if !outcome.notApplicable {
				return outcome, nil
			}
		}
		return checksumUnsupported, nil
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ALL_CHECKS:
		return m.evalAllChecks(f, node)
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ANY_CHECK:
		return m.evalAnyCheck(f, node)
	default:
		return checksumUnsupported, enginef("unsupported checksum operation %v", op.GetKind())
	}
}

func (m *machine) evalAllChecks(f *frame, node *irv1.Node) (checksumOutcome, error) {
	var firstUnsupported *checksumOutcome
	for _, in := range node.GetInputNodes() {
		outcome, err := m.evalChecksum(f, in)
		if err != nil {
			return checksumUnsupported, err
		}
		switch outcome.status {
		case StatusInvalid:
			return outcome, nil
		case StatusUnsupported:
			if firstUnsupported == nil {
				captured := outcome
				firstUnsupported = &captured
			}
		default:
		}
	}
	if firstUnsupported != nil {
		return *firstUnsupported, nil
	}
	return checksumOutcome{status: StatusValid, reason: irv1.ReasonCode_REASON_CODE_OK}, nil
}

func (m *machine) evalAnyCheck(f *frame, node *irv1.Node) (checksumOutcome, error) {
	var firstUnsupported, firstInvalid *checksumOutcome
	for _, in := range node.GetInputNodes() {
		outcome, err := m.evalChecksum(f, in)
		if err != nil {
			return checksumUnsupported, err
		}
		switch outcome.status {
		case StatusValid:
			return outcome, nil
		case StatusUnsupported:
			if firstUnsupported == nil {
				captured := outcome
				firstUnsupported = &captured
			}
		case StatusInvalid:
			if firstInvalid == nil {
				captured := outcome
				firstInvalid = &captured
			}
		default:
		}
	}
	switch {
	case firstUnsupported != nil:
		return *firstUnsupported, nil
	case firstInvalid != nil:
		return *firstInvalid, nil
	default:
		return checksumUnsupported, nil
	}
}

// evalCompareConstant compares a computed integer against a literal.
//
// COMPARE_DIGIT and COMPARE_SLICE can only compare against part of the value
// being checked, which leaves no way to state that a remainder must equal a
// fixed number. An indeterminate operand stays unsupported, exactly as it does
// for the other comparisons: an integer that could not be evaluated never
// proves an identifier wrong.
func (m *machine) evalCompareConstant(f *frame, node *irv1.Node, op *irv1.ChecksumOperation) (checksumOutcome, error) {
	left, err := operand(node, 0)
	if err != nil {
		return checksumUnsupported, err
	}
	actual, err := m.evalInteger(f, left)
	if err != nil {
		return checksumUnsupported, err
	}
	if actual.indeterminate {
		return checksumUnsupported, nil
	}
	if actual.value == op.GetConstant() {
		return checksumOutcome{status: StatusValid, reason: irv1.ReasonCode_REASON_CODE_OK}, nil
	}
	return withMessageKey(checksumOutcome{
		status: StatusInvalid,
		reason: irv1.ReasonCode_REASON_CODE_INVALID_CHECKSUM,
	}, op.MessageKey), nil
}

func (m *machine) evalCompare(f *frame, node *irv1.Node, op *irv1.ChecksumOperation) (checksumOutcome, error) {
	left, err := operand(node, 0)
	if err != nil {
		return checksumUnsupported, err
	}
	right, err := operand(node, 1)
	if err != nil {
		return checksumUnsupported, err
	}
	expected, err := m.evalInteger(f, left)
	if err != nil {
		return checksumUnsupported, err
	}
	actualText, err := m.evalString(f, right)
	if err != nil {
		return checksumUnsupported, err
	}
	if expected.indeterminate || actualText.absent {
		return checksumUnsupported, nil
	}
	runes := []rune(actualText.text)
	var actual integerValue
	if op.GetKind() == irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT {
		if int(op.GetIndex()) >= len(runes) {
			return checksumUnsupported, nil
		}
		if !isASCIIDigit(runes[op.GetIndex()]) {
			return checksumUnsupported, nil
		}
		actual = integerValue{value: int64(runes[op.GetIndex()] - '0')}
	} else {
		if int(op.GetEnd()) > len(runes) {
			return checksumUnsupported, nil
		}
		actual, err = digitsToInteger(present(string(runes[op.GetStart():op.GetEnd()])))
		if err != nil {
			return checksumUnsupported, err
		}
		if actual.indeterminate {
			return checksumUnsupported, nil
		}
	}
	if actual.value == expected.value {
		return checksumOutcome{status: StatusValid, reason: irv1.ReasonCode_REASON_CODE_OK}, nil
	}
	return withMessageKey(checksumOutcome{
		status: StatusInvalid,
		reason: irv1.ReasonCode_REASON_CODE_INVALID_CHECKSUM,
	}, op.MessageKey), nil
}

func withMessageKey(outcome checksumOutcome, key *string) checksumOutcome {
	if outcome.status != StatusValid && key != nil {
		outcome.messageKey = key
	}
	return outcome
}

func (m *machine) callChecksum(f *frame, node *irv1.Node, call *irv1.CallOperation) (checksumOutcome, error) {
	if f.depth >= 32 {
		return checksumUnsupported, enginef("maximum call depth reached")
	}
	first, err := operand(node, 0)
	if err != nil {
		return checksumUnsupported, err
	}
	view, err := m.evalString(f, first)
	if err != nil {
		return checksumUnsupported, err
	}
	callee, err := m.program(call.GetProgramId())
	if err != nil {
		return checksumUnsupported, err
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
	return m.evalChecksum(sub, callee.GetRootNode())
}

// RunChecksum executes a checksum program at top level.
func (m *machine) RunChecksum(program *irv1.Program, base *frame) (checksumOutcome, error) {
	if err := m.tick(); err != nil {
		return checksumUnsupported, err
	}
	f := *base
	f.program = program
	f.hasSubject = false
	return m.evalChecksum(&f, program.GetRootNode())
}
