package typecheck

import (
	"math"
	"sort"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/ast"
	"github.com/entid-org/spec/internal/features"
	"github.com/entid-org/spec/internal/limits"
)

// maxMappedDigit is the largest value a character mapping can contribute.
const maxMappedDigit = 35

// finishNode applies the operation specific invariants and computes the static
// bounds proving that no arithmetic can overflow.
func (c *checker) finishNode(n *Node, call *ast.CallExpr) bool {
	if !c.checkAlphabet(n, call) {
		return false
	}
	if len(n.Inputs) < n.Op.MinOperands() {
		c.bag.Errorf(n.Pos, CodeArity, "%s expects at least %d operand(s), got %d",
			call.Name, n.Op.MinOperands(), len(n.Inputs))
		return false
	}
	if upper := n.Op.MaxOperands(); upper >= 0 && len(n.Inputs) > upper {
		c.bag.Errorf(n.Pos, CodeArity, "%s accepts at most %d operand(s), got %d",
			call.Name, upper, len(n.Inputs))
		return false
	}
	if n.Op.Category == features.CategoryChecksum &&
		n.Op.Code != int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_CHOOSE) {
		for _, in := range n.Inputs {
			if isChecksumWhen(in) {
				c.bag.Errorf(in.Pos, CodeContext,
					"when_checksum is only accepted as a direct branch of choose")
				return false
			}
		}
	}

	switch n.Op.Category {
	case features.CategoryString:
		return c.finishString(n, call)
	case features.CategoryInteger:
		return c.finishInteger(n, call)
	case features.CategoryPredicate:
		return c.finishPredicate(n, call)
	case features.CategoryCanonicalization:
		return c.finishCanonicalization(n, call)
	case features.CategoryChecksum:
		return c.finishChecksum(n, call)
	case features.CategoryAssertion, features.CategoryCall:
		return true
	default:
		return true
	}
}

func isChecksumWhen(n *Node) bool {
	return n.Op.Category == features.CategoryChecksum &&
		n.Op.Code == int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN)
}

func (c *checker) finishString(n *Node, call *ast.CallExpr) bool {
	kind := irv1.StringOpKind(n.Op.Code)
	switch kind {
	case irv1.StringOpKind_STRING_OP_KIND_VALUE, irv1.StringOpKind_STRING_OP_KIND_SUBJECT:
		n.MaxLen = unknownLength
	case irv1.StringOpKind_STRING_OP_KIND_COUNTRY_CODE:
		n.MaxLen = 2
	case irv1.StringOpKind_STRING_OP_KIND_SLICE:
		if *n.Start > *n.End {
			c.bag.Errorf(n.Pos, CodeBounds, "slice start %d is greater than end %d", *n.Start, *n.End)
			return false
		}
		n.MaxLen = int(*n.End - *n.Start)
	case irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM:
		if in := n.Inputs[0].MaxLen; in != unknownLength {
			n.MaxLen = maxInt(0, in-int(*n.Start))
		}
	case irv1.StringOpKind_STRING_OP_KIND_SLICE_TO:
		n.MaxLen = int(*n.End)
		if in := n.Inputs[0].MaxLen; in != unknownLength && in < n.MaxLen {
			n.MaxLen = in
		}
	case irv1.StringOpKind_STRING_OP_KIND_BEFORE_FIRST,
		irv1.StringOpKind_STRING_OP_KIND_AFTER_FIRST,
		irv1.StringOpKind_STRING_OP_KIND_STRIP_PREFIX:
		if !c.requireNonEmptyText(n, call) {
			return false
		}
		n.MaxLen = n.Inputs[0].MaxLen
	case irv1.StringOpKind_STRING_OP_KIND_CONCAT:
		total := 0
		for _, in := range n.Inputs {
			if in.MaxLen == unknownLength {
				total = unknownLength
				break
			}
			total += in.MaxLen
		}
		n.MaxLen = total
	case irv1.StringOpKind_STRING_OP_KIND_CONSTANT:
	default:
	}
	return true
}

func (c *checker) finishInteger(n *Node, call *ast.CallExpr) bool {
	switch irv1.IntegerOpKind(n.Op.Code) {
	case irv1.IntegerOpKind_INTEGER_OP_KIND_DIGITS_TO_INTEGER:
		bound := n.Inputs[0].MaxLen
		if bound == unknownLength {
			c.bag.Suggestf(n.Pos, CodeStaticProof,
				"apply digits_to_integer to a bounded view such as slice(...), or use mod_digits",
				"digits_to_integer requires a statically bounded operand")
			return false
		}
		if bound > limits.MaxDigitsToIntegerLength {
			c.bag.Suggestf(n.Pos, CodeStaticProof,
				"use the digit by digit mod_digits family for longer identifiers",
				"digits_to_integer operand can hold %d digits, the provable limit is %d",
				bound, limits.MaxDigitsToIntegerLength)
			return false
		}
	case irv1.IntegerOpKind_INTEGER_OP_KIND_WEIGHTED_SUM:
		positions := len(n.Weights)
		if *n.Alignment == irv1.WeightAlignment_WEIGHT_ALIGNMENT_CYCLE {
			if n.Inputs[0].MaxLen == unknownLength {
				c.bag.Suggestf(n.Pos, CodeStaticProof,
					"apply the cycle alignment to a bounded view such as slice(...)",
					"a cycling weighted_sum requires a statically bounded operand")
				return false
			}
			positions = n.Inputs[0].MaxLen
		}
		maxWeight := int64(0)
		for _, w := range n.Weights {
			if abs64(w) > maxWeight {
				maxWeight = abs64(w)
			}
		}
		if maxWeight != 0 && int64(positions) > math.MaxInt64/(maxWeight*maxMappedDigit) {
			c.bag.Errorf(n.Pos, CodeBounds, "weighted_sum can overflow a signed 64 bit integer")
			return false
		}
	case irv1.IntegerOpKind_INTEGER_OP_KIND_MOD_DIGITS,
		irv1.IntegerOpKind_INTEGER_OP_KIND_MODULO,
		irv1.IntegerOpKind_INTEGER_OP_KIND_COMPLEMENT:
	case irv1.IntegerOpKind_INTEGER_OP_KIND_REMAINDER_MAP:
	default:
	}
	_ = call
	return true
}

func (c *checker) finishPredicate(n *Node, call *ast.CallExpr) bool {
	switch irv1.PredicateOpKind(n.Op.Code) {
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_LENGTH_BETWEEN:
		if *n.MinLength > *n.MaxLength {
			c.bag.Errorf(n.Pos, CodeBounds, "length_between minimum %d is greater than maximum %d",
				*n.MinLength, *n.MaxLength)
			return false
		}
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_CHARSET,
		irv1.PredicateOpKind_PREDICATE_OP_KIND_CHAR_AT_IN:
		if !c.requireNonEmptyText(n, call) {
			return false
		}
		if !isASCII(*n.Text) {
			c.bag.Errorf(n.Pos, CodeBadConstant, "%s expects an ASCII character set", call.Name)
			return false
		}
		n.Text = stringPtr(sortedUnique(*n.Text))
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_STARTS_WITH,
		irv1.PredicateOpKind_PREDICATE_OP_KIND_ENDS_WITH,
		irv1.PredicateOpKind_PREDICATE_OP_KIND_CONTAINS:
		if !c.requireNonEmptyText(n, call) {
			return false
		}
	case irv1.PredicateOpKind_PREDICATE_OP_KIND_PROFILE_IS:
		switch *n.Text {
		case "compatible", "strict_current":
		default:
			c.bag.Errorf(n.Pos, CodeBadConstant,
				"unknown profile %q, expected compatible or strict_current", *n.Text)
			return false
		}
	default:
	}
	return true
}

func (c *checker) finishCanonicalization(n *Node, call *ast.CallExpr) bool {
	switch irv1.CanonicalizationOpKind(n.Op.Code) {
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_CHARS:
		if !c.requireNonEmptyText(n, call) {
			return false
		}
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REPLACE_PREFIX:
		if !c.requireNonEmptyText(n, call) {
			return false
		}
		if *n.Text == *n.Replacement {
			c.bag.Errorf(n.Pos, CodeBadConstant, "replace_prefix replaces a prefix by itself")
			return false
		}
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_APPEND,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_INSERT:
		if !c.requireNonEmptyText(n, call) {
			return false
		}
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_LEFT_PAD:
		if n.Text == nil || runeLen(*n.Text) != 1 {
			c.bag.Errorf(n.Pos, CodeBadConstant, "left_pad expects exactly one padding code point")
			return false
		}
		if *n.Length == 0 {
			c.bag.Errorf(n.Pos, CodeBounds, "left_pad length must be at least 1")
			return false
		}
	default:
	}
	return true
}

func (c *checker) finishChecksum(n *Node, call *ast.CallExpr) bool {
	if irv1.ChecksumOpKind(n.Op.Code) == irv1.ChecksumOpKind_CHECKSUM_OP_KIND_COMPARE_SLICE {
		if *n.Start >= *n.End {
			c.bag.Errorf(n.Pos, CodeBounds, "compare_slice start %d must be lower than end %d", *n.Start, *n.End)
			return false
		}
		if int(*n.End-*n.Start) > limits.MaxDigitsToIntegerLength {
			c.bag.Errorf(n.Pos, CodeBounds,
				"compare_slice covers %d digits, the provable limit is %d",
				*n.End-*n.Start, limits.MaxDigitsToIntegerLength)
			return false
		}
	}
	_ = call
	return true
}

func (c *checker) requireNonEmptyText(n *Node, call *ast.CallExpr) bool {
	if n.Text == nil || *n.Text == "" {
		c.bag.Errorf(n.Pos, CodeBadConstant, "%s expects a non empty constant", call.Name)
		return false
	}
	return true
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sortStrings(v []string) { sort.Strings(v) }

func sortUint32(v []uint32) { sort.Slice(v, func(i, j int) bool { return v[i] < v[j] }) }

// nearestName returns the closest surface function name.
func nearestName(want string) (string, bool) {
	best, bestDistance := "", 1<<30
	for _, candidate := range SurfaceNames() {
		d := distance(want, candidate)
		if d < bestDistance {
			best, bestDistance = candidate, d
		}
	}
	limit := len(want) / 3
	if limit < 2 {
		limit = 2
	}
	if best == "" || bestDistance > limit {
		return "", false
	}
	return best, true
}

func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = minOf(minOf(cur[j-1]+1, prev[j]+1), prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}

func minOf(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// checkAlphabet keeps the mapping and the alphabet in step.
//
// CUSTOM_ALPHABET reads the value of a code point from the alphabet the
// operation carries, so an operation naming that mapping without one computes
// nothing, and an operation carrying one under another mapping states
// something no runtime reads. Both are caught here rather than at load time,
// because a rule author should hear about it while writing the rule.
func (c *checker) checkAlphabet(n *Node, call *ast.CallExpr) bool {
	custom := n.Mapping != nil &&
		*n.Mapping == irv1.CharMapping_CHAR_MAPPING_CUSTOM_ALPHABET
	switch {
	case custom && (n.Alphabet == nil || *n.Alphabet == ""):
		c.bag.Errorf(n.Pos, CodeBadConstant,
			"%s uses custom_alphabet and must be given a non empty alphabet", call.Name)
		return false
	case !custom && n.Alphabet != nil:
		c.bag.Errorf(n.Pos, CodeBadConstant,
			"%s carries an alphabet that only custom_alphabet reads", call.Name)
		return false
	}
	if !custom {
		return true
	}
	// A repeated code point would carry two values at once, and which one wins
	// would depend on how an implementation happens to search the alphabet.
	seen := map[rune]bool{}
	for _, r := range *n.Alphabet {
		if seen[r] {
			c.bag.Errorf(n.Pos, CodeBadConstant,
				"the alphabet lists %q twice, so its value would depend on the implementation", string(r))
			return false
		}
		seen[r] = true
	}
	if len(seen) > limits.MaxAlphabetRunes {
		c.bag.Errorf(n.Pos, CodeLimit,
			"the alphabet holds %d code points, above the accepted %d",
			len(seen), limits.MaxAlphabetRunes)
		return false
	}
	return true
}
