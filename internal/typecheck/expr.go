package typecheck

import (
	"fmt"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/ast"
	"github.com/entid-org/spec/internal/features"
	"github.com/entid-org/spec/internal/limits"
)

// expr checks one expression against the expected static type.
func (c *checker) expr(cx *ctx, e ast.Expr, want irv1.ValueType) *Node {
	switch n := e.(type) {
	case *ast.StringLit:
		if want != irv1.ValueType_VALUE_TYPE_STRING {
			c.bag.Errorf(n.Position, CodeTypeMismatch,
				"expected %s, got a string literal", typeName(want))
			return nil
		}
		if len(n.Value) > limits.MaxConstantBytes {
			c.bag.Errorf(n.Position, CodeLimit,
				"string constant of %d bytes exceeds the limit of %d", len(n.Value), limits.MaxConstantBytes)
			return nil
		}
		op, _ := features.LookupOp(features.CategoryString, int32(irv1.StringOpKind_STRING_OP_KIND_CONSTANT))
		cx.nodes++
		return &Node{Op: op, Pos: n.Position, Text: stringPtr(n.Value), MaxLen: runeLen(n.Value)}
	case *ast.RefExpr:
		return c.capture(cx, n, want)
	case *ast.CallExpr:
		return c.call(cx, n, want)
	case *ast.ListExpr:
		c.bag.Errorf(n.Position, CodeTypeMismatch, "a list is not a value of type %s", typeName(want))
		return nil
	case *ast.IntLit:
		c.bag.Suggestf(n.Position, CodeTypeMismatch,
			"integers are only accepted as declared constant arguments",
			"an integer is not a value of type %s", typeName(want))
		return nil
	case *ast.BoolLit:
		c.bag.Errorf(n.Position, CodeTypeMismatch, "a boolean literal is not a value of type %s", typeName(want))
		return nil
	default:
		return nil
	}
}

func (c *checker) capture(cx *ctx, ref *ast.RefExpr, want irv1.ValueType) *Node {
	if len(ref.Parts) != 2 || ref.Parts[0] != "capture" {
		c.bag.Suggestf(ref.Position, CodeUnknownCapture,
			"inside an expression, only capture.<name> references are accepted",
			"unexpected reference %q", ref.String())
		return nil
	}
	if !cx.allowCaptures {
		c.bag.Errorf(ref.Position, CodeContext,
			"captures are only available inside the format rule that declares them")
		return nil
	}
	node, ok := cx.captures[ref.Parts[1]]
	if !ok {
		c.bag.Suggestf(ref.Position, CodeUnknownCapture,
			"declare it with a capture block before it is referenced",
			"unknown capture %q", ref.Parts[1])
		return nil
	}
	if want != irv1.ValueType_VALUE_TYPE_STRING {
		c.bag.Errorf(ref.Position, CodeTypeMismatch, "a capture is a string, not %s", typeName(want))
		return nil
	}
	return node
}

func (c *checker) call(cx *ctx, call *ast.CallExpr, want irv1.ValueType) *Node {
	sig, ok := signatures[call.Name]
	if !ok {
		if best, found := nearestName(call.Name); found {
			c.bag.Suggestf(call.Position, CodeUnknownFunction, "did you mean "+best+"?",
				"unknown function %q", call.Name)
		} else {
			c.bag.Errorf(call.Position, CodeUnknownFunction, "unknown function %q", call.Name)
		}
		return nil
	}
	op := sig.op
	// The program restriction is checked first: "this operation belongs to
	// another declaration" is a more actionable diagnostic than a type
	// mismatch, and both would otherwise fire on the same mistake.
	if !c.allowedInProgram(cx, op, call) {
		return nil
	}
	if op.Output != want {
		c.bag.Errorf(call.Position, CodeTypeMismatch,
			"%s produces %s but %s is expected", call.Name, typeName(op.Output), typeName(want))
		return nil
	}

	node := &Node{Op: op, Pos: call.Position, MaxLen: unknownLength}
	cx.nodes++
	args := call.Args
	index := 0
	for _, slot := range sig.args {
		if slot.kind == argVariadicOperand {
			for index < len(args) {
				child := c.operand(cx, op, len(node.Inputs), args[index])
				if child != nil {
					node.Inputs = append(node.Inputs, child)
				}
				index++
			}
			continue
		}
		if index >= len(args) {
			if slot.kind == argOptionalString {
				continue
			}
			c.bag.Errorf(call.Position, CodeArity, "%s: missing argument %d", call.Name, index+1)
			return nil
		}
		arg := args[index]
		index++
		if !c.bindArg(cx, node, op, slot, arg, call) {
			return nil
		}
	}
	if index < len(args) {
		c.bag.Errorf(args[index].Pos(), CodeArity,
			"%s accepts %d argument(s), got %d", call.Name, index, len(args))
		return nil
	}
	if !c.finishNode(node, call) {
		return nil
	}
	return node
}

func (c *checker) bindArg(cx *ctx, node *Node, op features.Op, slot argSlot, arg ast.Expr, call *ast.CallExpr) bool {
	switch slot.kind {
	case argOperand:
		return c.bindOperand(cx, node, op, arg, call)
	case argInt:
		v, ok := c.uintConstant(arg, limits.MaxIndex)
		if !ok {
			return false
		}
		return c.setUint(node, slot.param, v, arg)
	case argModulus:
		return c.bindModulus(node, arg)
	case argConstant:
		return c.bindConstant(node, arg)
	case argString, argOptionalString:
		return c.bindString(node, slot, arg, call)
	case argCharList:
		return c.bindCharList(node, arg, call)
	case argIntList:
		return c.bindIntList(node, slot.param, arg, call)
	case argStringList:
		return c.bindStringList(node, arg, call)
	case argEnum:
		lit, ok := arg.(*ast.StringLit)
		if !ok {
			c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a string literal", call.Name)
			return false
		}
		return c.setEnum(node, slot.param, lit.Value, arg)
	case argChecksumRef:
		return c.bindChecksumRef(node, arg, call)
	case argReasonCode:
		return c.bindReasonCode(node, arg, call)
	case argVariadicOperand:
		return true
	default:
		return false
	}
}

func (c *checker) bindOperand(cx *ctx, node *Node, op features.Op, arg ast.Expr, call *ast.CallExpr) bool {
	want, ok := op.OperandType(len(node.Inputs))
	if !ok {
		c.bag.Errorf(arg.Pos(), CodeArity, "%s: too many operands", call.Name)
		return false
	}
	child := c.expr(cx, arg, want)
	if child == nil {
		return false
	}
	node.Inputs = append(node.Inputs, child)
	return true
}

func (c *checker) bindModulus(node *Node, arg ast.Expr) bool {
	v, ok := c.intConstant(arg)
	if !ok {
		return false
	}
	if v < limits.MinModulus || v > limits.MaxModulus {
		c.bag.Errorf(arg.Pos(), CodeBounds,
			"modulus %d is outside the accepted range %d..%d", v, limits.MinModulus, limits.MaxModulus)
		return false
	}
	node.Modulus = int64Ptr(v)
	return true
}

// bindConstant binds the literal an outcome is compared against. It is bounded
// like every other integer of the IR so that a rule cannot smuggle an unchecked
// value past the arithmetic limits.
func (c *checker) bindConstant(node *Node, arg ast.Expr) bool {
	v, ok := c.intConstant(arg)
	if !ok {
		return false
	}
	if v < limits.MinConstant || v > limits.MaxConstant {
		c.bag.Errorf(arg.Pos(), CodeBounds,
			"constant %d is outside the accepted range %d..%d", v, limits.MinConstant, limits.MaxConstant)
		return false
	}
	node.Constant = int64Ptr(v)
	return true
}

func (c *checker) bindString(node *Node, slot argSlot, arg ast.Expr, call *ast.CallExpr) bool {
	lit, ok := arg.(*ast.StringLit)
	if !ok {
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a string literal", call.Name)
		return false
	}
	if len(lit.Value) > limits.MaxConstantBytes {
		c.bag.Errorf(arg.Pos(), CodeLimit, "string constant exceeds %d bytes", limits.MaxConstantBytes)
		return false
	}
	switch slot.param {
	case features.ParamText:
		node.Text = stringPtr(lit.Value)
	case features.ParamReplacement:
		node.Replacement = stringPtr(lit.Value)
	case features.ParamMessageKey:
		node.MessageKey = stringPtr(lit.Value)
	case features.ParamAlphabet:
		node.Alphabet = stringPtr(lit.Value)
	default:
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "unsupported string parameter")
		return false
	}
	return true
}

func (c *checker) bindCharList(node *Node, arg ast.Expr, call *ast.CallExpr) bool {
	list, ok := arg.(*ast.ListExpr)
	if !ok {
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a list of single character strings", call.Name)
		return false
	}
	chars := ""
	for _, item := range list.Items {
		lit, ok := item.(*ast.StringLit)
		if !ok || runeLen(lit.Value) != 1 {
			c.bag.Errorf(item.Pos(), CodeBadConstant, "%s expects single character string literals", call.Name)
			return false
		}
		chars += lit.Value
	}
	if chars == "" {
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a non empty character list", call.Name)
		return false
	}
	node.Text = stringPtr(sortedUnique(chars))
	return true
}

func (c *checker) bindStringList(node *Node, arg ast.Expr, call *ast.CallExpr) bool {
	list, ok := arg.(*ast.ListExpr)
	if !ok {
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a list of string literals", call.Name)
		return false
	}
	seen := map[string]bool{}
	values := make([]string, 0, len(list.Items))
	for _, item := range list.Items {
		lit, ok := item.(*ast.StringLit)
		if !ok || lit.Value == "" {
			c.bag.Errorf(item.Pos(), CodeBadConstant, "%s expects non empty string literals", call.Name)
			return false
		}
		if seen[lit.Value] {
			continue
		}
		seen[lit.Value] = true
		values = append(values, lit.Value)
	}
	if len(values) == 0 {
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a non empty list", call.Name)
		return false
	}
	sortStrings(values)
	node.Values = values
	return true
}

func (c *checker) bindChecksumRef(node *Node, arg ast.Expr, call *ast.CallExpr) bool {
	ref, ok := arg.(*ast.RefExpr)
	if !ok || len(ref.Parts) != 3 || ref.Parts[0] != "checksum" {
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a checksum.<namespace>.<name> reference", call.Name)
		return false
	}
	if _, found := c.table.Checksums[ref.String()]; !found {
		c.bag.Errorf(arg.Pos(), CodeUnknownFunction, "unknown checksum %q", ref.String())
		return false
	}
	node.CallTarget = ref.String()
	return true
}

func (c *checker) bindReasonCode(node *Node, arg ast.Expr, call *ast.CallExpr) bool {
	lit, ok := arg.(*ast.StringLit)
	if !ok {
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a reason code string literal", call.Name)
		return false
	}
	code, ok := reasonCode(lit.Value)
	if !ok {
		c.bag.Suggestf(arg.Pos(), CodeReasonCode, "see the reason code registry in docs/ir.md",
			"unknown reason code %q", lit.Value)
		return false
	}
	if !allowedReason(node.Op, code) {
		c.bag.Errorf(arg.Pos(), CodeReasonCode,
			"reason code %q is not allowed by %s", lit.Value, node.Op.HCLName())
		return false
	}
	node.ReasonCode = &code
	return true
}

func (c *checker) bindIntList(node *Node, param features.Param, arg ast.Expr, call *ast.CallExpr) bool {
	list, ok := arg.(*ast.ListExpr)
	if !ok {
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a list of integer literals", call.Name)
		return false
	}
	values := make([]int64, 0, len(list.Items))
	for _, item := range list.Items {
		v, ok := c.intConstant(item)
		if !ok {
			return false
		}
		values = append(values, v)
	}
	switch param {
	case features.ParamWeights:
		if len(values) < limits.MinWeights || len(values) > limits.MaxWeights {
			c.bag.Errorf(arg.Pos(), CodeBounds,
				"%d weights are outside the accepted range %d..%d", len(values), limits.MinWeights, limits.MaxWeights)
			return false
		}
		for _, v := range values {
			if v < -limits.MaxWeightMagnitude || v > limits.MaxWeightMagnitude {
				c.bag.Errorf(arg.Pos(), CodeBounds,
					"weight %d exceeds the magnitude limit of %d", v, limits.MaxWeightMagnitude)
				return false
			}
		}
		node.Weights = values
	case features.ParamRemainderValues:
		if len(values) < limits.MinRemainderValues || len(values) > limits.MaxRemainderValues {
			c.bag.Errorf(arg.Pos(), CodeBounds,
				"%d remainder values are outside the accepted range %d..%d",
				len(values), limits.MinRemainderValues, limits.MaxRemainderValues)
			return false
		}
		node.Remainders = values
	case features.ParamLengths:
		seen := map[uint32]bool{}
		lengths := make([]uint32, 0, len(values))
		for _, v := range values {
			if v < 0 || v > limits.MaxIndex {
				c.bag.Errorf(arg.Pos(), CodeBounds,
					"length %d is outside the accepted range 0..%d", v, limits.MaxIndex)
				return false
			}
			if seen[uint32(v)] {
				continue
			}
			seen[uint32(v)] = true
			lengths = append(lengths, uint32(v))
		}
		if len(lengths) == 0 {
			c.bag.Errorf(arg.Pos(), CodeBadConstant, "%s expects a non empty list", call.Name)
			return false
		}
		sortUint32(lengths)
		node.Lengths = lengths
	default:
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "unsupported integer list parameter")
		return false
	}
	return true
}

func (c *checker) setUint(node *Node, param features.Param, v uint32, arg ast.Expr) bool {
	switch param {
	case features.ParamStart:
		node.Start = uint32Ptr(v)
	case features.ParamEnd:
		node.End = uint32Ptr(v)
	case features.ParamIndex:
		node.Index = uint32Ptr(v)
	case features.ParamLength:
		node.Length = uint32Ptr(v)
	case features.ParamMinLength:
		node.MinLength = uint32Ptr(v)
	case features.ParamMaxLength:
		node.MaxLength = uint32Ptr(v)
	default:
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "unsupported integer parameter")
		return false
	}
	return true
}

func (c *checker) setEnum(node *Node, param features.Param, value string, arg ast.Expr) bool {
	switch param {
	case features.ParamAlignment:
		switch value {
		case "left":
			node.Alignment = alignmentPtr(irv1.WeightAlignment_WEIGHT_ALIGNMENT_LEFT)
		case "right":
			node.Alignment = alignmentPtr(irv1.WeightAlignment_WEIGHT_ALIGNMENT_RIGHT)
		case "cycle":
			node.Alignment = alignmentPtr(irv1.WeightAlignment_WEIGHT_ALIGNMENT_CYCLE)
		default:
			c.bag.Errorf(arg.Pos(), CodeBadConstant,
				"unknown alignment %q, expected left, right or cycle", value)
			return false
		}
	case features.ParamMapping:
		switch value {
		case "digit_value":
			node.Mapping = mappingPtr(irv1.CharMapping_CHAR_MAPPING_DIGIT_VALUE)
		case "alnum_base36":
			node.Mapping = mappingPtr(irv1.CharMapping_CHAR_MAPPING_ALNUM_BASE36)
		case "custom_alphabet":
			node.Mapping = mappingPtr(irv1.CharMapping_CHAR_MAPPING_CUSTOM_ALPHABET)
		default:
			c.bag.Errorf(arg.Pos(), CodeBadConstant,
				"unknown mapping %q, expected digit_value, alnum_base36 or custom_alphabet", value)
			return false
		}
	default:
		c.bag.Errorf(arg.Pos(), CodeBadConstant, "unsupported enum parameter")
		return false
	}
	return true
}

func (c *checker) operand(cx *ctx, op features.Op, position int, arg ast.Expr) *Node {
	want, ok := op.OperandType(position)
	if !ok {
		c.bag.Errorf(arg.Pos(), CodeArity, "too many operands")
		return nil
	}
	return c.expr(cx, arg, want)
}

func (c *checker) uintConstant(e ast.Expr, upper uint32) (uint32, bool) {
	lit, ok := e.(*ast.IntLit)
	if !ok {
		c.bag.Errorf(e.Pos(), CodeBadConstant, "expected an integer literal")
		return 0, false
	}
	if lit.Value < 0 || lit.Value > int64(upper) {
		c.bag.Errorf(e.Pos(), CodeBounds, "integer %d is outside the accepted range 0..%d", lit.Value, upper)
		return 0, false
	}
	return uint32(lit.Value), true
}

func (c *checker) intConstant(e ast.Expr) (int64, bool) {
	lit, ok := e.(*ast.IntLit)
	if !ok {
		c.bag.Errorf(e.Pos(), CodeBadConstant, "expected an integer literal")
		return 0, false
	}
	return lit.Value, true
}

func (c *checker) allowedInProgram(cx *ctx, op features.Op, call *ast.CallExpr) bool {
	switch {
	case op.Category == features.CategoryString &&
		op.Code == int32(irv1.StringOpKind_STRING_OP_KIND_SUBJECT) && !cx.allowSubject:
		c.bag.Suggestf(call.Position, CodeContext, "use value() instead",
			"subject() is not available in a canonicalization program")
		return false
	case op.Category == features.CategoryCanonicalization &&
		cx.kind != irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION:
		c.bag.Errorf(call.Position, CodeContext,
			"%s is only available in a canonicalizer", call.Name)
		return false
	case op.Category == features.CategoryAssertion &&
		cx.kind != irv1.ProgramKind_PROGRAM_KIND_FORMAT:
		c.bag.Errorf(call.Position, CodeContext, "%s is only available in a format rule", call.Name)
		return false
	case (op.Category == features.CategoryChecksum || op.Category == features.CategoryCall) &&
		cx.kind != irv1.ProgramKind_PROGRAM_KIND_CHECKSUM:
		c.bag.Errorf(call.Position, CodeContext, "%s is only available in a checksum rule", call.Name)
		return false
	}
	return true
}

func alignmentPtr(v irv1.WeightAlignment) *irv1.WeightAlignment { return &v }
func mappingPtr(v irv1.CharMapping) *irv1.CharMapping           { return &v }

func typeName(t irv1.ValueType) string {
	switch t {
	case irv1.ValueType_VALUE_TYPE_STRING:
		return "StringExpr"
	case irv1.ValueType_VALUE_TYPE_INTEGER:
		return "IntExpr"
	case irv1.ValueType_VALUE_TYPE_BOOLEAN:
		return "Predicate"
	case irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP:
		return "CanonicalizationStep"
	case irv1.ValueType_VALUE_TYPE_ASSERTION:
		return "Assertion"
	case irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME:
		return "ChecksumRule"
	default:
		return fmt.Sprintf("type(%d)", int32(t))
	}
}

func reasonCode(name string) (irv1.ReasonCode, bool) {
	v, ok := irv1.ReasonCode_value["REASON_CODE_"+upperASCII(name)]
	if !ok || v == 0 {
		return irv1.ReasonCode_REASON_CODE_UNSPECIFIED, false
	}
	return irv1.ReasonCode(v), true
}

// ReasonCodeName returns the lower case registry name of a reason code.
func ReasonCodeName(code irv1.ReasonCode) string {
	return lowerASCII(irv1.ReasonCode_name[int32(code)][len("REASON_CODE_"):])
}

func allowedReason(op features.Op, code irv1.ReasonCode) bool {
	if op.Category == features.CategoryAssertion {
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
	switch code {
	case irv1.ReasonCode_REASON_CODE_UNSUPPORTED_CHECKSUM,
		irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED:
		return true
	default:
		return false
	}
}

func upperASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'a' && b[i] <= 'z' {
			b[i] -= 'a' - 'A'
		}
	}
	return string(b)
}

func lowerASCII(s string) string {
	b := []byte(s)
	for i := range b {
		if b[i] >= 'A' && b[i] <= 'Z' {
			b[i] += 'a' - 'A'
		}
	}
	return string(b)
}
