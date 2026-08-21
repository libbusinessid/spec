package typecheck

import (
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/features"
)

//nolint:funlen // the surface calling conventions are one flat, reviewable table.
func buildSignatures() map[string]signature {
	table := map[string]signature{}
	s := features.CategoryString
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_VALUE))
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_SUBJECT))
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_COUNTRY_CODE))
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_SLICE),
		operand(), intArg(features.ParamStart), intArg(features.ParamEnd))
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM),
		operand(), intArg(features.ParamStart))
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_SLICE_TO),
		operand(), intArg(features.ParamEnd))
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_BEFORE_FIRST),
		operand(), strArg(features.ParamText))
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_AFTER_FIRST),
		operand(), strArg(features.ParamText))
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX),
		operand(), strArg(features.ParamText))
	register(table, s, int32(irv1.StringOpKind_STRING_OP_KIND_CONCAT), variadic())

	i := features.CategoryInteger
	register(table, i, int32(irv1.IntegerOpKind_INTEGER_OP_KIND_DIGITS_TO_INTEGER), operand())
	register(table, i, int32(irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS), operand(), modulusArg())
	register(table, i, int32(irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM),
		operand(), intListArg(features.ParamWeights),
		enumArg(features.ParamAlignment), enumArg(features.ParamMapping),
		optStrArg(features.ParamAlphabet))
	register(table, i, int32(irv1.IntegerOpKind_INTEGER_OP_KIND_MODULO), operand(), modulusArg())
	register(table, i, int32(irv1.IntegerOpKind_INTEGER_OP_KIND_COMPLEMENT), operand(), modulusArg())
	register(table, i, int32(irv1.IntegerOpKind_INTEGER_OP_KIND_REMAINDER_MAP),
		operand(), intListArg(features.ParamRemainderValues))

	p := features.CategoryPredicate
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_IS_EMPTY), operand())
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_IS_ABSENT), operand())
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_EQUALS), operand(), operand())
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_EQ),
		operand(), intArg(features.ParamLength))
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_IN),
		operand(), intListArg(features.ParamLengths))
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_BETWEEN),
		operand(), intArg(features.ParamMinLength), intArg(features.ParamMaxLength))
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS), operand())
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_UPPER_LETTERS), operand())
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_ALPHANUMERIC), operand())
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_CHARSET),
		operand(), strArg(features.ParamText))
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_STARTS_WITH),
		operand(), strArg(features.ParamText))
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ENDS_WITH),
		operand(), strArg(features.ParamText))
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_PREFIX_IN),
		operand(), strListArg(features.ParamValues))
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_CHAR_AT_IN),
		operand(), intArg(features.ParamIndex), strArg(features.ParamText))
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_CONTAINS),
		operand(), strArg(features.ParamText))
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ALL), variadic())
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_ANY), variadic())
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_NOT), operand())
	register(table, p, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_PROFILE_IS), strArg(features.ParamText))

	c := features.CategoryCanonicalization
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_TRIM_WHITESPACE))
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_WHITESPACE))
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_UPPERCASE_ASCII))
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_CHARS), charListArg())
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REPLACE_PREFIX),
		strArg(features.ParamText), strArg(features.ParamReplacement))
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND), strArg(features.ParamText))
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_APPEND), strArg(features.ParamText))
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_INSERT),
		intArg(features.ParamIndex), strArg(features.ParamText))
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD),
		intArg(features.ParamLength), strArg(features.ParamText))
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND_COUNTRY_IF_MISSING))
	register(table, c, int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_WHEN), operand(), variadic())

	a := features.CategoryAssertion
	register(table, a, int32(irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE),
		operand(), reasonArg(), optStrArg(features.ParamMessageKey))

	register(table, features.CategoryPredicate, int32(irv1.PredicateOpKind_PREDICATE_OP_KIND_INTEGER_IS),
		operand(), constantArg())

	k := features.CategoryChecksum
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_LUHN),
		operand(), optStrArg(features.ParamMessageKey))
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ISO7064_MOD97_10), operand())
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_DIGIT),
		operand(), operand(), intArg(features.ParamIndex),
		optStrArg(features.ParamMessageKey))
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE),
		operand(), operand(), intArg(features.ParamStart), intArg(features.ParamEnd),
		optStrArg(features.ParamMessageKey))
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_CONSTANT),
		operand(), constantArg(), optStrArg(features.ParamMessageKey))
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_CHOOSE), variadic())
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN), operand(), operand())
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ALL_CHECKS), variadic())
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_ANY_CHECK), variadic())
	register(table, k, int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_UNSUPPORTED), reasonArg())

	register(table, features.CategoryCall, int32(irv1.CallOpKind_CALL_OP_KIND_CHECKSUM),
		checksumRefArg(), operand())

	return table
}
