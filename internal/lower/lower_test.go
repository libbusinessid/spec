package lower_test

import (
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/ast"
	"github.com/entid-org/spec/internal/features"
	"github.com/entid-org/spec/internal/hcllang"
	"github.com/entid-org/spec/internal/linker"
	"github.com/entid-org/spec/internal/lower"
	"github.com/entid-org/spec/internal/typecheck"
)

const unit = `
canonicalizer "dispatch" "identifier" {
  steps = [trim_whitespace(), uppercase_ascii()]
}

canonicalizer "z" "later" {
  steps = [trim_whitespace()]
}

format "a" "b" {
  checks = [
    require(length_eq(subject(), 9), "invalid_length", "a.b.length"),
    require(length_eq(subject(), 9), "invalid_length", "a.b.length"),
  ]
}

checksum "a" "b" {
  rule = luhn(subject())
}

dispatcher "beta" {
  aliases           = ["zeta", "alpha"]
  pre_canonicalizer = canonicalizer.dispatch.identifier

  country_aliases = {
    "ZZ" = "BE"
    "AA" = "FR"
  }

  target {
    country_code      = "FR"
    accepted_prefixes = ["ZZ", "AA"]
    canonical_prefix  = "AA"
    identifier        = identifier.beta.FR
  }

  target {
    country_code = "BE"
    identifier   = identifier.beta.BE
  }
}

dispatcher "alphakind" {
  pre_canonicalizer = canonicalizer.z.later

  target {
    identifier = identifier.alphakind.GLOBAL
  }
}

identifier "beta" "FR" {
  canonicalizer   = canonicalizer.dispatch.identifier
  format          = format.a.b
  checksum        = checksum.a.b
  default_profile = "compatible"

  source {
    id               = "z-source"
    url              = "https://example.invalid/z"
    authority        = "a"
    title            = "t"
    accessed_at      = "2026-08-18"
    jurisdiction     = "FR"
    language         = "en"
    notes            = "n"
    license_or_terms = "l"
    tier = "primary"
  }

  source {
    id               = "a-source"
    url              = "https://example.invalid/a"
    authority        = "a"
    title            = "t"
    accessed_at      = "2026-08-18"
    jurisdiction     = "FR"
    language         = "en"
    notes            = "n"
    license_or_terms = "l"
    tier = "primary"
    archive_url      = "https://archive.invalid/a"
  }
}

identifier "beta" "BE" {
  canonicalizer   = canonicalizer.dispatch.identifier
  format          = format.a.b
  default_profile = "strict_current"

  no_checksum {
    reason_code = "checksum_not_published"
    notes       = "no published algorithm"
  }

  source {
    id               = "be-source"
    url              = "https://example.invalid/be"
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

identifier "alphakind" "GLOBAL" {
  canonicalizer   = canonicalizer.z.later
  format          = format.a.b
  checksum        = checksum.a.b
  default_profile = "compatible"

  source {
    id               = "global-source"
    url              = "https://example.invalid/global"
    authority        = "a"
    title            = "t"
    accessed_at      = "2026-08-18"
    jurisdiction     = "GLOBAL"
    language         = "en"
    notes            = "n"
    license_or_terms = "l"
    tier = "primary"
  }
}
`

func build(t *testing.T, optimize bool) *irv1.RuleBundle {
	t.Helper()
	file, bag := hcllang.ParseFile("rules.hcl", []byte(unit))
	if bag.HasErrors() {
		t.Fatalf("parse errors: %v", bag.Sorted())
	}
	table, linkBag := linker.Link(&ast.Unit{Files: []*ast.File{file}})
	if linkBag.HasErrors() {
		t.Fatalf("link errors: %v", linkBag.Sorted())
	}
	checked, typeBag := typecheck.Check(table)
	if typeBag.HasErrors() {
		t.Fatalf("type errors: %v", typeBag.Sorted())
	}
	bundle, lowerBag := lower.Lower(table, checked, lower.Options{RulesVersion: "2026.08.0", Optimize: optimize})
	if lowerBag.HasErrors() {
		t.Fatalf("lowering errors: %v", lowerBag.Sorted())
	}
	return bundle
}

func TestLoweringIsFullyOrdered(t *testing.T) {
	bundle := build(t, false)

	if bundle.GetFormatVersion() != lower.FormatVersion || bundle.GetRulesVersion() != "2026.08.0" {
		t.Fatalf("unexpected header %+v", bundle)
	}
	for i := 1; i < len(bundle.GetPrograms()); i++ {
		if bundle.GetPrograms()[i-1].GetId() >= bundle.GetPrograms()[i].GetId() {
			t.Fatal("programs must be sorted by id")
		}
	}
	for i := 1; i < len(bundle.GetRequiredFeatureIds()); i++ {
		if bundle.GetRequiredFeatureIds()[i-1] >= bundle.GetRequiredFeatureIds()[i] {
			t.Fatal("capabilities must be strictly ascending")
		}
	}
	if bundle.GetDispatchers()[0].GetKind() != "alphakind" {
		t.Fatalf("dispatchers must be sorted by kind: %s", bundle.GetDispatchers()[0].GetKind())
	}
	beta := bundle.GetDispatchers()[1]
	if beta.GetKindAliases()[0] != "alpha" || beta.GetKindAliases()[1] != "zeta" {
		t.Fatalf("kind aliases must be sorted: %v", beta.GetKindAliases())
	}
	if beta.GetCountryAliases()[0].GetAlias() != "AA" {
		t.Fatalf("country aliases must be sorted: %v", beta.GetCountryAliases())
	}
	if beta.GetTargets()[0].GetCountryCode() != "BE" {
		t.Fatalf("targets must be sorted by country: %v", beta.GetTargets())
	}
	fr := beta.GetTargets()[1]
	if fr.GetAcceptedPrefixes()[0] != "AA" {
		t.Fatalf("prefixes must be sorted: %v", fr.GetAcceptedPrefixes())
	}
	if bundle.GetIdentifiers()[0].GetKind() != "alphakind" || bundle.GetIdentifiers()[0].CountryCode != nil {
		t.Fatalf("identifiers must be sorted by kind then GLOBAL first: %v", bundle.GetIdentifiers()[0])
	}
	frDef := bundle.GetIdentifiers()[2]
	if frDef.GetSources()[0].GetId() != "a-source" {
		t.Fatalf("sources must be sorted by id: %v", frDef.GetSources())
	}
	if frDef.GetSources()[0].ArchiveUrl == nil {
		t.Fatal("the archive url must be lowered")
	}
	beDef := bundle.GetIdentifiers()[1]
	if beDef.ChecksumProgram != nil ||
		beDef.GetAbsentChecksumReason() != irv1.ReasonCode_REASON_CODE_CHECKSUM_NOT_PUBLISHED {
		t.Fatalf("the explicit absence must be lowered: %v", beDef)
	}
	if beDef.GetDefaultProfile() != "strict_current" {
		t.Fatalf("unexpected profile %q", beDef.GetDefaultProfile())
	}
}

func TestLoweringIsDeterministic(t *testing.T) {
	first := build(t, true)
	second := build(t, true)
	if first.String() != second.String() {
		t.Fatal("two lowerings of the same unit must be identical")
	}
}

func TestDeduplicationSharesIdenticalNodes(t *testing.T) {
	plain := build(t, false)
	optimized := build(t, true)
	count := func(b *irv1.RuleBundle, kind irv1.ProgramKind) int {
		total := 0
		for _, p := range b.GetPrograms() {
			if p.GetKind() == kind {
				total += len(p.GetNodes())
			}
		}
		return total
	}
	if count(optimized, irv1.ProgramKind_PROGRAM_KIND_FORMAT) >=
		count(plain, irv1.ProgramKind_PROGRAM_KIND_FORMAT) {
		t.Fatal("the duplicated assertion must be deduplicated")
	}
}

func TestRequiredCapabilitiesMatchTheOperations(t *testing.T) {
	bundle := build(t, false)
	declared := map[uint32]bool{}
	for _, id := range bundle.GetRequiredFeatureIds() {
		declared[id] = true
	}
	for _, expected := range []uint32{
		features.CoreGraphV1, features.AsciiAndWhitespaceV1, features.CanonicalizationBasicV1,
		features.IdentifierDispatchV1, features.FormatAssertionsV1, features.ProfilesV1,
		features.ChecksumTristateV1, features.ChecksumLuhnV1, features.ProvenanceV1,
	} {
		if !declared[expected] {
			t.Fatalf("capability %d must be declared", expected)
		}
	}
	if declared[features.ChecksumMod97V1] {
		t.Fatal("an unused capability must not be declared")
	}
}
