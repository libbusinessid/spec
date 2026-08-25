package linker_test

import (
	"strings"
	"testing"

	"github.com/entid-org/spec/internal/ast"
	"github.com/entid-org/spec/internal/hcllang"
	"github.com/entid-org/spec/internal/linker"
)

const validUnit = `
canonicalizer "dispatch" "identifier" {
  steps = [trim_whitespace(), remove_whitespace(), uppercase_ascii(), remove_chars([".", "-"])]
}

canonicalizer "vat" "common" {
  steps = [trim_whitespace(), uppercase_ascii(), prepend_country_if_missing()]
}

format "vat" "be" {
  checks = [require(length_eq(subject(), 12), "invalid_length", "vat.be.length")]
}

checksum "vat" "be" {
  rule = luhn(subject())
}

dispatcher "vat" {
  aliases           = ["vat_id"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  country_aliases = {
    "UK" = "GB"
  }

  target {
    country_code      = "BE"
    accepted_prefixes = ["BE"]
    canonical_prefix  = "BE"
    identifier        = identifier.vat.BE
  }
}

identifier "vat" "BE" {
  canonicalizer   = canonicalizer.vat.common
  format          = format.vat.be
  checksum        = checksum.vat.be
  default_profile = "compatible"

  source {
    id               = "be-vat"
    url              = "https://example.invalid/be"
    authority        = "Belgian tax authority"
    title            = "VAT numbers"
    accessed_at      = "2026-08-18"
    jurisdiction     = "BE"
    language         = "en"
    notes            = "official"
    license_or_terms = "public sector information"
    tier = "primary"
  }
}
`

func link(t *testing.T, src string) (*linker.Table, string) {
	t.Helper()
	file, bag := hcllang.ParseFile("rules.hcl", []byte(src))
	if bag.HasErrors() {
		t.Fatalf("parse errors: %v", bag.Sorted())
	}
	return linkParsed(t, file)
}

// linkTolerant links a source that may also hold parser diagnostics, so that a
// single table can cover both stages.
func linkTolerant(t *testing.T, src string) string {
	t.Helper()
	file, bag := hcllang.ParseFile("rules.hcl", []byte(src))
	var sb strings.Builder
	for _, d := range bag.Sorted() {
		sb.WriteString(d.Code + ": " + d.Message + " | " + d.Suggestion + "\n")
	}
	if file == nil {
		return sb.String()
	}
	_, out := linkParsed(t, file)
	return sb.String() + out
}

func linkParsed(t *testing.T, file *ast.File) (*linker.Table, string) {
	t.Helper()
	table, lbag := linker.Link(&ast.Unit{Files: []*ast.File{file}})
	var sb strings.Builder
	for _, d := range lbag.Sorted() {
		sb.WriteString(d.Code + ": " + d.Message + " | " + d.Suggestion + "\n")
	}
	return table, sb.String()
}

func TestLinkValidUnit(t *testing.T) {
	table, out := link(t, validUnit)
	if out != "" {
		t.Fatalf("unexpected diagnostics:\n%s", out)
	}
	if len(table.Dispatchers) != 1 || len(table.IdentifierOrder) != 1 {
		t.Fatalf("unexpected table: %#v", table)
	}
	if table.TargetOf["identifier.vat.BE"] == nil || table.DispatcherOf["identifier.vat.BE"] == nil {
		t.Fatal("target binding is missing")
	}
}

func TestLinkRejections(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(string) string
		code    string
		message string
	}{
		{"duplicate symbol", func(s string) string {
			return s + "\nformat \"vat\" \"be\" {\n checks = [require(is_empty(subject()), \"empty\", \"k\")]\n}\n"
		}, linker.CodeDuplicateSymbol, "duplicate symbol"},
		{"unknown symbol", func(s string) string {
			return strings.Replace(s, "format          = format.vat.be", "format          = format.vat.bee", 1)
		}, linker.CodeUnknownSymbol, "unknown symbol"},
		{"wrong reference family", func(s string) string {
			return strings.Replace(s, "format          = format.vat.be", "format          = checksum.vat.be", 1)
		}, linker.CodeUnknownSymbol, "expected a format"},
		{"bad country label", func(s string) string {
			return strings.Replace(s, `identifier "vat" "BE"`, `identifier "vat" "be"`, 1)
		}, linker.CodeBadLabel, "invalid identifier country label"},
		{"bad kind label", func(s string) string {
			return strings.Replace(s, `dispatcher "vat"`, `dispatcher "VAT"`, 1)
		}, linker.CodeBadLabel, "invalid dispatcher kind"},
		{"orphan definition", func(s string) string {
			return s + "\nidentifier \"vat\" \"NL\" {\n canonicalizer = canonicalizer.vat.common\n format = format.vat.be\n checksum = checksum.vat.be\n default_profile = \"compatible\"\n}\n"
		}, linker.CodeOrphanDefinition, "not referenced by any dispatcher target"},
		{"definition claimed twice", func(s string) string {
			return strings.Replace(s, "    identifier        = identifier.vat.BE\n",
				"    identifier        = identifier.vat.BE\n  }\n\n  target {\n    country_code      = \"NL\"\n    identifier        = identifier.vat.BE\n", 1)
		}, linker.CodeDispatch, "is already referenced by dispatcher"},
		{"kind mismatch", func(s string) string {
			return strings.Replace(s, `identifier "vat" "BE" {`, `identifier "euid" "BE" {`, 1)
		}, linker.CodeUnknownSymbol, ""},
		{"duplicate country target", func(s string) string {
			return strings.Replace(s, "  target {\n    country_code      = \"BE\"",
				"  target {\n    country_code      = \"BE\"\n    identifier        = identifier.vat.BE\n  }\n\n  target {\n    country_code      = \"BE\"", 1)
		}, linker.CodeDispatch, "two targets declare country"},
		{"duplicate prefix", func(s string) string {
			return strings.Replace(s, `accepted_prefixes = ["BE"]`, `accepted_prefixes = ["BE", "BE"]`, 1)
		}, linker.CodeDispatch, "claimed by two targets"},
		{"lowercase prefix", func(s string) string {
			return strings.Replace(s, `accepted_prefixes = ["BE"]`, `accepted_prefixes = ["be"]`, 1)
		}, linker.CodePrefixCase, "can never match"},
		{"canonical prefix not accepted", func(s string) string {
			return strings.Replace(s, `canonical_prefix  = "BE"`, `canonical_prefix  = "BX"`, 1)
		}, linker.CodeDispatch, "not part of accepted_prefixes"},
		{"self country alias", func(s string) string {
			return strings.Replace(s, `"UK" = "GB"`, `"GB" = "GB"`, 1)
		}, linker.CodeDispatch, "maps to itself"},
		{"country alias shadows target", func(s string) string {
			return strings.Replace(s, `"UK" = "GB"`, `"BE" = "NL"`, 1)
		}, linker.CodeDispatch, "shadows a target country"},
		{"invalid country alias", func(s string) string {
			return strings.Replace(s, `"UK" = "GB"`, `"UKX" = "GB"`, 1)
		}, linker.CodeBadLabel, "invalid country alias"},
		{"missing profile", func(s string) string {
			return strings.Replace(s, "  default_profile = \"compatible\"\n", "", 1)
		}, linker.CodeProfile, "must declare default_profile"},
		{"bad profile", func(s string) string {
			return strings.Replace(s, `default_profile = "compatible"`, `default_profile = "loose"`, 1)
		}, linker.CodeProfile, "invalid default_profile"},
		{"missing source", func(s string) string {
			idx := strings.Index(s, "  source {")
			return s[:idx] + "}\n"
		}, linker.CodeMissingSource, "declares no source"},
		{"bad accessed_at", func(s string) string {
			return strings.Replace(s, `accessed_at      = "2026-08-18"`, `accessed_at      = "18/08/2026"`, 1)
		}, linker.CodeSourceField, "ISO 8601"},
		{"empty source field", func(s string) string {
			return strings.Replace(s, `authority        = "Belgian tax authority"`, `authority        = ""`, 1)
		}, linker.CodeSourceField, "must not be empty"},
		{"both checksum and no_checksum", func(s string) string {
			return strings.Replace(s, "  default_profile = \"compatible\"\n",
				"  default_profile = \"compatible\"\n\n  no_checksum {\n    reason_code = \"unsupported_checksum\"\n    notes = \"x\"\n  }\n", 1)
		}, linker.CodeChecksumDecl, "both a checksum"},
		{"no checksum declaration", func(s string) string {
			return strings.Replace(s, "  checksum        = checksum.vat.be\n", "", 1)
		}, linker.CodeChecksumDecl, "must declare a checksum"},
		{"global mixed", func(s string) string {
			return strings.Replace(s, "  target {\n    country_code      = \"BE\"",
				"  target {\n    identifier = identifier.vat.BE\n  }\n\n  target {\n    country_code      = \"BE\"", 1)
		}, linker.CodeDispatch, ""},
		{"duplicate kind alias", func(s string) string {
			return strings.Replace(s, `aliases           = ["vat_id"]`, `aliases           = ["vat_id", "vat_id"]`, 1)
		}, linker.CodeDispatch, "duplicate kind alias"},
		{"invalid kind alias", func(s string) string {
			return strings.Replace(s, `aliases           = ["vat_id"]`, `aliases           = ["VAT"]`, 1)
		}, linker.CodeBadLabel, "invalid kind alias"},
		{"alias equals kind", func(s string) string {
			return strings.Replace(s, `aliases           = ["vat_id"]`, `aliases           = ["vat"]`, 1)
		}, linker.CodeDispatch, "duplicate kind alias"},
		{"invalid prefix", func(s string) string {
			return strings.Replace(s, `accepted_prefixes = ["BE"]`, `accepted_prefixes = ["B-E"]`, 1)
		}, linker.CodeBadLabel, "invalid prefix"},
		{"no target", func(s string) string {
			start := strings.Index(s, "  target {")
			end := strings.Index(s, "identifier \"vat\" \"BE\"")
			return s[:start] + "}\n\n" + s[end:]
		}, linker.CodeDispatch, "declares no target"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, out := link(t, tc.mutate(validUnit))
			if !strings.Contains(out, tc.code) {
				t.Fatalf("expected %s, got:\n%s", tc.code, out)
			}
			if tc.message != "" && !strings.Contains(out, tc.message) {
				t.Fatalf("expected message %q, got:\n%s", tc.message, out)
			}
		})
	}
}

func TestGlobalDispatcher(t *testing.T) {
	src := `
canonicalizer "dispatch" "identifier" {
  steps = [trim_whitespace(), uppercase_ascii()]
}

canonicalizer "lei" "common" {
  steps = [trim_whitespace(), uppercase_ascii()]
}

format "lei" "generic" {
  checks = [require(length_eq(subject(), 20), "invalid_length", "lei.length")]
}

checksum "lei" "generic" {
  rule = iso7064_mod97_10(subject())
}

dispatcher "lei" {
  pre_canonicalizer = canonicalizer.dispatch.identifier

  target {
    identifier = identifier.lei.GLOBAL
  }
}

identifier "lei" "GLOBAL" {
  canonicalizer   = canonicalizer.lei.common
  format          = format.lei.generic
  checksum        = checksum.lei.generic
  default_profile = "compatible"

  source {
    id               = "iso-17442"
    url              = "https://example.invalid/lei"
    authority        = "ISO"
    title            = "ISO 17442"
    accessed_at      = "2026-08-18"
    jurisdiction     = "GLOBAL"
    language         = "en"
    notes            = "structure"
    license_or_terms = "standard"
    tier = "primary"
  }
}
`
	if _, out := link(t, src); out != "" {
		t.Fatalf("unexpected diagnostics:\n%s", out)
	}
	bad := strings.Replace(src, "    identifier = identifier.lei.GLOBAL",
		"    accepted_prefixes = [\"LEI\"]\n    identifier = identifier.lei.GLOBAL", 1)
	if _, out := link(t, bad); !strings.Contains(out, "GLOBAL target must not declare prefixes") {
		t.Fatalf("expected a prefix rejection, got:\n%s", out)
	}
}

func TestCycleDetection(t *testing.T) {
	src := `
format "a" "one" {
  checks = [require(is_empty(subject()), "empty", "a")]
  use_format {
    rule  = format.b.two
    input = subject()
  }
}

format "b" "two" {
  checks = [require(is_empty(subject()), "empty", "b")]
  use_format {
    rule  = format.a.one
    input = subject()
  }
}
`
	if _, out := link(t, src); !strings.Contains(out, linker.CodeCycle) {
		t.Fatalf("expected a cycle diagnostic, got:\n%s", out)
	}
}

func TestChecksumCycleDetection(t *testing.T) {
	src := `
checksum "a" "one" {
  rule = apply_checksum(checksum.a.one, subject())
}
`
	if _, out := link(t, src); !strings.Contains(out, linker.CodeCycle) {
		t.Fatalf("expected a self reference cycle, got:\n%s", out)
	}
}

func TestUnknownCallTargets(t *testing.T) {
	src := `
format "a" "one" {
  checks = [require(is_empty(subject()), "empty", "a")]
  use_format {
    rule  = format.b.missing
    input = subject()
  }
}

checksum "a" "one" {
  rule = apply_checksum(checksum.b.missing, subject())
}
`
	_, out := link(t, src)
	if strings.Count(out, linker.CodeUnknownSymbol) != 2 {
		t.Fatalf("expected two unknown symbol diagnostics, got:\n%s", out)
	}
}

func TestLinkerDeclarationRejections(t *testing.T) {
	identifierBlock := func(kind, country, body string) string {
		return "identifier \"" + kind + "\" \"" + country + "\" {\n" +
			"  canonicalizer   = canonicalizer.a.b\n" +
			"  format          = format.a.b\n" +
			"  default_profile = \"compatible\"\n" +
			body + "}\n"
	}
	target := func(body string) string {
		return "  target {\n" + body + "  }\n"
	}
	dispatcher := func(kind, body string) string {
		return "dispatcher \"" + kind + "\" {\n  pre_canonicalizer = canonicalizer.a.b\n" + body + "}\n"
	}
	tests := []struct {
		name string
		src  string
		code string
		msg  string
	}{
		{"duplicate canonicalizer", "canonicalizer \"a\" \"b\" {\n  steps = []\n}\ncanonicalizer \"a\" \"b\" {\n  steps = []\n}\n",
			linker.CodeDuplicateSymbol, "canonicalizer.a.b"},
		{"duplicate checksum", "checksum \"a\" \"b\" {\n  rule = luhn(value())\n}\nchecksum \"a\" \"b\" {\n  rule = luhn(value())\n}\n",
			linker.CodeDuplicateSymbol, "checksum.a.b"},
		{"duplicate identifier", identifierBlock("vat", "BE", "") + identifierBlock("vat", "BE", ""),
			linker.CodeDuplicateSymbol, "identifier.vat.BE"},
		{"duplicate dispatcher", dispatcher("vat", "") + dispatcher("vat", ""),
			linker.CodeDuplicateSymbol, "dispatcher.vat"},
		{"invalid namespace", "canonicalizer \"1a\" \"b\" {\n  steps = []\n}\n", linker.CodeBadLabel, "namespace label"},
		{"invalid name", "canonicalizer \"a\" \"1b\" {\n  steps = []\n}\n", linker.CodeBadLabel, "name label"},
		{"invalid format namespace", "format \"1a\" \"b\" {\n  checks = []\n}\n", linker.CodeBadLabel, "namespace label"},
		{"invalid checksum namespace", "checksum \"1a\" \"b\" {\n  rule = luhn(value())\n}\n", linker.CodeBadLabel, "namespace label"},
		{"invalid identifier kind", identifierBlock("VAT", "BE", ""), linker.CodeBadLabel, "identifier kind"},
		{"canonicalizer reference form", "identifier \"vat\" \"BE\" {\n  canonicalizer = format.a.b\n  format = format.a.b\n}\n",
			linker.CodeUnknownSymbol, "expected a canonicalizer"},
		{"unknown canonicalizer", identifierBlock("vat", "BE", ""), linker.CodeUnknownSymbol, "unknown symbol"},
		{"checksum reference form", "identifier \"vat\" \"BE\" {\n  canonicalizer = canonicalizer.a.b\n  format = format.a.b\n  checksum = format.a.b\n}\n",
			linker.CodeUnknownSymbol, "expected a checksum"},
		{"identifier reference form", dispatcher("vat", target("    identifier = format.a.b\n")),
			linker.CodeUnknownSymbol, "expected an identifier"},
		{"empty no_checksum notes", identifierBlock("vat", "BE",
			"  no_checksum {\n    reason_code = \"unsupported_checksum\"\n    notes       = \"  \"\n  }\n"),
			linker.CodeChecksumDecl, "document why"},
		{"duplicate source id", identifierBlock("vat", "BE", sourceBlock("s")+sourceBlock("s")),
			linker.CodeSourceField, "duplicate source id"},
		{"invalid country alias target",
			dispatcher("vat", "  country_aliases = {\n    \"UK\" = \"gb\"\n  }\n"+
				target("    country_code = \"BE\"\n    identifier   = identifier.vat.BE\n")),
			linker.CodeBadLabel, "must map to an ISO"},
		{"duplicate country alias",
			dispatcher("vat", "  country_aliases = {\n    \"UK\" = \"GB\"\n    \"UK\" = \"IE\"\n  }\n"+
				target("    country_code = \"BE\"\n    identifier   = identifier.vat.BE\n")),
			linker.CodeDispatch, "duplicate country alias"},
		{"invalid target country",
			dispatcher("vat", target("    country_code = \"be\"\n    identifier   = identifier.vat.BE\n")),
			linker.CodeBadLabel, "invalid target country_code"},
		{"invalid canonical prefix",
			dispatcher("vat", target("    country_code     = \"BE\"\n    canonical_prefix = \"B-E\"\n    identifier       = identifier.vat.BE\n")),
			linker.CodeBadLabel, "invalid canonical_prefix"},
		{"two implicit targets",
			dispatcher("vat",
				target("    country_code                     = \"BE\"\n    allow_unprefixed_without_country = true\n    identifier                       = identifier.vat.BE\n")+
					target("    country_code                     = \"FR\"\n    allow_unprefixed_without_country = true\n    identifier                       = identifier.vat.FR\n")) +
				identifierBlock("vat", "BE", "") + identifierBlock("vat", "FR", ""),
			linker.CodeDispatch, "implicit targets"},
		{"kind mismatch in a target",
			dispatcher("vat", target("    country_code = \"BE\"\n    identifier   = identifier.euid.BE\n")) +
				identifierBlock("euid", "BE", ""),
			linker.CodeDispatch, "of kind"},
		{"country target on a global definition",
			dispatcher("vat", target("    country_code = \"BE\"\n    identifier   = identifier.vat.GLOBAL\n")) +
				identifierBlock("vat", "GLOBAL", ""),
			linker.CodeDispatch, "references the GLOBAL definition"},
		{"global target on a country definition",
			dispatcher("vat", target("    identifier = identifier.vat.BE\n")) + identifierBlock("vat", "BE", ""),
			linker.CodeDispatch, "GLOBAL target references the country definition"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := linkTolerant(t, tc.src)
			if !strings.Contains(out, tc.code) {
				t.Fatalf("expected %s, got:\n%s", tc.code, out)
			}
			if !strings.Contains(out, tc.msg) {
				t.Fatalf("expected %q, got:\n%s", tc.msg, out)
			}
		})
	}
}

func sourceBlock(id string) string {
	return "  source {\n    id = \"" + id + "\"\n    url = \"https://example.invalid\"\n" +
		"    authority = \"a\"\n    title = \"t\"\n    accessed_at = \"2026-08-18\"\n" +
		"    jurisdiction = \"BE\"\n    language = \"en\"\n    notes = \"n\"\n    license_or_terms = \"l\"\n  }\n"
}

func TestFormatWithoutRequireDoesNotNeedASource(t *testing.T) {
	src := `
canonicalizer "a" "b" {
  steps = [trim_whitespace()]
}

format "a" "b" {
  checks = []
}

checksum "a" "b" {
  rule = luhn(subject())
}

dispatcher "demo" {
  pre_canonicalizer = canonicalizer.a.b

  target {
    identifier = identifier.demo.GLOBAL
  }
}

identifier "demo" "GLOBAL" {
  canonicalizer   = canonicalizer.a.b
  format          = format.a.b
  checksum        = checksum.a.b
  default_profile = "compatible"
}
`
	if _, out := link(t, src); strings.Contains(out, linker.CodeMissingSource) {
		t.Fatalf("a rule that cannot reject an input needs no source:\n%s", out)
	}
}

func TestFormatWithOnlyAUseNeedsASource(t *testing.T) {
	src := `
canonicalizer "a" "b" {
  steps = [trim_whitespace()]
}

format "a" "b" {
  checks = []
  use_format {
    rule  = format.c.d
    input = subject()
  }
}

format "c" "d" {
  checks = [require(is_empty(subject()), "empty", "c.d.empty")]
}

checksum "a" "b" {
  rule = luhn(subject())
}

dispatcher "demo" {
  pre_canonicalizer = canonicalizer.a.b

  target {
    identifier = identifier.demo.GLOBAL
  }
}

identifier "demo" "GLOBAL" {
  canonicalizer   = canonicalizer.a.b
  format          = format.a.b
  checksum        = checksum.a.b
  default_profile = "compatible"
}
`
	if _, out := link(t, src); !strings.Contains(out, linker.CodeMissingSource) {
		t.Fatalf("a rule reusing a rejecting rule needs a source:\n%s", out)
	}
}

func TestCallDepthLimit(t *testing.T) {
	var sb strings.Builder
	const depth = 40
	for i := 0; i < depth; i++ {
		sb.WriteString("format \"chain\" \"n" + itoa(i) + "\" {\n")
		sb.WriteString("  checks = [require(is_empty(subject()), \"empty\", \"k\")]\n")
		if i+1 < depth {
			sb.WriteString("  use_format {\n    rule  = format.chain.n" + itoa(i+1) + "\n    input = subject()\n  }\n")
		}
		sb.WriteString("}\n")
	}
	if _, out := link(t, sb.String()); !strings.Contains(out, linker.CodeCallDepth) {
		t.Fatalf("expected a call depth diagnostic, got:\n%s", out)
	}
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
