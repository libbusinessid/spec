package hcllang_test

import (
	"testing"

	"github.com/entid-org/spec/internal/ast"
	"github.com/entid-org/spec/internal/hcllang"
	"github.com/entid-org/spec/internal/linker"
)

// FuzzParseHCL feeds arbitrary bytes to the whole front end. No input may cause
// a panic, an infinite loop or an unbounded allocation.
func FuzzParseHCL(f *testing.F) {
	seeds := []string{
		"",
		"canonicalizer \"a\" \"b\" { steps = [] }",
		validUnitSeed,
		"format \"a\" \"b\" { checks = [require(is_empty(subject()), \"empty\", \"k\")] }",
		"dispatcher \"vat\" { pre_canonicalizer = canonicalizer.a.b\n target { identifier = identifier.vat.BE } }",
		"identifier \"vat\" \"GLOBAL\" { canonicalizer = canonicalizer.a.b\n format = format.a.b }",
		"checksum \"a\" \"b\" { rule = choose(when_checksum(is_empty(value()), luhn(value())), unsupported_checksum(\"unsupported_checksum\")) }",
		"a = ${b}",
		"\x00\x01\x02",
		"canonicalizer \"a\" \"b\" { steps = [insert(99999999999999999999, \"x\")] }",
		"format \"\" \"\" { checks = [] }",
		"identifier \"vat\" \"BE\" { checksum = checksum.a.b }",
		"canonicalizer \"a\" \"b\" { steps = [when(profile_is(\"compatible\"), when(profile_is(\"compatible\"), trim_whitespace()))] }",
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<16 {
			t.Skip("oversized input")
		}
		file, bag := hcllang.ParseFile("fuzz.hcl", data)
		if file == nil {
			if !bag.HasErrors() {
				t.Fatal("a rejected file must report at least one diagnostic")
			}
			return
		}
		// The linker must also survive an arbitrary but parseable unit.
		linker.Link(&ast.Unit{Files: []*ast.File{file}})
	})
}

const validUnitSeed = `
canonicalizer "dispatch" "identifier" {
  steps = [trim_whitespace(), remove_whitespace(), uppercase_ascii()]
}

format "a" "b" {
  capture "x" {
    value = after_first(subject(), ".")
  }
  checks = [
    require(not(is_empty(subject())), "empty", "a.b.empty"),
    require(length_between(capture.x, 1, 9), "invalid_length", "a.b.length"),
  ]
}

checksum "a" "b" {
  subject = slice_from(value(), 2)
  rule = luhn(subject())
}

dispatcher "demo" {
  aliases           = ["demo_id"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    country_code      = "BE"
    accepted_prefixes = ["BE"]
    canonical_prefix  = "BE"
    identifier        = identifier.demo.BE
  }
}

identifier "demo" "BE" {
  canonicalizer   = canonicalizer.dispatch.identifier
  format          = format.a.b
  checksum        = checksum.a.b
  default_profile = "compatible"

  source {
    id               = "s"
    url              = "https://example.invalid"
    authority        = "a"
    title            = "t"
    accessed_at      = "2026-08-18"
    jurisdiction     = "BE"
    language         = "en"
    notes            = "n"
    license_or_terms = "l"
    tier = "primary"
  }
}
`
