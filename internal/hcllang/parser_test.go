package hcllang_test

import (
	"strings"
	"testing"

	"github.com/libbusinessid/spec/internal/ast"
	"github.com/libbusinessid/spec/internal/hcllang"
)

func parse(t *testing.T, src string) (*ast.File, []string) {
	t.Helper()
	file, bag := hcllang.ParseFile("test.hcl", []byte(src))
	codes := make([]string, 0, bag.Len())
	for _, d := range bag.Sorted() {
		codes = append(codes, d.Code+": "+d.Message)
	}
	return file, codes
}

func mustParse(t *testing.T, src string) *ast.File {
	t.Helper()
	file, codes := parse(t, src)
	if len(codes) != 0 {
		t.Fatalf("unexpected diagnostics: %v", codes)
	}
	if file == nil {
		t.Fatal("nil file")
	}
	return file
}

func TestParseCanonicalizer(t *testing.T) {
	f := mustParse(t, `
canonicalizer "vat" "common" {
  steps = [
    trim_whitespace(),
    uppercase_ascii(),
    remove_chars([".", "-"]),
    replace_prefix("GR", "EL"),
    when(length_eq(value(), 11), insert(2, "0")),
  ]
}
`)
	if len(f.Canonicalizers) != 1 {
		t.Fatalf("got %d canonicalizers", len(f.Canonicalizers))
	}
	c := f.Canonicalizers[0]
	if c.Symbol() != "canonicalizer.vat.common" {
		t.Fatalf("symbol %q", c.Symbol())
	}
	if len(c.Steps) != 5 {
		t.Fatalf("got %d steps", len(c.Steps))
	}
	call, ok := c.Steps[2].(*ast.CallExpr)
	if !ok || call.Name != "remove_chars" {
		t.Fatalf("unexpected step: %#v", c.Steps[2])
	}
	list, ok := call.Args[0].(*ast.ListExpr)
	if !ok || len(list.Items) != 2 {
		t.Fatalf("unexpected argument: %#v", call.Args[0])
	}
	if c.Position.Line != 2 || c.Position.File != "test.hcl" {
		t.Fatalf("unexpected position %+v", c.Position)
	}
}

func TestParseFormatWithCapturesAndUse(t *testing.T) {
	f := mustParse(t, `
format "euid" "FR" {
  subject = value()

  capture "registration" {
    value = after_first(value(), ".")
  }

  checks = [
    require(starts_with(value(), "FR"), "invalid_format", "euid.fr.prefix"),
  ]

  use_format {
    rule  = format.fr.siren
    input = capture.registration
  }
}
`)
	d := f.Formats[0]
	if d.Symbol() != "format.euid.FR" {
		t.Fatalf("symbol %q", d.Symbol())
	}
	if d.Subject == nil || len(d.Captures) != 1 || len(d.Checks) != 1 || len(d.Uses) != 1 {
		t.Fatalf("unexpected format: %#v", d)
	}
	if d.Captures[0].Name != "registration" {
		t.Fatalf("capture name %q", d.Captures[0].Name)
	}
	if d.Uses[0].Rule.String() != "format.fr.siren" {
		t.Fatalf("rule %q", d.Uses[0].Rule.String())
	}
	if ref, ok := d.Uses[0].Input.(*ast.RefExpr); !ok || ref.String() != "capture.registration" {
		t.Fatalf("input %#v", d.Uses[0].Input)
	}
}

func TestParseChecksumAndDispatcherAndIdentifier(t *testing.T) {
	f := mustParse(t, `
checksum "fr" "siren" {
  rule = luhn(subject())
}

dispatcher "vat" {
  aliases           = ["vat_id"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  country_aliases = {
    "UK" = "GB"
    EL   = "GR"
  }

  target {
    country_code                     = "GR"
    accepted_prefixes                = ["EL", "GR"]
    canonical_prefix                 = "EL"
    identifier                       = identifier.vat.GR
    allow_unprefixed_without_country = false
  }

  target {
    identifier = identifier.vat.GLOBAL
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
    title            = "VAT identification"
    accessed_at      = "2026-08-18"
    jurisdiction     = "BE"
    language         = "en"
    notes            = "n/a"
    license_or_terms = "public"
    archive_url      = "https://archive.invalid/be"
  }
}

identifier "vat" "DE" {
  canonicalizer   = canonicalizer.vat.common
  format          = format.vat.de
  default_profile = "compatible"

  no_checksum {
    reason_code = "unsupported_checksum"
    notes       = "not expressible in IR v1"
  }
}
`)
	if len(f.Checksums) != 1 || f.Checksums[0].Symbol() != "checksum.fr.siren" {
		t.Fatalf("unexpected checksums: %#v", f.Checksums)
	}
	if f.Checksums[0].Subject != nil {
		t.Fatal("subject must be absent by default")
	}
	d := f.Dispatchers[0]
	if d.Symbol() != "dispatcher.vat" || len(d.Aliases) != 1 || len(d.Targets) != 2 {
		t.Fatalf("unexpected dispatcher: %#v", d)
	}
	if len(d.CountryAliases) != 2 {
		t.Fatalf("unexpected country aliases: %#v", d.CountryAliases)
	}
	if d.CountryAliases[0].Alias != "UK" || d.CountryAliases[1].Alias != "EL" {
		t.Fatalf("alias order or parsing is wrong: %#v", d.CountryAliases)
	}
	if d.Targets[0].Global || d.Targets[0].CountryCode != "GR" || d.Targets[0].CanonicalPrefix != "EL" {
		t.Fatalf("unexpected target: %#v", d.Targets[0])
	}
	if !d.Targets[1].Global {
		t.Fatalf("second target must be global: %#v", d.Targets[1])
	}
	be := f.Identifiers[0]
	if be.Symbol() != "identifier.vat.BE" || be.Checksum == nil || len(be.Sources) != 1 {
		t.Fatalf("unexpected identifier: %#v", be)
	}
	if !be.Sources[0].HasArchiveURL || be.Sources[0].ArchiveURL == "" {
		t.Fatal("archive url must be parsed")
	}
	de := f.Identifiers[1]
	if de.Checksum != nil || de.NoChecksum == nil || de.NoChecksum.ReasonCode != "unsupported_checksum" {
		t.Fatalf("unexpected identifier: %#v", de)
	}
}

func TestParseGlobalIdentifierLabel(t *testing.T) {
	f := mustParse(t, `
identifier "lei" "GLOBAL" {
  canonicalizer   = canonicalizer.lei.common
  format          = format.lei.generic
  checksum        = checksum.lei.generic
  default_profile = "compatible"
}
`)
	id := f.Identifiers[0]
	if !id.Global || id.CountryCode != "" || id.Symbol() != "identifier.lei.GLOBAL" {
		t.Fatalf("unexpected identifier: %#v", id)
	}
}

func TestParserRejections(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
	}{
		{"syntax", "canonicalizer \"a\" \"b\" {", hcllang.CodeSyntax},
		{"unknown block", "widget \"a\" {}", hcllang.CodeUnknownBlock},
		{"top level attribute", "x = 1", hcllang.CodeUnknownAttribute},
		{"bad labels", "canonicalizer \"a\" {\n steps = []\n}", hcllang.CodeBadLabels},
		{"missing attribute", "canonicalizer \"a\" \"b\" {}", hcllang.CodeMissingAttribute},
		{"unknown attribute", "canonicalizer \"a\" \"b\" {\n steps = []\n oops = 1\n}", hcllang.CodeUnknownAttribute},
		{"nested unknown block", "canonicalizer \"a\" \"b\" {\n steps = []\n oops {}\n}", hcllang.CodeUnknownBlock},
		{"interpolation", "checksum \"a\" \"b\" {\n rule = \"FR.${rule.fr.siren.format}\"\n}", hcllang.CodeBadExpression},
		{"arithmetic", "checksum \"a\" \"b\" {\n rule = 1 + 2\n}", hcllang.CodeBadExpression},
		{"index traversal", "checksum \"a\" \"b\" {\n rule = luhn(a[0])\n}", hcllang.CodeBadExpression},
		{"not a reference", "identifier \"vat\" \"BE\" {\n canonicalizer = \"x\"\n format = format.a.b\n}", hcllang.CodeBadValue},
		{"not a list", "canonicalizer \"a\" \"b\" {\n steps = 1\n}", hcllang.CodeBadValue},
		{"aliases not strings", "dispatcher \"vat\" {\n aliases = [1]\n pre_canonicalizer = canonicalizer.a.b\n}", hcllang.CodeBadValue},
		{"aliases not a list", "dispatcher \"vat\" {\n aliases = \"x\"\n pre_canonicalizer = canonicalizer.a.b\n}", hcllang.CodeBadValue},
		{"country aliases not object", "dispatcher \"vat\" {\n pre_canonicalizer = canonicalizer.a.b\n country_aliases = [\"UK\"]\n}", hcllang.CodeBadValue},
		{"country alias value", "dispatcher \"vat\" {\n pre_canonicalizer = canonicalizer.a.b\n country_aliases = {\"UK\" = 1}\n}", hcllang.CodeBadValue},
		{"bool attribute", "dispatcher \"vat\" {\n pre_canonicalizer = canonicalizer.a.b\n target {\n identifier = identifier.a.b\n allow_unprefixed_without_country = \"yes\"\n }\n}", hcllang.CodeBadValue},
		{"float literal", "canonicalizer \"a\" \"b\" {\n steps = [insert(1.5, \"x\")]\n}", hcllang.CodeBadValue},
		{"null literal", "canonicalizer \"a\" \"b\" {\n steps = [insert(null, \"x\")]\n}", hcllang.CodeBadValue},
		{"duplicate no_checksum", "identifier \"vat\" \"BE\" {\n canonicalizer = canonicalizer.a.b\n format = format.a.b\n no_checksum {\n reason_code = \"unsupported_checksum\"\n notes = \"a\"\n }\n no_checksum {\n reason_code = \"unsupported_checksum\"\n notes = \"b\"\n }\n}", hcllang.CodeDuplicateBlock},
		{"labelled target", "dispatcher \"vat\" {\n pre_canonicalizer = canonicalizer.a.b\n target \"x\" {\n identifier = identifier.a.b\n }\n}", hcllang.CodeBadLabels},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, codes := parse(t, tc.src)
			joined := strings.Join(codes, "\n")
			if !strings.Contains(joined, tc.code) {
				t.Fatalf("expected %s, got:\n%s", tc.code, joined)
			}
		})
	}
}

func TestNegativeIntegerLiteral(t *testing.T) {
	f := mustParse(t, `
checksum "a" "b" {
  rule = compare_digit(remainder_map(modulo(digits_to_integer(subject()), 11), [-1]), subject(), 0)
}
`)
	call := f.Checksums[0].Rule.(*ast.CallExpr)
	inner := call.Args[0].(*ast.CallExpr)
	list := inner.Args[1].(*ast.ListExpr)
	if list.Items[0].(*ast.IntLit).Value != -1 {
		t.Fatal("negative literal must be parsed")
	}
}

func TestInvalidUTF8AndBOM(t *testing.T) {
	if _, bag := hcllang.ParseFile("t.hcl", []byte{0xff, 0xfe}); !bag.HasErrors() {
		t.Fatal("invalid UTF-8 must be refused")
	}
	if _, bag := hcllang.ParseFile("t.hcl", []byte("\xef\xbb\xbfcanonicalizer \"a\" \"b\" {}")); !bag.HasErrors() {
		t.Fatal("BOM must be refused")
	}
}

func TestParseUnitCollectsReadErrors(t *testing.T) {
	files := []hcllang.SourceFile{{AbsPath: "/x", RelPath: "a.hcl"}, {AbsPath: "/y", RelPath: "b.hcl"}}
	unit, bag := hcllang.ParseUnit(files, func(sf hcllang.SourceFile) ([]byte, error) {
		if sf.RelPath == "a.hcl" {
			return nil, errRead
		}
		return []byte("canonicalizer \"a\" \"b\" {\n steps = []\n}"), nil
	})
	if len(unit.Files) != 1 || unit.Files[0].Path != "b.hcl" {
		t.Fatalf("unexpected unit: %#v", unit.Files)
	}
	if !bag.HasErrors() {
		t.Fatal("read error must be reported")
	}
	if len(unit.Canonicalizers()) != 1 || len(unit.Formats()) != 0 || len(unit.Checksums()) != 0 ||
		len(unit.Dispatchers()) != 0 || len(unit.Identifiers()) != 0 {
		t.Fatal("unit accessors are wrong")
	}
}

var errRead = errReader{}

type errReader struct{}

func (errReader) Error() string { return "read error" }

func TestParseAcceptsParenthesesAndTemplateWrap(t *testing.T) {
	f := mustParse(t, `
checksum "a" "b" {
  rule = (luhn(subject()))
}

format "a" "b" {
  checks = [require(starts_with(value(), "${"FR"}"), "invalid_format", "k")]
}
`)
	if f.Checksums[0].Rule.(*ast.CallExpr).Name != "luhn" {
		t.Fatal("parentheses must be transparent")
	}
	call := f.Formats[0].Checks[0].(*ast.CallExpr).Args[0].(*ast.CallExpr)
	if call.Args[1].(*ast.StringLit).Value != "FR" {
		t.Fatal("a wrapped template of a single literal must be accepted")
	}
}

func TestParseAcceptsBareIdentifierCountryAliasKeys(t *testing.T) {
	f := mustParse(t, `
dispatcher "vat" {
  pre_canonicalizer = canonicalizer.a.b

  country_aliases = {
    UK = "GB"
  }

  target {
    identifier = identifier.vat.GLOBAL
  }
}
`)
	if f.Dispatchers[0].CountryAliases[0].Alias != "UK" {
		t.Fatalf("unexpected alias %#v", f.Dispatchers[0].CountryAliases[0])
	}
}

func TestParserAdditionalRejections(t *testing.T) {
	tests := []struct {
		name string
		src  string
		code string
	}{
		{"argument expansion", "checksum \"a\" \"b\" {\n rule = luhn(subject()...)\n}", hcllang.CodeBadExpression},
		{"dotted country alias key", "dispatcher \"vat\" {\n pre_canonicalizer = canonicalizer.a.b\n country_aliases = {\n a.b = \"GB\"\n }\n}", hcllang.CodeBadValue},
		{"labelled use_format", "format \"a\" \"b\" {\n checks = []\n use_format \"x\" {\n rule = format.a.b\n input = subject()\n }\n}", hcllang.CodeBadLabels},
		{"labelled source", "identifier \"vat\" \"BE\" {\n canonicalizer = canonicalizer.a.b\n format = format.a.b\n source \"x\" {\n id = \"a\"\n }\n}", hcllang.CodeBadLabels},
		{"labelled no_checksum", "identifier \"vat\" \"BE\" {\n canonicalizer = canonicalizer.a.b\n format = format.a.b\n no_checksum \"x\" {\n reason_code = \"unsupported_checksum\"\n }\n}", hcllang.CodeBadLabels},
		{"labelled capture", "format \"a\" \"b\" {\n checks = []\n capture {\n value = value()\n }\n}", hcllang.CodeBadLabels},
		{"capture without value", "format \"a\" \"b\" {\n checks = []\n capture \"x\" {\n }\n}", hcllang.CodeMissingAttribute},
		{"use_format without rule", "format \"a\" \"b\" {\n checks = []\n use_format {\n input = subject()\n }\n}", hcllang.CodeMissingAttribute},
		{"use_format with a bad rule", "format \"a\" \"b\" {\n checks = []\n use_format {\n rule = \"x\"\n input = subject()\n }\n}", hcllang.CodeBadValue},
		{"target without identifier", "dispatcher \"vat\" {\n pre_canonicalizer = canonicalizer.a.b\n target {\n country_code = \"BE\"\n }\n}", hcllang.CodeMissingAttribute},
		{"source with a missing field", "identifier \"vat\" \"BE\" {\n canonicalizer = canonicalizer.a.b\n format = format.a.b\n source {\n id = \"a\"\n }\n}", hcllang.CodeMissingAttribute},
		{"checksum without rule", "checksum \"a\" \"b\" {\n subject = value()\n}", hcllang.CodeMissingAttribute},
		{"dispatcher without pre_canonicalizer", "dispatcher \"vat\" {\n target {\n identifier = identifier.vat.GLOBAL\n }\n}", hcllang.CodeMissingAttribute},
		{"identifier without canonicalizer", "identifier \"vat\" \"BE\" {\n format = format.a.b\n}", hcllang.CodeMissingAttribute},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, codes := parse(t, tc.src)
			joined := strings.Join(codes, "\n")
			if !strings.Contains(joined, tc.code) {
				t.Fatalf("expected %s, got:\n%s", tc.code, joined)
			}
		})
	}
}

func TestParseAcceptsAStringLiteralFromCty(t *testing.T) {
	// A heredoc produces a literal value rather than a template expression.
	f := mustParse(t, "format \"a\" \"b\" {\n  checks = [require(starts_with(value(), <<-EOT\nFR\nEOT\n), \"invalid_format\", \"k\")]\n}\n")
	call := f.Formats[0].Checks[0].(*ast.CallExpr).Args[0].(*ast.CallExpr)
	if lit, ok := call.Args[1].(*ast.StringLit); !ok || lit.Value == "" {
		t.Fatalf("unexpected argument %#v", call.Args[1])
	}
}

func TestParseRejectsUnsupportedLiteralTypes(t *testing.T) {
	_, codes := parse(t, "canonicalizer \"a\" \"b\" {\n  steps = [insert(1.5e400, \"x\")]\n}")
	if !strings.Contains(strings.Join(codes, "\n"), hcllang.CodeBadValue) {
		t.Fatalf("expected a value rejection, got %v", codes)
	}
}
