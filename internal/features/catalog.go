package features

import (
	"fmt"
	"sort"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
)

// Category groups the IR operations by the oneof branch that carries them.
type Category string

// The V1 operation categories.
const (
	CategoryString           Category = "string"
	CategoryInteger          Category = "integer"
	CategoryPredicate        Category = "predicate"
	CategoryCanonicalization Category = "canonicalization"
	CategoryAssertion        Category = "assertion"
	CategoryChecksum         Category = "checksum"
	CategoryCall             Category = "call"
)

// Categories lists the operation categories in documentation order.
var Categories = []Category{
	CategoryString, CategoryInteger, CategoryPredicate,
	CategoryCanonicalization, CategoryAssertion, CategoryChecksum, CategoryCall,
}

// Param names the Protobuf parameter fields an operation may carry. The names
// match the Protobuf field names of the operation message exactly.
type Param string

// Parameter fields of the operation messages.
const (
	ParamText            Param = "text"
	ParamReplacement     Param = "replacement"
	ParamStart           Param = "start"
	ParamEnd             Param = "end"
	ParamIndex           Param = "index"
	ParamLength          Param = "length"
	ParamMinLength       Param = "min_length"
	ParamMaxLength       Param = "max_length"
	ParamLengths         Param = "lengths"
	ParamValues          Param = "values"
	ParamModulus         Param = "modulus"
	ParamConstant        Param = "constant"
	ParamWeights         Param = "weights"
	ParamAlignment       Param = "alignment"
	ParamMapping         Param = "mapping"
	ParamAlphabet        Param = "alphabet"
	ParamRemainderValues Param = "remainder_values"
	ParamReasonCode      Param = "reason_code"
	ParamMessageKey      Param = "message_key"
	ParamProgramID       Param = "program_id"
)

// Operand describes one positional operand of an operation.
type Operand struct {
	// Name documents the operand.
	Name string
	// Type is the required static type of the operand.
	Type irv1.ValueType
}

// Op is the frozen description of one concrete IR operation.
type Op struct {
	// Category is the oneof branch carrying the operation.
	Category Category
	// Code is the numeric value of the operation kind enum.
	Code int32
	// Symbol is the Protobuf enum value name.
	Symbol string
	// HCL is the surface syntax name, empty for operations with no direct
	// surface syntax.
	HCL string
	// Output is the static output type of the node.
	Output irv1.ValueType
	// Operands are the fixed leading operands.
	Operands []Operand
	// Variadic, when set, describes the repeated trailing operand.
	Variadic *Operand
	// MinVariadic and MaxVariadic bound the number of trailing operands.
	MinVariadic int
	MaxVariadic int
	// Required lists the parameter fields that must be present.
	Required []Param
	// Optional lists the parameter fields that may be present.
	Optional []Param
	// Features lists every capability ID required by the operation.
	Features []uint32
	// Doc is the normative semantics of the operation.
	Doc string
}

// MinOperands returns the smallest accepted number of operands.
func (o Op) MinOperands() int {
	if o.Variadic == nil {
		return len(o.Operands)
	}
	return len(o.Operands) + o.MinVariadic
}

// MaxOperands returns the largest accepted number of operands, or -1 when the
// operation accepts an unbounded number of trailing operands.
func (o Op) MaxOperands() int {
	if o.Variadic == nil {
		return len(o.Operands)
	}
	if o.MaxVariadic < 0 {
		return -1
	}
	return len(o.Operands) + o.MaxVariadic
}

// OperandType returns the required type of the operand at the given position.
func (o Op) OperandType(i int) (irv1.ValueType, bool) {
	if i < len(o.Operands) {
		return o.Operands[i].Type, true
	}
	if o.Variadic == nil {
		return irv1.ValueType_VALUE_TYPE_UNSPECIFIED, false
	}
	return o.Variadic.Type, true
}

// AllowsParam reports whether the parameter field may be present.
func (o Op) AllowsParam(p Param) bool {
	for _, r := range o.Required {
		if r == p {
			return true
		}
	}
	for _, r := range o.Optional {
		if r == p {
			return true
		}
	}
	return false
}

// RequiresParam reports whether the parameter field must be present.
func (o Op) RequiresParam(p Param) bool {
	for _, r := range o.Required {
		if r == p {
			return true
		}
	}
	return false
}

const (
	tString = irv1.ValueType_VALUE_TYPE_STRING
	tInt    = irv1.ValueType_VALUE_TYPE_INTEGER
	tBool   = irv1.ValueType_VALUE_TYPE_BOOLEAN
	tStep   = irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP
	tAssert = irv1.ValueType_VALUE_TYPE_ASSERTION
	tCheck  = irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME
)

func str(name string) Operand { return Operand{Name: name, Type: tString} }

// integerOperand, predicateOperand, stepOperand, assertionOperand and
// checksumOperand name the single operand shape each category uses.
var (
	integerOperand   = Operand{Name: "int_expr", Type: tInt}
	predicateOperand = Operand{Name: "predicate", Type: tBool}
	stepOperand      = Operand{Name: "step", Type: tStep}
	assertionOperand = Operand{Name: "assertion", Type: tAssert}
	checksumOperand  = Operand{Name: "checksum_rule", Type: tCheck}
)

// ops is the frozen catalog of concrete IR operations.
var ops = []Op{
	// ---------------------------------------------------------------- string
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_CONSTANT),
		Symbol: "STRING_OP_KIND_CONSTANT", HCL: "", Output: tString,
		Required: []Param{ParamText}, Features: []uint32{CoreGraphV1},
		Doc: "Yields the constant `text`. Constants are UTF-8 and at most 4096 bytes long. A string literal in the surface language lowers to this operation.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_VALUE),
		Symbol: "STRING_OP_KIND_VALUE", HCL: "value()", Output: tString,
		Features: []uint32{CoreGraphV1},
		Doc:      "In a canonicalization program, yields the value current at the moment the enclosing step runs. In a format or checksum program, yields the canonical value of the identifier under validation. Never absent.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_SUBJECT),
		Symbol: "STRING_OP_KIND_SUBJECT", HCL: "subject()", Output: tString,
		Features: []uint32{CoreGraphV1},
		Doc:      "Yields the subject of the enclosing program: the caller supplied view for a called program, otherwise `Program.subject_node`, otherwise the canonical value. Forbidden in canonicalization programs. May be absent when the caller supplied an absent view.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_COUNTRY_CODE),
		Symbol: "STRING_OP_KIND_COUNTRY_CODE", HCL: "country_code()", Output: tString,
		Features: []uint32{CoreGraphV1, IdentifierDispatchV1},
		Doc:      "Yields the canonical country code of the selected dispatch target, or the absent string for a GLOBAL target.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_SLICE),
		Symbol: "STRING_OP_KIND_SLICE", HCL: "slice(expr, start, end)", Output: tString,
		Operands: []Operand{str("expr")}, Required: []Param{ParamStart, ParamEnd},
		Features: []uint32{CoreGraphV1, StringViewsV1},
		Doc:      "Yields the code points of `expr` in `[start, end)`. Absent when `expr` is absent, when `start > end` or when `end` exceeds the length of `expr`.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM),
		Symbol: "STRING_OP_KIND_SLICE_FROM", HCL: "slice_from(expr, start)", Output: tString,
		Operands: []Operand{str("expr")}, Required: []Param{ParamStart},
		Features: []uint32{CoreGraphV1, StringViewsV1},
		Doc:      "Yields the code points of `expr` from `start` to the end. Absent when `expr` is absent or when `start` exceeds the length of `expr`.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_SLICE_TO),
		Symbol: "STRING_OP_KIND_SLICE_TO", HCL: "slice_to(expr, end)", Output: tString,
		Operands: []Operand{str("expr")}, Required: []Param{ParamEnd},
		Features: []uint32{CoreGraphV1, StringViewsV1},
		Doc:      "Yields the code points of `expr` before `end`. Absent when `expr` is absent or when `end` exceeds the length of `expr`.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_BEFORE_FIRST),
		Symbol: "STRING_OP_KIND_BEFORE_FIRST", HCL: "before_first(expr, delimiter)", Output: tString,
		Operands: []Operand{str("expr")}, Required: []Param{ParamText},
		Features: []uint32{CoreGraphV1, StringViewsV1},
		Doc:      "Yields the part of `expr` before the first occurrence of the non empty constant `text`. Absent when `expr` is absent or when `text` does not occur.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_AFTER_FIRST),
		Symbol: "STRING_OP_KIND_AFTER_FIRST", HCL: "after_first(expr, delimiter)", Output: tString,
		Operands: []Operand{str("expr")}, Required: []Param{ParamText},
		Features: []uint32{CoreGraphV1, StringViewsV1},
		Doc:      "Yields the part of `expr` after the first occurrence of the non empty constant `text`. Absent when `expr` is absent or when `text` does not occur.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX),
		Symbol: "STRING_OP_KIND_STRIP_PREFIX", HCL: "strip_prefix(expr, prefix)", Output: tString,
		Operands: []Operand{str("expr")}, Required: []Param{ParamText},
		Features: []uint32{CoreGraphV1, StringViewsV1},
		Doc:      "Yields `expr` without its exact leading `text`. Absent when `expr` is absent or does not start with `text`.",
	},
	{
		Category: CategoryString, Code: int32(irv1.StringOpKind_STRING_OP_KIND_CONCAT),
		Symbol: "STRING_OP_KIND_CONCAT", HCL: "concat(expr...)", Output: tString,
		Variadic: ptrOperand(str("expr")), MinVariadic: 1, MaxVariadic: 256,
		Features: []uint32{CoreGraphV1, StringViewsV1},
		Doc:      "Concatenates its operands in order. Absent when any operand is absent.",
	},

	// --------------------------------------------------------------- integer
	{
		Category: CategoryInteger, Code: int32(irv1.IntegerOpKind_INTEGER_OP_KIND_DIGITS_TO_INTEGER),
		Symbol: "INTEGER_OP_KIND_DIGITS_TO_INTEGER", HCL: "digits_to_integer(expr)", Output: tInt,
		Operands: []Operand{str("expr")}, Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc: "Reads `expr` as a non negative decimal integer. Indeterminate when `expr` is absent, empty or contains a non ASCII digit. Accepted only when the compiler proves the maximum length of `expr` is at most 18 code points.",
	},
	{
		Category: CategoryInteger, Code: int32(irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS),
		Symbol: "INTEGER_OP_KIND_MOD_DIGITS", HCL: "mod_digits(expr, modulus)", Output: tInt,
		Operands: []Operand{str("expr")}, Required: []Param{ParamModulus},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Computes the remainder of `expr` modulo `modulus`, digit by digit, without any big integer conversion. Indeterminate when `expr` is absent, empty or contains a non ASCII digit. The result lies in `[0, modulus)`.",
	},
	{
		Category: CategoryInteger, Code: int32(irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM),
		Symbol: "INTEGER_OP_KIND_WEIGHTED_SUM", HCL: "weighted_sum(expr, weights, alignment, mapping[, alphabet])", Output: tInt,
		Operands: []Operand{str("expr")}, Required: []Param{ParamWeights, ParamAlignment, ParamMapping},
		Optional: []Param{ParamAlphabet},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1, ChecksumWeightedV1},
		Doc:      "Sums `mapping(expr[i]) * weight(i)` over the paired positions. `LEFT` pairs position `i` with `weights[i]`, `RIGHT` pairs the last position with the last weight, and `CYCLE` pairs position `i` with `weights[i mod len(weights)]`. `LEFT` and `RIGHT` only pair `min(len(expr), len(weights))` positions; the remaining positions of `expr` contribute nothing. Indeterminate when `expr` is absent, empty, or contains a code point outside the mapping domain. `CUSTOM_ALPHABET` takes the value of a code point from its index in `alphabet`, which is required by that mapping and forbidden by the others. The alphabet holds between 1 and 256 code points and lists none of them twice: a repeated code point would carry two values, and which one an engine returned would depend on how it searched, which is how two conformant engines disagree without either being wrong.",
	},
	{
		Category: CategoryInteger, Code: int32(irv1.IntegerOpKind_INTEGER_OP_KIND_MODULO),
		Symbol: "INTEGER_OP_KIND_MODULO", HCL: "modulo(int_expr, modulus)", Output: tInt,
		Operands: []Operand{integerOperand}, Required: []Param{ParamModulus},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Euclidean remainder of `int_expr` modulo `modulus`. The result always lies in `[0, modulus)`. Indeterminate when the operand is indeterminate.",
	},
	{
		Category: CategoryInteger, Code: int32(irv1.IntegerOpKind_INTEGER_OP_KIND_COMPLEMENT),
		Symbol: "INTEGER_OP_KIND_COMPLEMENT", HCL: "complement(int_expr, modulus)", Output: tInt,
		Operands: []Operand{integerOperand}, Required: []Param{ParamModulus},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Yields `modulus - int_expr`. Indeterminate when the operand is indeterminate or outside `[0, modulus]`.",
	},
	{
		Category: CategoryInteger, Code: int32(irv1.IntegerOpKind_INTEGER_OP_KIND_REMAINDER_MAP),
		Symbol: "INTEGER_OP_KIND_REMAINDER_MAP", HCL: "remainder_map(int_expr, values)", Output: tInt,
		Operands: []Operand{integerOperand}, Required: []Param{ParamRemainderValues},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Yields `remainder_values[int_expr]`. Indeterminate when the operand is indeterminate or outside `[0, len(remainder_values))`.",
	},

	// ------------------------------------------------------------- predicate
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_IS_EMPTY),
		Symbol: "PREDICATE_OP_KIND_IS_EMPTY", HCL: "is_empty(expr)", Output: tBool,
		Operands: []Operand{str("expr")}, Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc: "True when `expr` is present and holds zero code points. False when `expr` is absent.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_IS_ABSENT),
		Symbol: "PREDICATE_OP_KIND_IS_ABSENT", HCL: "is_absent(expr)", Output: tBool,
		Operands: []Operand{str("expr")}, Features: []uint32{CoreGraphV1, StringViewsV1},
		Doc: "True when `expr` is absent. This is the only predicate that observes absence as true.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_EQUALS),
		Symbol: "PREDICATE_OP_KIND_EQUALS", HCL: "equals(left, right)", Output: tBool,
		Operands: []Operand{str("left"), str("right")}, Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc: "True when both operands are present and hold the same code point sequence. False when either operand is absent.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_EQ),
		Symbol: "PREDICATE_OP_KIND_LENGTH_EQ", HCL: "length_eq(expr, n)", Output: tBool,
		Operands: []Operand{str("expr")}, Required: []Param{ParamLength},
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "True when `expr` is present and holds exactly `length` code points.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_IN),
		Symbol: "PREDICATE_OP_KIND_LENGTH_IN", HCL: "length_in(expr, [n...])", Output: tBool,
		Operands: []Operand{str("expr")}, Required: []Param{ParamLengths},
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "True when `expr` is present and its code point length belongs to `lengths`. `lengths` is sorted, deduplicated and non empty.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_BETWEEN),
		Symbol: "PREDICATE_OP_KIND_LENGTH_BETWEEN", HCL: "length_between(expr, min, max)", Output: tBool,
		Operands: []Operand{str("expr")}, Required: []Param{ParamMinLength, ParamMaxLength},
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "True when `expr` is present and its code point length lies in `[min_length, max_length]`. The compiler requires `min_length <= max_length`.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS),
		Symbol: "PREDICATE_OP_KIND_ASCII_DIGITS", HCL: "ascii_digits(expr)", Output: tBool,
		Operands: []Operand{str("expr")}, Features: []uint32{CoreGraphV1, AsciiAndWhitespaceV1},
		Doc: "True when `expr` is present, non empty and made only of `U+0030..U+0039`.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_UPPER_LETTERS),
		Symbol: "PREDICATE_OP_KIND_ASCII_UPPER_LETTERS", HCL: "ascii_upper_letters(expr)", Output: tBool,
		Operands: []Operand{str("expr")}, Features: []uint32{CoreGraphV1, AsciiAndWhitespaceV1},
		Doc: "True when `expr` is present, non empty and made only of `U+0041..U+005A`.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_ALPHANUMERIC),
		Symbol: "PREDICATE_OP_KIND_ASCII_ALPHANUMERIC", HCL: "ascii_alphanumeric(expr)", Output: tBool,
		Operands: []Operand{str("expr")}, Features: []uint32{CoreGraphV1, AsciiAndWhitespaceV1},
		Doc: "True when `expr` is present, non empty and made only of `U+0030..U+0039` or `U+0041..U+005A`.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_CHARSET),
		Symbol: "PREDICATE_OP_KIND_ASCII_CHARSET", HCL: "ascii_charset(expr, chars)", Output: tBool,
		Operands: []Operand{str("expr")}, Required: []Param{ParamText},
		Features: []uint32{CoreGraphV1, AsciiAndWhitespaceV1},
		Doc:      "True when `expr` is present, non empty and every code point belongs to the non empty ASCII set `text`. `text` is deduplicated and sorted by code point by the compiler.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_STARTS_WITH),
		Symbol: "PREDICATE_OP_KIND_STARTS_WITH", HCL: "starts_with(expr, prefix)", Output: tBool,
		Operands: []Operand{str("expr")}, Required: []Param{ParamText},
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "True when `expr` is present and starts with the non empty constant `text`.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ENDS_WITH),
		Symbol: "PREDICATE_OP_KIND_ENDS_WITH", HCL: "ends_with(expr, suffix)", Output: tBool,
		Operands: []Operand{str("expr")}, Required: []Param{ParamText},
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "True when `expr` is present and ends with the non empty constant `text`.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_PREFIX_IN),
		Symbol: "PREDICATE_OP_KIND_PREFIX_IN", HCL: "prefix_in(expr, prefixes)", Output: tBool,
		Operands: []Operand{str("expr")}, Required: []Param{ParamValues},
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc: "True when `expr` is present and starts with at least one element of `values`. " +
			"`values` is non empty, sorted and deduplicated by the compiler, and every element is non empty. " +
			"Every element has the same length, and a bundle mixing lengths is refused: over one sorted list " +
			"of mixed lengths, a search for the greatest element not after `expr` answers wrongly rather than " +
			"slowly, since `[\"AB\", \"ABA\"]` against `\"ABCD\"` finds `ABA`, which is not a prefix, while `AB` is. " +
			"At one length, starting with an element is equalling its opening of that length, so the search is " +
			"exact. Mixed lengths are written as one `prefix_in` per length under an `any`.\n\n" +
			"That refusal takes the evidence with it. Every `prefix_in` a bundle may now carry holds " +
			"one length, so no conformance case can distinguish a search run per length from one run " +
			"over the whole table: the shape that separates them is the shape the loader refuses. An " +
			"engine MUST therefore pin the semantics below its loader, by a native test comparing its " +
			"search against the definition transcribed literally -- some element is a prefix of the " +
			"subject -- over tables of mixed lengths. This is the second rule the corpus cannot carry, " +
			"alongside `invalid_encoding`, and for the same kind of reason: what makes a case " +
			"expressible and what makes a rule worth stating are not the same thing.\n\n" +
			"The unit is bytes because the search is over bytes, and an engine working in another " +
			"unit may group more finely without contradicting this: two elements of the same byte " +
			"length can differ in code points, since `PZ` and `\u00e9` are both two bytes and are not " +
			"both two code points. A finer grouping refuses nothing this accepts. No conformance " +
			"case separates the two readings, because every element of the published bundle is " +
			"ASCII, where they agree.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_CHAR_AT_IN),
		Symbol: "PREDICATE_OP_KIND_CHAR_AT_IN", HCL: "char_at_in(expr, index, chars)", Output: tBool,
		Operands: []Operand{str("expr")}, Required: []Param{ParamIndex, ParamText},
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "True when `expr` is present, `index` is a valid code point position and the code point at `index` belongs to the non empty ASCII set `text`.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_INTEGER_IS),
		Symbol: "PREDICATE_OP_KIND_INTEGER_IS", HCL: "integer_is(int_expr, constant)", Output: tBool,
		Operands: []Operand{integerOperand}, Required: []Param{ParamConstant},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1, ChecksumIntegerPredicateV1},
		Doc:      "True when `int_expr` equals the literal `constant`. It is the only predicate reading an integer, and exists so that a checksum can branch on the value of a remainder: several national registers recompute their weighted sum with a second set of weights when the first remainder reaches a given value. An indeterminate operand yields `false`, so the branch does not apply and the enclosing `CHOOSE` falls through.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_CONTAINS),
		Symbol: "PREDICATE_OP_KIND_CONTAINS", HCL: "contains(expr, literal)", Output: tBool,
		Operands: []Operand{str("expr")}, Required: []Param{ParamText},
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "True when `expr` is present and contains the non empty constant `text`.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ALL),
		Symbol: "PREDICATE_OP_KIND_ALL", HCL: "all(predicate...)", Output: tBool,
		Variadic: ptrOperand(predicateOperand), MinVariadic: 1, MaxVariadic: -1,
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "True when every operand is true. Operands are evaluated in order and evaluation stops at the first false operand.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ANY),
		Symbol: "PREDICATE_OP_KIND_ANY", HCL: "any(predicate...)", Output: tBool,
		Variadic: ptrOperand(predicateOperand), MinVariadic: 1, MaxVariadic: -1,
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "True when at least one operand is true. Operands are evaluated in order and evaluation stops at the first true operand.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_NOT),
		Symbol: "PREDICATE_OP_KIND_NOT", HCL: "not(predicate)", Output: tBool,
		Operands: []Operand{predicateOperand}, Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc: "Negates its operand.",
	},
	{
		Category: CategoryPredicate, Code: int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_PROFILE_IS),
		Symbol: "PREDICATE_OP_KIND_PROFILE_IS", HCL: "profile_is(name)", Output: tBool,
		Required: []Param{ParamText}, Features: []uint32{CoreGraphV1, ProfilesV1},
		Doc: "True when the effective validation profile equals `text`, which is either `compatible` or `strict_current`.",
	},

	// ------------------------------------------------------ canonicalization
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_SEQUENCE),
		Symbol: "CANONICALIZATION_OP_KIND_SEQUENCE", HCL: "", Output: tStep,
		Variadic: ptrOperand(stepOperand), MinVariadic: 0, MaxVariadic: -1,
		Features: []uint32{CoreGraphV1, CanonicalizationBasicV1},
		Doc:      "Applies its operands in order to the current value. It is the only accepted root of a canonicalization program.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_TRIM_WHITESPACE),
		Symbol: "CANONICALIZATION_OP_KIND_TRIM_WHITESPACE", HCL: "trim_whitespace()", Output: tStep,
		Features: []uint32{CoreGraphV1, AsciiAndWhitespaceV1, CanonicalizationBasicV1},
		Doc:      "Removes every leading and trailing code point of the frozen `whitespace_v1` table.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_WHITESPACE),
		Symbol: "CANONICALIZATION_OP_KIND_REMOVE_WHITESPACE", HCL: "remove_whitespace()", Output: tStep,
		Features: []uint32{CoreGraphV1, AsciiAndWhitespaceV1, CanonicalizationBasicV1},
		Doc:      "Removes every code point of the frozen `whitespace_v1` table.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_UPPERCASE_ASCII),
		Symbol: "CANONICALIZATION_OP_KIND_UPPERCASE_ASCII", HCL: "uppercase_ascii()", Output: tStep,
		Features: []uint32{CoreGraphV1, AsciiAndWhitespaceV1, CanonicalizationBasicV1},
		Doc:      "Maps only `a..z` to `A..Z`. Every other code point is preserved.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_CHARS),
		Symbol: "CANONICALIZATION_OP_KIND_REMOVE_CHARS", HCL: "remove_chars(list)", Output: tStep,
		Required: []Param{ParamText}, Features: []uint32{CoreGraphV1, CanonicalizationBasicV1},
		Doc: "Removes every code point belonging to the non empty set `text`. The compiler deduplicates and sorts `text` by code point.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REPLACE_PREFIX),
		Symbol: "CANONICALIZATION_OP_KIND_REPLACE_PREFIX", HCL: "replace_prefix(from, to)", Output: tStep,
		Required: []Param{ParamText, ParamReplacement}, Features: []uint32{CoreGraphV1, CanonicalizationBasicV1},
		Doc: "Replaces the exact leading `text` by `replacement` when present. `text` is non empty and differs from `replacement`.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND),
		Symbol: "CANONICALIZATION_OP_KIND_PREPEND", HCL: "prepend(value)", Output: tStep,
		Required: []Param{ParamText}, Features: []uint32{CoreGraphV1, CanonicalizationBasicV1},
		Doc: "Inserts the non empty constant `text` before the current value.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_APPEND),
		Symbol: "CANONICALIZATION_OP_KIND_APPEND", HCL: "append(value)", Output: tStep,
		Required: []Param{ParamText}, Features: []uint32{CoreGraphV1, CanonicalizationBasicV1},
		Doc: "Appends the non empty constant `text` after the current value.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_INSERT),
		Symbol: "CANONICALIZATION_OP_KIND_INSERT", HCL: "insert(index, value)", Output: tStep,
		Required: []Param{ParamIndex, ParamText}, Features: []uint32{CoreGraphV1, CanonicalizationBasicV1},
		Doc: "Inserts the non empty constant `text` at code point position `index`. When `index` is greater than the current length the step leaves the value unchanged.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD),
		Symbol: "CANONICALIZATION_OP_KIND_LEFT_PAD", HCL: "left_pad(length, char)", Output: tStep,
		Required: []Param{ParamLength, ParamText}, Features: []uint32{CoreGraphV1, CanonicalizationBasicV1},
		Doc: "Prepends copies of the single code point `text` until the value holds `length` code points. A longer value is never truncated. `length` is at least 1 and bounded like every other slice bound, so an engine that sizes a buffer from it has a static maximum.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND_COUNTRY_IF_MISSING),
		Symbol: "CANONICALIZATION_OP_KIND_PREPEND_COUNTRY_IF_MISSING", HCL: "prepend_country_if_missing()", Output: tStep,
		Features: []uint32{CoreGraphV1, CanonicalizationBasicV1, IdentifierDispatchV1},
		Doc:      "Leaves the value unchanged when it starts with one of the `accepted_prefixes` of the selected target; otherwise prepends the `canonical_prefix` of the target, or its `country_code` when no canonical prefix is declared. Forbidden in a pre-canonicalization program and in a canonicalizer referenced by a GLOBAL definition.",
	},
	{
		Category: CategoryCanonicalization, Code: int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_WHEN),
		Symbol: "CANONICALIZATION_OP_KIND_WHEN", HCL: "when(predicate, step...)", Output: tStep,
		Operands: []Operand{predicateOperand}, Variadic: ptrOperand(stepOperand), MinVariadic: 1, MaxVariadic: -1,
		Features: []uint32{CoreGraphV1, CanonicalizationConditionalV1},
		Doc:      "Evaluates the predicate against the value current at that point and, when true, applies the trailing steps in order.",
	},

	// ------------------------------------------------------------- assertion
	{
		Category: CategoryAssertion, Code: int32(irv1.AssertionOpKind_ASSERTION_OP_KIND_SEQUENCE),
		Symbol: "ASSERTION_OP_KIND_SEQUENCE", HCL: "", Output: tAssert,
		Variadic: ptrOperand(assertionOperand), MinVariadic: 1, MaxVariadic: -1,
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "Evaluates its operands in order and stops at the first failure, whose reason code and message key become the result of the program. It is the only accepted root of a format program.",
	},
	{
		Category: CategoryAssertion, Code: int32(irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE),
		Symbol: "ASSERTION_OP_KIND_REQUIRE", HCL: "require(predicate, reason_code, message_key)", Output: tAssert,
		Operands: []Operand{predicateOperand}, Required: []Param{ParamReasonCode}, Optional: []Param{ParamMessageKey},
		Features: []uint32{CoreGraphV1, FormatAssertionsV1},
		Doc:      "Succeeds when the predicate is true. On failure the format step is `invalid` with `reason_code` and, when declared, `message_key`. `reason_code` is restricted to codes that prove invalidity.",
	},

	// -------------------------------------------------------------- checksum
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN),
		Symbol: "CHECKSUM_OP_KIND_LUHN", HCL: "luhn(expr)", Output: tCheck,
		Operands: []Operand{str("expr")}, Optional: []Param{ParamMessageKey},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1, ChecksumLuhnV1},
		Doc:      "Applies the Luhn algorithm to `expr`, whose rightmost digit is the check digit. `valid` when the weighted sum is a multiple of ten, `invalid`/`invalid_checksum` otherwise. Indeterminate, hence `unsupported`/`unsupported_checksum`, when `expr` is absent, shorter than two code points, or contains a non ASCII digit.",
	},
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ISO7064_MOD97_10),
		Symbol: "CHECKSUM_OP_KIND_ISO7064_MOD97_10", HCL: "iso7064_mod97_10(expr)", Output: tCheck,
		Operands: []Operand{str("expr")}, Optional: []Param{ParamMessageKey},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1, ChecksumMod97V1},
		Doc:      "Expands every ASCII letter of `expr` to its base 36 decimal value, every ASCII digit to itself, then requires the resulting decimal string to be congruent to one modulo 97. Indeterminate when `expr` is absent, shorter than three code points, or contains a code point outside `0..9` and `A..Z`.",
	},
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT),
		Symbol: "CHECKSUM_OP_KIND_COMPARE_DIGIT", HCL: "compare_digit(int_expr, string_expr, index)", Output: tCheck,
		Operands: []Operand{integerOperand, str("string_expr")}, Required: []Param{ParamIndex}, Optional: []Param{ParamMessageKey},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Compares `int_expr` to the ASCII digit of `string_expr` at code point position `index`. Indeterminate when the integer is indeterminate, the string is absent, `index` is out of range or the code point is not an ASCII digit.",
	},
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_CONSTANT),
		Symbol: "CHECKSUM_OP_KIND_COMPARE_CONSTANT", HCL: "compare_constant(int_expr, constant)", Output: tCheck,
		Operands: []Operand{integerOperand}, Required: []Param{ParamConstant}, Optional: []Param{ParamMessageKey},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1, ChecksumCompareConstantV1},
		Doc:      "Compares `int_expr` to the literal `constant`. `COMPARE_DIGIT` and `COMPARE_SLICE` can only compare against part of the value being checked, so a rule stating that a remainder must equal a fixed number had nothing to compare with. Indeterminate when the integer is indeterminate, which never proves an identifier wrong.",
	},
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE),
		Symbol: "CHECKSUM_OP_KIND_COMPARE_SLICE", HCL: "compare_slice(int_expr, string_expr, start, end)", Output: tCheck,
		Operands: []Operand{integerOperand, str("string_expr")}, Required: []Param{ParamStart, ParamEnd}, Optional: []Param{ParamMessageKey},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Compares `int_expr` to the decimal value of `string_expr[start:end)`. The compiler requires `end - start` between one and 18. Indeterminate when the integer is indeterminate, the string is absent, the slice is out of range or holds a non ASCII digit.",
	},
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_CHOOSE),
		Symbol: "CHECKSUM_OP_KIND_CHOOSE", HCL: "choose(branches...)", Output: tCheck,
		Variadic: ptrOperand(Operand{Name: "branch", Type: tCheck}), MinVariadic: 1, MaxVariadic: -1,
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Evaluates its branches in order and returns the outcome of the first applicable one. A `WHEN` branch whose predicate is false is not applicable; any other branch is always applicable. When no branch applies the result is `unsupported`/`unsupported_checksum`.",
	},
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN),
		Symbol: "CHECKSUM_OP_KIND_WHEN", HCL: "when_checksum(predicate, checksum_rule)", Output: tCheck,
		Operands: []Operand{predicateOperand, checksumOperand},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Applicable only when the predicate is true, in which case it yields the outcome of its checksum operand. It is accepted only as a direct operand of `CHOOSE`; its non applicable state is never observable elsewhere.",
	},
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ALL_CHECKS),
		Symbol: "CHECKSUM_OP_KIND_ALL_CHECKS", HCL: "all_checks(checksum_rule...)", Output: tCheck,
		Variadic: ptrOperand(checksumOperand), MinVariadic: 1, MaxVariadic: -1,
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Evaluates every operand in order. Returns the first `invalid` outcome, otherwise the first `unsupported` outcome, otherwise `valid`. `WHEN` is not accepted as an operand.",
	},
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ANY_CHECK),
		Symbol: "CHECKSUM_OP_KIND_ANY_CHECK", HCL: "any_check(checksum_rule...)", Output: tCheck,
		Variadic: ptrOperand(checksumOperand), MinVariadic: 1, MaxVariadic: -1,
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Evaluates every operand in order. Returns `valid` as soon as one operand is valid, otherwise the first `unsupported` outcome, otherwise the first `invalid` outcome. `WHEN` is not accepted as an operand.",
	},
	{
		Category: CategoryChecksum, Code: int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_UNSUPPORTED),
		Symbol: "CHECKSUM_OP_KIND_UNSUPPORTED", HCL: "unsupported_checksum(reason_code)", Output: tCheck,
		Required: []Param{ParamReasonCode}, Optional: []Param{ParamMessageKey},
		Features: []uint32{CoreGraphV1, ChecksumTristateV1},
		Doc:      "Always yields `unsupported` with `reason_code`, restricted to `unsupported_checksum` and `checksum_not_published`.",
	},

	// ------------------------------------------------------------------ call
	{
		Category: CategoryCall, Code: int32(irv1.CallOpKind_CALL_OP_KIND_FORMAT),
		Symbol: "CALL_OP_KIND_FORMAT", HCL: "use_format", Output: tAssert,
		Operands: []Operand{str("input")}, Required: []Param{ParamProgramID},
		Features: []uint32{CoreGraphV1, CapturesAndCallsV1, FormatAssertionsV1},
		Doc:      "Runs the format program `program_id` with `input` as its subject and propagates its reason code and message key unchanged.",
	},
	{
		Category: CategoryCall, Code: int32(irv1.CallOpKind_CALL_OP_KIND_CHECKSUM),
		Symbol: "CALL_OP_KIND_CHECKSUM", HCL: "apply_checksum(checksum_reference, string_expr)", Output: tCheck,
		Operands: []Operand{str("input")}, Required: []Param{ParamProgramID},
		Features: []uint32{CoreGraphV1, CapturesAndCallsV1, ChecksumTristateV1},
		Doc:      "Runs the checksum program `program_id` with `input` as its subject and propagates its outcome unchanged.",
	},
}

func ptrOperand(o Operand) *Operand { return &o }

type opKey struct {
	category Category
	code     int32
}

var opIndex = func() map[opKey]Op {
	m := make(map[opKey]Op, len(ops))
	for _, o := range ops {
		k := opKey{o.Category, o.Code}
		if _, dup := m[k]; dup {
			panic(fmt.Sprintf("duplicate op %s/%d", o.Category, o.Code))
		}
		m[k] = o
	}
	return m
}()

// Ops returns the catalog in documentation order.
func Ops() []Op {
	out := make([]Op, len(ops))
	copy(out, ops)
	return out
}

// OpsByCategory returns the catalog entries of one category, ordered by code.
func OpsByCategory(c Category) []Op {
	var out []Op
	for _, o := range ops {
		if o.Category == c {
			out = append(out, o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// LookupOp returns the catalog entry of a concrete operation.
func LookupOp(c Category, code int32) (Op, bool) {
	o, ok := opIndex[opKey{c, code}]
	return o, ok
}

// HCLName returns the surface function name of the operation, or the empty
// string when the operation has no direct surface syntax.
func (o Op) HCLName() string {
	if o.HCL == "" {
		return ""
	}
	if idx := indexByte(o.HCL, '('); idx >= 0 {
		return o.HCL[:idx]
	}
	return o.HCL
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
