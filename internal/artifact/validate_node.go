package artifact

import (
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/features"
	"github.com/libbusinessid/spec/internal/limits"
)

// present records which parameter fields a node actually carries.
type present map[features.Param]bool

// nodeShape is the decoded, category independent view of one node.
type nodeShape struct {
	category features.Category
	code     int32
	params   present
}

func shapeOf(n *irv1.Node) (nodeShape, error) {
	switch op := n.GetOperation().(type) {
	case *irv1.Node_StringOperation:
		s := op.StringOperation
		p := present{}
		mark(p, features.ParamText, s.Text != nil)
		mark(p, features.ParamStart, s.Start != nil)
		mark(p, features.ParamEnd, s.End != nil)
		return nodeShape{features.CategoryString, int32(s.GetKind()), p}, nil
	case *irv1.Node_IntegerOperation:
		s := op.IntegerOperation
		p := present{}
		mark(p, features.ParamModulus, s.Modulus != nil)
		mark(p, features.ParamWeights, len(s.GetWeights()) > 0)
		mark(p, features.ParamAlignment, s.Alignment != nil)
		mark(p, features.ParamMapping, s.Mapping != nil)
		mark(p, features.ParamRemainderValues, len(s.GetRemainderValues()) > 0)
		return nodeShape{features.CategoryInteger, int32(s.GetKind()), p}, nil
	case *irv1.Node_PredicateOperation:
		s := op.PredicateOperation
		p := present{}
		mark(p, features.ParamText, s.Text != nil)
		mark(p, features.ParamValues, len(s.GetValues()) > 0)
		mark(p, features.ParamLengths, len(s.GetLengths()) > 0)
		mark(p, features.ParamLength, s.Length != nil)
		mark(p, features.ParamMinLength, s.MinLength != nil)
		mark(p, features.ParamMaxLength, s.MaxLength != nil)
		mark(p, features.ParamIndex, s.Index != nil)
		mark(p, features.ParamConstant, s.Constant != nil)
		return nodeShape{features.CategoryPredicate, int32(s.GetKind()), p}, nil
	case *irv1.Node_CanonicalizationOperation:
		s := op.CanonicalizationOperation
		p := present{}
		mark(p, features.ParamText, s.Text != nil)
		mark(p, features.ParamReplacement, s.Replacement != nil)
		mark(p, features.ParamIndex, s.Index != nil)
		mark(p, features.ParamLength, s.Length != nil)
		return nodeShape{features.CategoryCanonicalization, int32(s.GetKind()), p}, nil
	case *irv1.Node_AssertionOperation:
		s := op.AssertionOperation
		p := present{}
		mark(p, features.ParamReasonCode, s.ReasonCode != nil)
		mark(p, features.ParamMessageKey, s.MessageKey != nil)
		return nodeShape{features.CategoryAssertion, int32(s.GetKind()), p}, nil
	case *irv1.Node_ChecksumOperation:
		s := op.ChecksumOperation
		p := present{}
		mark(p, features.ParamIndex, s.Index != nil)
		mark(p, features.ParamStart, s.Start != nil)
		mark(p, features.ParamEnd, s.End != nil)
		mark(p, features.ParamConstant, s.Constant != nil)
		mark(p, features.ParamReasonCode, s.ReasonCode != nil)
		mark(p, features.ParamMessageKey, s.MessageKey != nil)
		return nodeShape{features.CategoryChecksum, int32(s.GetKind()), p}, nil
	case *irv1.Node_CallOperation:
		s := op.CallOperation
		return nodeShape{features.CategoryCall, int32(s.GetKind()), present{features.ParamProgramID: true}}, nil
	default:
		return nodeShape{}, invalidf("node carries no operation")
	}
}

func mark(p present, param features.Param, ok bool) {
	if ok {
		p[param] = true
	}
}

func (v *validator) validateNode(p *irv1.Program, index int, n *irv1.Node, maxLen []int) error {
	shape, err := shapeOf(n)
	if err != nil {
		return invalidf("program %d node %d: %v", p.GetId(), index, err)
	}
	op, ok := features.LookupOp(shape.category, shape.code)
	if !ok {
		return invalidf("program %d node %d: unknown %s operation %d",
			p.GetId(), index, shape.category, shape.code)
	}
	v.used.Add(op.Features...)
	if n.GetOutputType() != op.Output {
		return invalidf("program %d node %d: declares output %v but %s produces %v",
			p.GetId(), index, n.GetOutputType(), op.Symbol, op.Output)
	}
	if !v.categoryAllowed(p.GetKind(), shape.category, shape.code) {
		return invalidf("program %d node %d: %s is not allowed in a %v program",
			p.GetId(), index, op.Symbol, p.GetKind())
	}
	inputs := n.GetInputNodes()
	if len(inputs) < op.MinOperands() {
		return invalidf("program %d node %d: %s expects at least %d operand(s), got %d",
			p.GetId(), index, op.Symbol, op.MinOperands(), len(inputs))
	}
	if upper := op.MaxOperands(); upper >= 0 && len(inputs) > upper {
		return invalidf("program %d node %d: %s accepts at most %d operand(s), got %d",
			p.GetId(), index, op.Symbol, upper, len(inputs))
	}
	for i, in := range inputs {
		if int(in) >= index {
			return invalidf("program %d node %d: operand %d references node %d which is not lower",
				p.GetId(), index, i, in)
		}
		want, _ := op.OperandType(i)
		if p.GetNodes()[in].GetOutputType() != want {
			return invalidf("program %d node %d: operand %d has type %v, %s expects %v",
				p.GetId(), index, i, p.GetNodes()[in].GetOutputType(), op.Symbol, want)
		}
	}
	for param := range shape.params {
		if !op.AllowsParam(param) {
			return invalidf("program %d node %d: %s must not carry the parameter %q",
				p.GetId(), index, op.Symbol, param)
		}
	}
	for _, param := range op.Required {
		if !shape.params[param] {
			return invalidf("program %d node %d: %s requires the parameter %q",
				p.GetId(), index, op.Symbol, param)
		}
	}
	if err := v.validateNodeValues(p, index, n, shape, maxLen); err != nil {
		return err
	}
	maxLen[index] = computeMaxLen(n, shape, inputs, maxLen)
	return nil
}

func (v *validator) categoryAllowed(kind irv1.ProgramKind, category features.Category, code int32) bool {
	switch kind {
	case irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION:
		switch category {
		case features.CategoryString, features.CategoryPredicate, features.CategoryCanonicalization:
			return category != features.CategoryString ||
				code != int32(irv1.StringOpKind_STRING_OP_KIND_SUBJECT)
		default:
			return false
		}
	case irv1.ProgramKind_PROGRAM_KIND_FORMAT:
		switch category {
		case features.CategoryString, features.CategoryPredicate, features.CategoryAssertion:
			return true
		case features.CategoryCall:
			return code == int32(irv1.CallOpKind_CALL_OP_KIND_FORMAT)
		default:
			return false
		}
	case irv1.ProgramKind_PROGRAM_KIND_CHECKSUM:
		switch category {
		case features.CategoryString, features.CategoryPredicate,
			features.CategoryInteger, features.CategoryChecksum:
			return true
		case features.CategoryCall:
			return code == int32(irv1.CallOpKind_CALL_OP_KIND_CHECKSUM)
		default:
			return false
		}
	default:
		return false
	}
}

//nolint:gocyclo // one exhaustive bound check per operation family.
func (v *validator) validateNodeValues(p *irv1.Program, index int, n *irv1.Node, shape nodeShape, maxLen []int) error {
	fail := func(format string, args ...any) error {
		return invalidf("program %d node %d: "+format, append([]any{p.GetId(), index}, args...)...)
	}
	switch shape.category {
	case features.CategoryString:
		s := n.GetStringOperation()
		if s.Text != nil {
			if len(s.GetText()) > limits.MaxConstantBytes {
				return fail("constant exceeds %d bytes", limits.MaxConstantBytes)
			}
			if s.GetKind() != irv1.StringOpKind_STRING_OP_KIND_CONSTANT && s.GetText() == "" {
				return fail("the constant argument must not be empty")
			}
		}
		if err := checkIndex(s.Start, fail); err != nil {
			return err
		}
		if err := checkIndex(s.End, fail); err != nil {
			return err
		}
		if s.GetKind() == irv1.StringOpKind_STRING_OP_KIND_SLICE && s.GetStart() > s.GetEnd() {
			return fail("slice start %d is greater than end %d", s.GetStart(), s.GetEnd())
		}
	case features.CategoryInteger:
		if err := v.validateIntegerNode(n, shape, maxLen, fail); err != nil {
			return err
		}
	case features.CategoryPredicate:
		if err := validatePredicateNode(n, fail); err != nil {
			return err
		}
	case features.CategoryCanonicalization:
		if err := validateCanonicalizationNode(n, fail); err != nil {
			return err
		}
		c := n.GetCanonicalizationOperation()
		if c.GetKind() == irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND_COUNTRY_IF_MISSING {
			v.prependCountryPrograms[p.GetId()] = true
		}
		if p.GetKind() == irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION {
			v.canonicalizationKinds[p.GetId()] = append(v.canonicalizationKinds[p.GetId()], c.GetKind())
		}
	case features.CategoryAssertion:
		a := n.GetAssertionOperation()
		if a.GetKind() == irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE &&
			!assertionReasonAllowed(a.GetReasonCode()) {
			return fail("reason code %v cannot prove a format invalidity", a.GetReasonCode())
		}
		if err := checkMessageKey(a.MessageKey, fail); err != nil {
			return err
		}
	case features.CategoryChecksum:
		if err := validateChecksumNode(p, n, fail); err != nil {
			return err
		}
	case features.CategoryCall:
		call := n.GetCallOperation()
		if call.GetProgramId() == 0 {
			return fail("call operation references program 0")
		}
		v.calls[p.GetId()] = append(v.calls[p.GetId()], call.GetProgramId())
	}
	return nil
}

func (v *validator) validateIntegerNode(n *irv1.Node, shape nodeShape, maxLen []int, fail func(string, ...any) error) error {
	s := n.GetIntegerOperation()
	if s.Modulus != nil && (s.GetModulus() < limits.MinModulus || s.GetModulus() > limits.MaxModulus) {
		return fail("modulus %d is outside %d..%d", s.GetModulus(), limits.MinModulus, limits.MaxModulus)
	}
	switch s.GetKind() {
	case irv1.IntegerOpKind_INTEGER_OP_KIND_DIGITS_TO_INTEGER:
		bound := maxLen[n.GetInputNodes()[0]]
		if bound == unknownLength || bound > limits.MaxDigitsToIntegerLength {
			return fail("digits_to_integer needs a provable bound of at most %d digits",
				limits.MaxDigitsToIntegerLength)
		}
	case irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM:
		return validateWeightedSum(s, maxLen[n.GetInputNodes()[0]], fail)
	case irv1.IntegerOpKind_INTEGER_OP_KIND_REMAINDER_MAP:
		// Presence of the field already implies at least one entry.
		if len(s.GetRemainderValues()) > limits.MaxRemainderValues {
			return fail("%d remainder values exceed the limit of %d",
				len(s.GetRemainderValues()), limits.MaxRemainderValues)
		}
	default:
	}
	_ = shape
	return nil
}

// validateWeightedSum proves that a weighted sum cannot overflow.
func validateWeightedSum(s *irv1.IntegerOperation, operandBound int, fail func(string, ...any) error) error {
	// Presence of the field already implies at least one weight, so only the
	// upper bound can be violated here.
	if len(s.GetWeights()) > limits.MaxWeights {
		return fail("%d weights exceed the limit of %d", len(s.GetWeights()), limits.MaxWeights)
	}
	positions := int64(len(s.GetWeights()))
	maxWeight := int64(0)
	for _, w := range s.GetWeights() {
		if w < -limits.MaxWeightMagnitude || w > limits.MaxWeightMagnitude {
			return fail("weight %d exceeds the magnitude limit of %d", w, limits.MaxWeightMagnitude)
		}
		if abs64(w) > maxWeight {
			maxWeight = abs64(w)
		}
	}
	if s.GetAlignment() == irv1.WeightAlignment_WEIGHT_ALIGNMENT_UNSPECIFIED {
		return fail("weighted_sum has an unspecified alignment")
	}
	if s.GetMapping() == irv1.CharMapping_CHAR_MAPPING_UNSPECIFIED {
		return fail("weighted_sum has an unspecified mapping")
	}
	if s.GetAlignment() == irv1.WeightAlignment_WEIGHT_ALIGNMENT_CYCLE {
		if operandBound == unknownLength {
			return fail("a cycling weighted_sum needs a statically bounded operand")
		}
		positions = int64(operandBound)
	}
	if maxWeight != 0 && positions > (1<<62)/(maxWeight*35) {
		return fail("weighted_sum can overflow a signed 64 bit integer")
	}
	return nil
}

func validatePredicateNode(n *irv1.Node, fail func(string, ...any) error) error {
	s := n.GetPredicateOperation()
	// INTEGER_IS carries a constant, like COMPARE_CONSTANT does. Only the
	// checksum side was bounded when the two opcodes landed, which left a bundle
	// free to state a comparison no checked expression could ever reach.
	if err := checkConstant(s.Constant, fail); err != nil {
		return err
	}
	if err := validatePredicateBounds(s, fail); err != nil {
		return err
	}
	if s.Text != nil {
		if len(s.GetText()) > limits.MaxConstantBytes {
			return fail("constant exceeds %d bytes", limits.MaxConstantBytes)
		}
		if s.GetText() == "" {
			return fail("the constant argument must not be empty")
		}
	}
	switch s.GetKind() {
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_CHARSET,
		irv1.PredicateOpKind_PREDICATE_OP_KIND_CHAR_AT_IN:
		for i := 0; i < len(s.GetText()); i++ {
			if s.GetText()[i] >= 0x80 {
				return fail("the character set must be ASCII")
			}
		}
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_PROFILE_IS:
		if s.GetText() != "compatible" && s.GetText() != "strict_current" {
			return fail("unknown profile %q", s.GetText())
		}
	default:
	}
	for _, value := range s.GetValues() {
		if value == "" {
			return fail("prefix_in must not carry an empty value")
		}
		if len(value) > limits.MaxConstantBytes {
			return fail("constant exceeds %d bytes", limits.MaxConstantBytes)
		}
	}
	return nil
}

// validatePredicateBounds checks every numeric parameter of a predicate.
func validatePredicateBounds(s *irv1.PredicateOperation, fail func(string, ...any) error) error {
	for _, v := range []*uint32{s.Index, s.Length, s.MinLength, s.MaxLength} {
		if err := checkIndex(v, fail); err != nil {
			return err
		}
	}
	for _, l := range s.GetLengths() {
		if l > limits.MaxIndex {
			return fail("length %d exceeds %d", l, limits.MaxIndex)
		}
	}
	if s.GetKind() == irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_BETWEEN &&
		s.GetMinLength() > s.GetMaxLength() {
		return fail("length_between minimum %d is greater than maximum %d", s.GetMinLength(), s.GetMaxLength())
	}
	return nil
}

func validateCanonicalizationNode(n *irv1.Node, fail func(string, ...any) error) error {
	s := n.GetCanonicalizationOperation()
	if err := checkIndex(s.Index, fail); err != nil {
		return err
	}
	if err := checkIndex(s.Length, fail); err != nil {
		return err
	}
	for _, text := range []*string{s.Text, s.Replacement} {
		if text != nil && len(*text) > limits.MaxConstantBytes {
			return fail("constant exceeds %d bytes", limits.MaxConstantBytes)
		}
	}
	if s.Text != nil && *s.Text == "" {
		return fail("the constant argument must not be empty")
	}
	switch s.GetKind() {
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REPLACE_PREFIX:
		if s.GetText() == s.GetReplacement() {
			return fail("replace_prefix replaces a prefix by itself")
		}
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD:
		if runeCount(s.GetText()) != 1 {
			return fail("left_pad expects exactly one padding code point")
		}
		if s.GetLength() == 0 {
			return fail("left_pad length must be at least 1")
		}
		// The target length is bounded like every other slice bound. Without an
		// upper bound an interpreter is still protected by the step budget, but
		// an engine that compiles the rules ahead of time sizes its buffer from
		// this number and has nothing to stop it.
		if s.GetLength() > limits.MaxIndex {
			return fail("left_pad length %d exceeds the limit of %d", s.GetLength(), limits.MaxIndex)
		}
	default:
	}
	return nil
}

func validateChecksumNode(p *irv1.Program, n *irv1.Node, fail func(string, ...any) error) error {
	s := n.GetChecksumOperation()
	if err := checkIndex(s.Index, fail); err != nil {
		return err
	}
	if err := checkIndex(s.Start, fail); err != nil {
		return err
	}
	if err := checkIndex(s.End, fail); err != nil {
		return err
	}
	if err := checkConstant(s.Constant, fail); err != nil {
		return err
	}
	if err := checkMessageKey(s.MessageKey, fail); err != nil {
		return err
	}
	switch s.GetKind() {
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE:
		if s.GetStart() >= s.GetEnd() {
			return fail("compare_slice start %d must be lower than end %d", s.GetStart(), s.GetEnd())
		}
		if int(s.GetEnd()-s.GetStart()) > limits.MaxDigitsToIntegerLength {
			return fail("compare_slice covers more than %d digits", limits.MaxDigitsToIntegerLength)
		}
	case irv1.ChecksumOpKind_CHECKSUM_OP_KIND_UNSUPPORTED:
		if !unsupportedReasonAllowed(s.GetReasonCode()) {
			return fail("reason code %v is not an unsupported checksum reason", s.GetReasonCode())
		}
	default:
	}
	if s.GetKind() != irv1.ChecksumOpKind_CHECKSUM_OP_KIND_CHOOSE {
		for _, in := range n.GetInputNodes() {
			child := p.GetNodes()[in].GetChecksumOperation()
			if child != nil && child.GetKind() == irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN {
				return fail("a WHEN branch is only accepted as a direct operand of CHOOSE")
			}
		}
	}
	return nil
}

func computeMaxLen(n *irv1.Node, shape nodeShape, inputs []uint32, maxLen []int) int {
	if shape.category != features.CategoryString {
		return unknownLength
	}
	s := n.GetStringOperation()
	switch s.GetKind() {
	case irv1.StringOpKind_STRING_OP_KIND_CONSTANT:
		return runeCount(s.GetText())
	case irv1.StringOpKind_STRING_OP_KIND_COUNTRY_CODE:
		return 2
	case irv1.StringOpKind_STRING_OP_KIND_SLICE:
		return int(s.GetEnd() - s.GetStart())
	case irv1.StringOpKind_STRING_OP_KIND_SLICE_TO:
		out := int(s.GetEnd())
		if in := maxLen[inputs[0]]; in != unknownLength && in < out {
			out = in
		}
		return out
	case irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM:
		if in := maxLen[inputs[0]]; in != unknownLength {
			out := in - int(s.GetStart())
			if out < 0 {
				out = 0
			}
			return out
		}
		return unknownLength
	case irv1.StringOpKind_STRING_OP_KIND_BEFORE_FIRST,
		irv1.StringOpKind_STRING_OP_KIND_AFTER_FIRST,
		irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX:
		return maxLen[inputs[0]]
	case irv1.StringOpKind_STRING_OP_KIND_CONCAT:
		total := 0
		for _, in := range inputs {
			if maxLen[in] == unknownLength {
				return unknownLength
			}
			total += maxLen[in]
		}
		return total
	default:
		return unknownLength
	}
}

func checkIndex(v *uint32, fail func(string, ...any) error) error {
	if v != nil && *v > limits.MaxIndex {
		return fail("index %d exceeds the limit of %d", *v, limits.MaxIndex)
	}
	return nil
}

// checkConstant bounds the literal a comparison is written against, so a bundle
// cannot state one no checked expression could ever produce. Both COMPARE_CONSTANT
// and INTEGER_IS carry it.
func checkConstant(v *int64, fail func(string, ...any) error) error {
	if v != nil && (*v < limits.MinConstant || *v > limits.MaxConstant) {
		return fail("constant %d is outside the accepted range %d..%d",
			*v, limits.MinConstant, limits.MaxConstant)
	}
	return nil
}

func checkMessageKey(v *string, fail func(string, ...any) error) error {
	if v == nil {
		return nil
	}
	// A present but empty key cannot be told apart from an absent one in an
	// idiomatic API, so two engines could report differently on the same
	// bundle. Constants are already forbidden from being empty.
	if *v == "" {
		return fail("message key must not be empty when present")
	}
	if len(*v) > limits.MaxConstantBytes {
		return fail("message key exceeds %d bytes", limits.MaxConstantBytes)
	}
	return nil
}

func assertionReasonAllowed(code irv1.ReasonCode) bool {
	switch code {
	case irv1.ReasonCode_REASON_CODE_EMPTY,
		irv1.ReasonCode_REASON_CODE_INVALID_LENGTH,
		irv1.ReasonCode_REASON_CODE_INVALID_CHARACTERS,
		irv1.ReasonCode_REASON_CODE_INVALID_FORMAT,
		irv1.ReasonCode_REASON_CODE_COUNTRY_MISMATCH:
		return true
	default:
		return false
	}
}

func unsupportedReasonAllowed(code irv1.ReasonCode) bool {
	switch code {
	case irv1.ReasonCode_REASON_CODE_UNSUPPORTED_CHECKSUM,
		irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED:
		return true
	default:
		return false
	}
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func runeCount(s string) int {
	n := 0
	for range s {
		n++
	}
	return n
}
