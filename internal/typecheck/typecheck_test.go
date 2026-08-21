package typecheck_test

import (
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/ast"
	"github.com/libbusinessid/spec/internal/features"
	"github.com/libbusinessid/spec/internal/hcllang"
	"github.com/libbusinessid/spec/internal/linker"
	"github.com/libbusinessid/spec/internal/typecheck"
)

func check(t *testing.T, src string) (*typecheck.Unit, string) {
	t.Helper()
	file, bag := hcllang.ParseFile("rules.hcl", []byte(src))
	if bag.HasErrors() {
		t.Fatalf("parse errors: %v", bag.Sorted())
	}
	table, _ := linker.Link(&ast.Unit{Files: []*ast.File{file}})
	unit, tbag := typecheck.Check(table)
	var sb strings.Builder
	for _, d := range tbag.Sorted() {
		sb.WriteString(d.Code + ": " + d.Message + " | " + d.Suggestion + "\n")
	}
	return unit, sb.String()
}

func mustCheck(t *testing.T, src string) *typecheck.Unit {
	t.Helper()
	unit, out := check(t, src)
	if out != "" {
		t.Fatalf("unexpected diagnostics:\n%s", out)
	}
	return unit
}

func TestCheckCanonicalizer(t *testing.T) {
	unit := mustCheck(t, `
canonicalizer "vat" "common" {
  steps = [
    trim_whitespace(),
    remove_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-", "."]),
    replace_prefix("GR", "EL"),
    prepend("X"),
    append("Y"),
    insert(2, "0"),
    left_pad(10, "0"),
    when(length_eq(value(), 11), insert(2, "0")),
  ]
}
`)
	p := unit.BySymbol["canonicalizer.vat.common"]
	if p == nil || p.Kind != irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION {
		t.Fatalf("unexpected program %#v", p)
	}
	if len(p.Root.Inputs) != 10 {
		t.Fatalf("got %d steps", len(p.Root.Inputs))
	}
	remove := p.Root.Inputs[3]
	if *remove.Text != "-." {
		t.Fatalf("remove_chars must be deduplicated and sorted, got %q", *remove.Text)
	}
}

func TestCheckFormatCapturesAndCalls(t *testing.T) {
	unit := mustCheck(t, `
format "fr" "siren" {
  checks = [
    require(not(is_empty(subject())), "empty", "fr.siren.empty"),
    require(length_eq(subject(), 9), "invalid_length", "fr.siren.length"),
    require(ascii_digits(subject()), "invalid_characters", "fr.siren.characters"),
  ]
}

format "euid" "fr" {
  capture "registration" {
    value = after_first(value(), ".")
  }

  checks = [
    require(starts_with(value(), "FR"), "invalid_format", "euid.fr.prefix"),
    require(contains(value(), "."), "invalid_format", "euid.fr.separator"),
    require(not(is_absent(capture.registration)), "invalid_format", "euid.fr.registration"),
  ]

  use_format {
    rule  = format.fr.siren
    input = capture.registration
  }
}
`)
	p := unit.BySymbol["format.euid.fr"]
	if len(p.Captures) != 1 || p.Captures[0].Name != "registration" {
		t.Fatalf("unexpected captures %#v", p.Captures)
	}
	last := p.Root.Inputs[len(p.Root.Inputs)-1]
	if last.Op.Category != features.CategoryCall || last.CallTarget != "format.fr.siren" {
		t.Fatalf("unexpected call node %#v", last)
	}
	if last.Inputs[0] != p.Captures[0].Node {
		t.Fatal("a capture reference must be shared, not duplicated")
	}
}

func TestCheckChecksum(t *testing.T) {
	unit := mustCheck(t, `
checksum "fr" "siren" {
  rule = luhn(subject())
}

checksum "vat" "fr" {
  subject = value()

  rule = choose(
    when_checksum(
      ascii_digits(slice(value(), 2, 4)),
      compare_slice(
        remainder_map(mod_digits(slice_from(value(), 4), 97), [12, 15]),
        value(), 2, 4,
      ),
    ),
    unsupported_checksum("checksum_not_published"),
  )
}

checksum "euid" "fr" {
  rule = apply_checksum(checksum.fr.siren, after_first(value(), "."))
}

checksum "vat" "el" {
  rule = all_checks(
    any_check(
      compare_digit(
        remainder_map(modulo(weighted_sum(slice(value(), 2, 10), [256, 128, 64, 32, 16, 8, 4, 2], "left", "digit_value"), 11), [0, 1]),
        value(), 10,
      ),
    ),
  )
}
`)
	fr := unit.BySymbol["checksum.vat.fr"]
	if fr.Subject == nil {
		t.Fatal("explicit subject must be checked")
	}
	if fr.Root.Op.Code != int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_CHOOSE) {
		t.Fatalf("unexpected root %#v", fr.Root.Op)
	}
	euid := unit.BySymbol["checksum.euid.fr"]
	if euid.Root.CallTarget != "checksum.fr.siren" {
		t.Fatalf("unexpected call target %q", euid.Root.CallTarget)
	}
}

func TestTypecheckRejections(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
		msg  string
	}{
		{"unknown function", `checksum "a" "b" { rule = lhun(subject()) }`, typecheck.CodeUnknownFunction, "did you mean luhn"},
		{"unknown function far", `checksum "a" "b" { rule = zzzzzzzzzzzz(subject()) }`, typecheck.CodeUnknownFunction, "unknown function"},
		{"type mismatch", `checksum "a" "b" { rule = luhn(length_eq(subject(), 3)) }`, typecheck.CodeTypeMismatch, ""},
		{"root type", `checksum "a" "b" { rule = subject() }`, typecheck.CodeTypeMismatch, ""},
		{"arity too few", `checksum "a" "b" { rule = compare_digit(digits_to_integer(slice(subject(),0,2)), subject()) }`, typecheck.CodeArity, "missing argument"},
		// luhn now takes an optional message key after its operand, so two
		// arguments are legal and three are not.
		{"arity too many", `checksum "a" "b" { rule = luhn(subject(), "k", "k") }`, typecheck.CodeArity, ""},
		{"message key must be a literal", `checksum "a" "b" { rule = luhn(subject(), subject()) }`, typecheck.CodeBadConstant, "string literal"},
		{"subject in canonicalizer", `canonicalizer "a" "b" { steps = [when(is_empty(subject()), trim_whitespace())] }`, typecheck.CodeContext, "subject() is not available"},
		{"canonicalization in format", `format "a" "b" { checks = [require(is_empty(trim_whitespace()), "empty", "k")] }`, typecheck.CodeContext, "only available in a canonicalizer"},
		{"assertion in checksum", `checksum "a" "b" { rule = require(is_empty(subject()), "empty", "k") }`, typecheck.CodeContext, "only available in a format rule"},
		{"checksum in format", `format "a" "b" { checks = [luhn(subject())] }`, typecheck.CodeContext, "only available in a checksum rule"},
		{"bad reason code", `format "a" "b" { checks = [require(is_empty(subject()), "nope", "k")] }`, typecheck.CodeReasonCode, "unknown reason code"},
		{"forbidden reason code", `format "a" "b" { checks = [require(is_empty(subject()), "unsupported_kind", "k")] }`, typecheck.CodeReasonCode, "not allowed"},
		{"forbidden checksum reason", `checksum "a" "b" { rule = unsupported_checksum("invalid_checksum") }`, typecheck.CodeReasonCode, "not allowed"},
		{"unknown capture", `format "a" "b" { checks = [require(is_empty(capture.nope), "empty", "k")] }`, typecheck.CodeUnknownCapture, "unknown capture"},
		{"capture outside format", `checksum "a" "b" { rule = luhn(capture.x) }`, typecheck.CodeContext, "only available inside the format"},
		{"duplicate capture", "format \"a\" \"b\" {\n capture \"x\" { value = value() }\n capture \"x\" { value = value() }\n checks = [require(is_empty(subject()), \"empty\", \"k\")]\n}", typecheck.CodeDuplicate, "duplicate capture"},
		{"slice bounds", `checksum "a" "b" { rule = luhn(slice(value(), 5, 2)) }`, typecheck.CodeBounds, "greater than end"},
		{"index limit", `checksum "a" "b" { rule = luhn(slice(value(), 0, 99999)) }`, typecheck.CodeBounds, "outside the accepted range"},
		{"modulus limit", `checksum "a" "b" { rule = compare_digit(mod_digits(value(), 1), value(), 0) }`, typecheck.CodeBounds, "modulus"},
		{"weights count", `checksum "a" "b" { rule = compare_digit(weighted_sum(value(), [], "left", "digit_value"), value(), 0) }`, typecheck.CodeBounds, "weights"},
		{"weight magnitude", `checksum "a" "b" { rule = compare_digit(weighted_sum(value(), [2000000], "left", "digit_value"), value(), 0) }`, typecheck.CodeBounds, "magnitude"},
		{"unbounded digits_to_integer", `checksum "a" "b" { rule = compare_digit(digits_to_integer(value()), value(), 0) }`, typecheck.CodeStaticProof, "statically bounded"},
		{"too long digits_to_integer", `checksum "a" "b" { rule = compare_digit(digits_to_integer(slice(value(), 0, 19)), value(), 0) }`, typecheck.CodeStaticProof, "provable limit"},
		{"unbounded cycle", `checksum "a" "b" { rule = compare_digit(weighted_sum(value(), [1,2], "cycle", "digit_value"), value(), 0) }`, typecheck.CodeStaticProof, "cycling weighted_sum"},
		{"bad alignment", `checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), [1,2], "middle", "digit_value"), value(), 0) }`, typecheck.CodeBadConstant, "unknown alignment"},
		{"bad mapping", `checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), [1,2], "left", "hex"), value(), 0) }`, typecheck.CodeBadConstant, "unknown mapping"},
		{"bad profile", `format "a" "b" { checks = [require(profile_is("loose"), "invalid_format", "k")] }`, typecheck.CodeBadConstant, "unknown profile"},
		{"empty constant", `format "a" "b" { checks = [require(starts_with(value(), ""), "invalid_format", "k")] }`, typecheck.CodeBadConstant, "non empty constant"},
		{"non ascii charset", `format "a" "b" { checks = [require(ascii_charset(value(), "é"), "invalid_format", "k")] }`, typecheck.CodeBadConstant, "ASCII"},
		{"length_between order", `format "a" "b" { checks = [require(length_between(value(), 9, 2), "invalid_length", "k")] }`, typecheck.CodeBounds, "greater than maximum"},
		{"left_pad char", `canonicalizer "a" "b" { steps = [left_pad(9, "00")] }`, typecheck.CodeBadConstant, "exactly one padding"},
		{"left_pad length", `canonicalizer "a" "b" { steps = [left_pad(0, "0")] }`, typecheck.CodeBounds, "at least 1"},
		{"replace prefix identity", `canonicalizer "a" "b" { steps = [replace_prefix("GR", "GR")] }`, typecheck.CodeBadConstant, "by itself"},
		{"remove_chars not list", `canonicalizer "a" "b" { steps = [remove_chars(".")] }`, typecheck.CodeBadConstant, "list of single character"},
		{"remove_chars multi rune", `canonicalizer "a" "b" { steps = [remove_chars([".."])] }`, typecheck.CodeBadConstant, "single character"},
		{"remove_chars empty", `canonicalizer "a" "b" { steps = [remove_chars([])] }`, typecheck.CodeBadConstant, "non empty character list"},
		{"prefix_in empty", `format "a" "b" { checks = [require(prefix_in(value(), []), "invalid_format", "k")] }`, typecheck.CodeBadConstant, "non empty list"},
		{"prefix_in not list", `format "a" "b" { checks = [require(prefix_in(value(), "FR"), "invalid_format", "k")] }`, typecheck.CodeBadConstant, "list of string literals"},
		{"length_in not list", `format "a" "b" { checks = [require(length_in(value(), 9), "invalid_length", "k")] }`, typecheck.CodeBadConstant, "list of integer literals"},
		{"length_in empty", `format "a" "b" { checks = [require(length_in(value(), []), "invalid_length", "k")] }`, typecheck.CodeBadConstant, "non empty list"},
		{"length_in bounds", `format "a" "b" { checks = [require(length_in(value(), [99999]), "invalid_length", "k")] }`, typecheck.CodeBounds, "outside the accepted range"},
		{"when_checksum outside choose", `checksum "a" "b" { rule = when_checksum(is_empty(value()), luhn(value())) }`, typecheck.CodeContext, "direct branch of choose"},
		{"when_checksum in all_checks", `checksum "a" "b" { rule = all_checks(when_checksum(is_empty(value()), luhn(value()))) }`, typecheck.CodeContext, "direct branch of choose"},
		{"unknown apply_checksum", `checksum "a" "b" { rule = apply_checksum(checksum.z.z, value()) }`, typecheck.CodeUnknownFunction, "unknown checksum"},
		{"apply_checksum not ref", `checksum "a" "b" { rule = apply_checksum("x", value()) }`, typecheck.CodeBadConstant, "checksum.<namespace>"},
		{"int where string", `format "a" "b" { checks = [require(equals(value(), 3), "invalid_format", "k")] }`, typecheck.CodeTypeMismatch, ""},
		{"list where string", `format "a" "b" { checks = [require(equals(value(), ["a"]), "invalid_format", "k")] }`, typecheck.CodeTypeMismatch, ""},
		{"bool where string", `format "a" "b" { checks = [require(equals(value(), true), "invalid_format", "k")] }`, typecheck.CodeTypeMismatch, ""},
		{"no checks", `format "a" "b" { checks = [] }`, typecheck.CodeArity, "declares no check"},
		{"compare_slice width", `checksum "a" "b" { rule = compare_slice(mod_digits(value(), 97), value(), 0, 19) }`, typecheck.CodeBounds, "provable limit"},
		{"compare_slice order", `checksum "a" "b" { rule = compare_slice(mod_digits(value(), 97), value(), 4, 4) }`, typecheck.CodeBounds, "lower than end"},
		{"bad reference", `format "a" "b" { checks = [require(is_empty(format.a.b), "empty", "k")] }`, typecheck.CodeUnknownCapture, "unexpected reference"},
		{"non literal integer", `canonicalizer "a" "b" { steps = [insert(value(), "0")] }`, typecheck.CodeBadConstant, "integer literal"},
		{"non literal string", `format "a" "b" { checks = [require(starts_with(value(), value()), "invalid_format", "k")] }`, typecheck.CodeBadConstant, "string literal"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, out := check(t, tc.src)
			if !strings.Contains(out, tc.code) {
				t.Fatalf("expected %s, got:\n%s", tc.code, out)
			}
			if tc.msg != "" && !strings.Contains(out, tc.msg) {
				t.Fatalf("expected message %q, got:\n%s", tc.msg, out)
			}
		})
	}
}

func TestPreCanonicalizerRestriction(t *testing.T) {
	src := `
canonicalizer "dispatch" "identifier" {
  steps = [trim_whitespace(), uppercase_ascii(), prepend("X")]
}

format "a" "b" {
  checks = [require(is_empty(subject()), "empty", "k")]
}

checksum "a" "b" {
  rule = luhn(subject())
}

dispatcher "vat" {
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code = "BE"
    identifier   = identifier.vat.BE
  }
}

identifier "vat" "BE" {
  canonicalizer   = canonicalizer.dispatch.identifier
  format          = format.a.b
  checksum        = checksum.a.b
  default_profile = "compatible"
}
`
	if _, out := check(t, src); !strings.Contains(out, "not allowed in the pre-canonicalizer") {
		t.Fatalf("expected a pre-canonicalizer restriction, got:\n%s", out)
	}
}

func TestGlobalDefinitionRejectsPrependCountry(t *testing.T) {
	src := `
canonicalizer "dispatch" "identifier" {
  steps = [trim_whitespace()]
}

canonicalizer "lei" "common" {
  steps = [trim_whitespace(), prepend_country_if_missing()]
}

format "a" "b" {
  checks = [require(is_empty(subject()), "empty", "k")]
}

checksum "a" "b" {
  rule = luhn(subject())
}

dispatcher "lei" {
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    identifier = identifier.lei.GLOBAL
  }
}

identifier "lei" "GLOBAL" {
  canonicalizer   = canonicalizer.lei.common
  format          = format.a.b
  checksum        = checksum.a.b
  default_profile = "compatible"
}
`
	if _, out := check(t, src); !strings.Contains(out, "cannot be used by the GLOBAL definition") {
		t.Fatalf("expected a GLOBAL restriction, got:\n%s", out)
	}
}

func TestSurfaceNamesCoverCatalog(t *testing.T) {
	names := map[string]bool{}
	for _, n := range typecheck.SurfaceNames() {
		names[n] = true
	}
	for _, op := range features.Ops() {
		name := op.HCLName()
		if name == "" || name == "use_format" {
			continue
		}
		if !names[name] {
			t.Fatalf("catalog operation %s has no surface signature", op.Symbol)
		}
	}
}

func TestTypecheckRemainingRejections(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
		msg  string
	}{
		{"string where a predicate is expected",
			`format "a" "b" { checks = [require("x", "empty", "k")] }`,
			typecheck.CodeTypeMismatch, "string literal"},
		{"oversized string constant",
			`format "a" "b" { checks = [require(equals(value(), "` + strings.Repeat("x", 5000) + `"), "invalid_format", "k")] }`,
			typecheck.CodeLimit, "exceeds the limit"},
		{"capture where a predicate is expected",
			"format \"a\" \"b\" {\n capture \"x\" { value = value() }\n checks = [require(capture.x, \"empty\", \"k\")]\n}",
			typecheck.CodeTypeMismatch, "a capture is a string"},
		{"oversized message key",
			`format "a" "b" { checks = [require(is_empty(value()), "empty", "` + strings.Repeat("k", 5000) + `")] }`,
			typecheck.CodeLimit, "exceeds"},
		{"duplicate prefix_in values",
			`format "a" "b" { checks = [require(prefix_in(value(), ["A", "A", "B"]), "invalid_format", "k")] }`,
			"", ""},
		{"duplicate length_in values",
			`format "a" "b" { checks = [require(length_in(value(), [9, 9, 10]), "invalid_length", "k")] }`,
			"", ""},
		{"negative length_in value",
			`format "a" "b" { checks = [require(length_in(value(), [-1]), "invalid_length", "k")] }`,
			typecheck.CodeBounds, "outside the accepted range"},
		{"remainder table too large is impossible but an empty one is refused",
			`checksum "a" "b" { rule = compare_digit(remainder_map(mod_digits(value(), 97), []), value(), 0) }`,
			typecheck.CodeBounds, "remainder values"},
		{"canonicalization step inside a checksum",
			`checksum "a" "b" { rule = luhn(trim_whitespace()) }`,
			typecheck.CodeContext, "only available in a canonicalizer"},
		{"assertion inside a canonicalizer",
			`canonicalizer "a" "b" { steps = [require(is_empty(value()), "empty", "k")] }`,
			typecheck.CodeContext, "only available in a format rule"},
		{"checksum inside a canonicalizer",
			`canonicalizer "a" "b" { steps = [luhn(value())] }`,
			typecheck.CodeContext, "only available in a checksum rule"},
		{"canonicalization step inside a format",
			`format "a" "b" { checks = [trim_whitespace()] }`,
			typecheck.CodeContext, "only available in a canonicalizer"},
		{"unknown alignment parameter kind",
			`checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), [1], "left", "digit_value"), value(), 0) }`,
			"", ""},
		{"non list weights",
			`checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), 1, "left", "digit_value"), value(), 0) }`,
			typecheck.CodeBadConstant, "list of integer literals"},
		{"non literal enum",
			`checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), [1], value(), "digit_value"), value(), 0) }`,
			typecheck.CodeBadConstant, "string literal"},
		{"too many captures", manyCaptures(200), typecheck.CodeLimit, "captures"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, out := check(t, tc.src)
			if tc.code == "" {
				if out != "" {
					t.Fatalf("unexpected diagnostics:\n%s", out)
				}
				return
			}
			if !strings.Contains(out, tc.code) {
				t.Fatalf("expected %s, got:\n%s", tc.code, out)
			}
			if tc.msg != "" && !strings.Contains(out, tc.msg) {
				t.Fatalf("expected %q, got:\n%s", tc.msg, out)
			}
		})
	}
}

func manyCaptures(n int) string {
	var sb strings.Builder
	sb.WriteString("format \"a\" \"b\" {\n")
	for i := 0; i < n; i++ {
		sb.WriteString("  capture \"c")
		sb.WriteString(itoa(i))
		sb.WriteString("\" {\n    value = value()\n  }\n")
	}
	sb.WriteString("  checks = [require(is_empty(subject()), \"empty\", \"k\")]\n}\n")
	return sb.String()
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var digits []byte
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	return string(digits)
}

func TestSurfaceNamesAreSorted(t *testing.T) {
	names := typecheck.SurfaceNames()
	if len(names) == 0 {
		t.Fatal("the surface language declares no function")
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatal("the surface names must be sorted and unique")
		}
	}
}

func TestEmptyConstantsAreRefusedEverywhere(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{"before_first", `format "a" "b" { checks = [require(is_empty(before_first(value(), "")), "empty", "k")] }`},
		{"after_first", `format "a" "b" { checks = [require(is_empty(after_first(value(), "")), "empty", "k")] }`},
		{"strip_prefix", `format "a" "b" { checks = [require(is_empty(strip_prefix(value(), "")), "empty", "k")] }`},
		{"ends_with", `format "a" "b" { checks = [require(ends_with(value(), ""), "invalid_format", "k")] }`},
		{"contains", `format "a" "b" { checks = [require(contains(value(), ""), "invalid_format", "k")] }`},
		{"ascii_charset", `format "a" "b" { checks = [require(ascii_charset(value(), ""), "invalid_format", "k")] }`},
		{"char_at_in", `format "a" "b" { checks = [require(char_at_in(value(), 0, ""), "invalid_format", "k")] }`},
		{"prepend", `canonicalizer "a" "b" { steps = [prepend("")] }`},
		{"append", `canonicalizer "a" "b" { steps = [append("")] }`},
		{"insert", `canonicalizer "a" "b" { steps = [insert(0, "")] }`},
		{"replace_prefix from", `canonicalizer "a" "b" { steps = [replace_prefix("", "X")] }`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, out := check(t, tc.src)
			if !strings.Contains(out, typecheck.CodeBadConstant) {
				t.Fatalf("expected an empty constant rejection, got:\n%s", out)
			}
		})
	}
}

func TestStaticLengthBoundsPropagate(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		accepted bool
	}{
		{"slice bound", `checksum "a" "b" { rule = compare_digit(digits_to_integer(slice(value(), 0, 9)), value(), 0) }`, true},
		{"slice_to bound", `checksum "a" "b" { rule = compare_digit(digits_to_integer(slice_to(value(), 9)), value(), 0) }`, true},
		{"slice_from of a bounded view", `checksum "a" "b" { rule = compare_digit(digits_to_integer(slice_from(slice(value(), 0, 9), 2)), value(), 0) }`, true},
		{"strip_prefix keeps the bound", `checksum "a" "b" { rule = compare_digit(digits_to_integer(strip_prefix(slice(value(), 0, 9), "FR")), value(), 0) }`, true},
		{"concat of bounded views", `checksum "a" "b" { rule = compare_digit(digits_to_integer(concat(slice(value(), 0, 4), slice(value(), 4, 8))), value(), 0) }`, true},
		{"concat with an unbounded view", `checksum "a" "b" { rule = compare_digit(digits_to_integer(concat(slice(value(), 0, 4), value())), value(), 0) }`, false},
		{"cycling weighted sum over a bounded view", `checksum "a" "b" { rule = compare_digit(modulo(weighted_sum(slice(value(), 0, 8), [1, 2], "cycle", "digit_value"), 10), value(), 0) }`, true},
		{"country code is bounded", `checksum "a" "b" { rule = compare_digit(digits_to_integer(country_code()), value(), 0) }`, true},
		{"constant is bounded", `checksum "a" "b" { rule = compare_digit(digits_to_integer(concat("12", "34")), value(), 0) }`, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, out := check(t, tc.src)
			if tc.accepted && out != "" {
				t.Fatalf("unexpected diagnostics:\n%s", out)
			}
			if !tc.accepted && !strings.Contains(out, typecheck.CodeStaticProof) {
				t.Fatalf("expected a static proof rejection, got:\n%s", out)
			}
		})
	}
}

// CUSTOM_ALPHABET reads the value of a code point from the alphabet the
// operation carries, so the alphabet has to be there, has to be usable, and
// has to be absent when another mapping would ignore it.
func TestCustomAlphabetIsCheckedAtCompileTime(t *testing.T) {
	const uscc = "0123456789ABCDEFGHJKLMNPQRTUWXY"
	for name, tc := range map[string]struct {
		src  string
		want string
	}{
		"accepted": {
			`checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), [1,3], "left", "custom_alphabet", "` + uscc + `"), value(), 0) }`,
			"",
		},
		"missing alphabet": {
			`checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), [1,3], "left", "custom_alphabet"), value(), 0) }`,
			"alphabet",
		},
		"alphabet without the mapping that reads it": {
			`checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), [1,3], "left", "digit_value", "` + uscc + `"), value(), 0) }`,
			"alphabet",
		},
		"empty alphabet": {
			`checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), [1,3], "left", "custom_alphabet", ""), value(), 0) }`,
			"alphabet",
		},
		// A repeated code point gives one character two values, so the sum it
		// produces depends on which index the implementation happens to find.
		"repeated code point": {
			`checksum "a" "b" { rule = compare_digit(weighted_sum(slice(value(),0,2), [1,3], "left", "custom_alphabet", "0123401234"), value(), 0) }`,
			"twice",
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, out := check(t, tc.src)
			if tc.want == "" {
				if out != "" {
					t.Fatalf("expected acceptance, got:\n%s", out)
				}
				return
			}
			if out == "" {
				t.Fatal("expected a compile error")
			}
			if !strings.Contains(out, tc.want) {
				t.Fatalf("expected an error mentioning %q, got:\n%s", tc.want, out)
			}
		})
	}
}
