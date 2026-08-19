package typecheck_test

import (
	"testing"

	"github.com/libbusinessid/spec/internal/ast"
	"github.com/libbusinessid/spec/internal/hcllang"
	"github.com/libbusinessid/spec/internal/linker"
	"github.com/libbusinessid/spec/internal/lower"
	"github.com/libbusinessid/spec/internal/typecheck"
)

// FuzzCompileUnit drives the whole compiler on arbitrary sources, including
// deeply nested and cyclic reference graphs.
func FuzzCompileUnit(f *testing.F) {
	seeds := []string{
		"",
		"checksum \"a\" \"b\" { rule = apply_checksum(checksum.a.b, value()) }",
		"format \"a\" \"b\" { checks = [require(is_empty(subject()), \"empty\", \"k\")]\n use_format { rule = format.a.b\n input = subject() } }",
		"canonicalizer \"a\" \"b\" { steps = [when(all(any(not(is_empty(value())))), trim_whitespace())] }",
		"checksum \"a\" \"b\" { rule = all_checks(any_check(choose(when_checksum(is_empty(value()), luhn(value())), unsupported_checksum(\"unsupported_checksum\")))) }",
		"format \"a\" \"b\" { checks = [require(prefix_in(concat(value(), value(), value()), [\"A\", \"B\"]), \"invalid_format\", \"k\")] }",
		"checksum \"a\" \"b\" { rule = compare_slice(remainder_map(modulo(weighted_sum(slice(value(), 0, 8), [1,2,3], \"cycle\", \"alnum_base36\"), 97), [1,2,3]), value(), 0, 2) }",
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<16 {
			t.Skip("oversized input")
		}
		file, bag := hcllang.ParseFile("fuzz.hcl", data)
		if file == nil || bag.HasErrors() {
			return
		}
		table, linkBag := linker.Link(&ast.Unit{Files: []*ast.File{file}})
		unit, typeBag := typecheck.Check(table)
		if linkBag.HasErrors() || typeBag.HasErrors() {
			return
		}
		bundle, lowerBag := lower.Lower(table, unit, lower.Options{RulesVersion: "2026.08.0", Optimize: true})
		if lowerBag.HasErrors() {
			return
		}
		if bundle == nil {
			t.Fatal("a successful lowering must produce a bundle")
		}
	})
}
