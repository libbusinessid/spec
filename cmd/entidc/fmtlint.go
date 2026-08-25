package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/conformance"
	"github.com/entid-org/spec/internal/diagnostics"
	"github.com/entid-org/spec/internal/hcllang"
	"github.com/entid-org/spec/internal/reference"
)

func runFmt(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("fmt", flag.ContinueOnError)
	fs.SetOutput(stderr)
	check := fs.Bool("check", false, "report the files that would change instead of rewriting them")
	asJSON := fs.Bool("json", false, "render the diagnostics as JSON on stdout")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	roots := fs.Args()
	if len(roots) == 0 {
		roots = []string{"rules", "conformance"}
	}
	bag := diagnostics.New()
	changed := 0
	for _, root := range roots {
		hclFiles, err := hcllang.Discover(root)
		if err != nil {
			bag.Errorf(diagnostics.Position{File: root}, "FMT001", "cannot walk the directory: %v", err)
			continue
		}
		for _, f := range hclFiles {
			source, err := os.ReadFile(f.AbsPath)
			if err != nil {
				bag.Errorf(diagnostics.Position{File: f.RelPath}, "FMT002", "cannot read the file: %v", err)
				continue
			}
			formatted := hclwrite.Format(source)
			if writeFormatted(bag, stdout, f.RelPath, f.AbsPath, source, formatted, *check) {
				changed++
			}
		}
		jsonlFiles, err := hcllang.DiscoverExt(root, ".jsonl")
		if err != nil {
			continue
		}
		for _, f := range jsonlFiles {
			source, err := os.ReadFile(f.AbsPath)
			if err != nil {
				bag.Errorf(diagnostics.Position{File: f.RelPath}, "FMT002", "cannot read the file: %v", err)
				continue
			}
			cases, readBag := conformance.Read(f.RelPath, source)
			bag.Extend(readBag)
			if readBag.HasErrors() {
				continue
			}
			formatted, err := conformance.WriteCanonicalJSONL(cases)
			if err != nil {
				bag.Errorf(diagnostics.Position{File: f.RelPath}, "FMT003", "cannot render the corpus: %v", err)
				continue
			}
			if writeFormatted(bag, stdout, f.RelPath, f.AbsPath, source, formatted, *check) {
				changed++
			}
		}
	}
	if *check && changed > 0 {
		bag.Errorf(diagnostics.Position{}, "FMT004", "%d file(s) are not in their canonical form", changed)
	}
	return report(bag, *asJSON, stdout, stderr)
}

func writeFormatted(bag *diagnostics.Bag, stdout io.Writer, rel, abs string, source, formatted []byte, check bool) bool {
	if string(source) == string(formatted) {
		return false
	}
	if check {
		_, _ = fmt.Fprintf(stdout, "%s\n", rel)
		return true
	}
	if err := artifact.WriteFileAtomic(abs, formatted); err != nil {
		bag.Errorf(diagnostics.Position{File: rel}, "FMT005", "cannot rewrite the file: %v", err)
	}
	return true
}

func runLint(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("lint", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var opts buildOptions
	opts.bind(fs)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	result, bag := build(opts)
	if result == nil {
		return report(bag, opts.json, stdout, stderr)
	}
	lintIdempotence(bag, result)
	lintFormatting(bag, opts)
	lintProvenance(bag, result)
	lintStructuralSeparators(bag, result)
	lintAtlas(bag, result)
	return report(bag, opts.json, stdout, stderr)
}

// idempotenceProbes are the generated values used to check that every
// canonicalizer is idempotent beyond the reviewed corpus.
var idempotenceProbes = []string{
	"", " ", "  \t\n", "0", "00", "0000000000", "000000000000",
	"AB", "ab", "A B", "a.b-c/d", "\u00a0\u2007X\ufeff",
	"BE0123456749", "be 0123.456.749", "FR09012345674", "EL012345670", "GR012345670",
	"FR7501.012345674", "00000000000000000098", "012345674", "12345678",
	"XX", "0-0-0-0", "É", "İ", "ß", "1234567890123456789012345",
}

// lintIdempotence verifies that canonicalizing a canonical value is a no-op,
// over the reviewed corpus and over generated probes.
func lintIdempotence(bag *diagnostics.Bag, result *buildResult) {
	engine := reference.NewEngineFromRuleset(result.rules.Ruleset)
	seen := map[string]bool{}
	check := func(kind, value string, country *string) {
		key := kind + "\x00" + value + "\x00"
		if country != nil {
			key += *country
		}
		if seen[key] {
			return
		}
		seen[key] = true
		first, err := engine.Canonicalize(reference.Input{Kind: kind, Value: value, CountryCode: country},
			reference.Options{})
		if err != nil {
			bag.Errorf(diagnostics.Position{}, "LINT001", "canonicalization failed for %q/%q: %v", kind, value, err)
			return
		}
		second, err := engine.Canonicalize(reference.Input{
			Kind: kind, Value: first.CanonicalValue, CountryCode: first.CountryCode,
		}, reference.Options{})
		if err != nil {
			bag.Errorf(diagnostics.Position{}, "LINT001", "canonicalization failed for %q/%q: %v",
				kind, first.CanonicalValue, err)
			return
		}
		if first.CanonicalValue != second.CanonicalValue {
			bag.Errorf(diagnostics.Position{}, "LINT002",
				"canonicalization of kind %q is not idempotent: %q then %q",
				kind, first.CanonicalValue, second.CanonicalValue)
		}
	}
	for _, c := range result.suite.Cases {
		if c.Kind == nil || c.Input == nil {
			continue
		}
		check(*c.Kind, *c.Input, c.CountryCode)
	}
	kinds := make([]string, 0, len(result.rules.Bundle.GetDispatchers()))
	for _, d := range result.rules.Bundle.GetDispatchers() {
		kinds = append(kinds, d.GetKind())
	}
	countries := []*string{nil}
	for _, d := range result.rules.Bundle.GetIdentifiers() {
		if d.CountryCode != nil {
			country := d.GetCountryCode()
			countries = append(countries, &country)
		}
	}
	for _, kind := range kinds {
		for _, probe := range idempotenceProbes {
			for _, country := range countries {
				check(kind, probe, country)
			}
		}
	}
}

// lintFormatting reports the sources that are not in their canonical form.
func lintFormatting(bag *diagnostics.Bag, opts buildOptions) {
	for _, root := range []string{opts.rules, opts.cases} {
		files, err := hcllang.Discover(root)
		if err != nil {
			continue
		}
		for _, f := range files {
			source, err := os.ReadFile(f.AbsPath)
			if err != nil {
				continue
			}
			if string(hclwrite.Format(source)) != string(source) {
				bag.Suggestf(diagnostics.Position{File: f.RelPath}, "LINT003",
					"run `entidc fmt`", "the file is not in its canonical form")
			}
		}
	}
}

// lintProvenance reports every definition able to reject an input without a
// source, and every conformance case referencing an unknown source id.
func lintProvenance(bag *diagnostics.Bag, result *buildResult) {
	known := map[string]bool{}
	for _, d := range result.rules.Bundle.GetIdentifiers() {
		for _, s := range d.GetSources() {
			known[s.GetId()] = true
		}
	}
	for _, c := range result.suite.Cases {
		for _, id := range c.SourceIDs {
			if !known[id] {
				bag.Errorf(diagnostics.Position{File: c.File, Line: c.Line}, "LINT004",
					"the case references the unknown source %q", id)
			}
		}
	}
	for _, d := range result.rules.Bundle.GetIdentifiers() {
		if len(d.GetSources()) > 0 {
			continue
		}
		country := globalCountry
		if d.CountryCode != nil {
			country = d.GetCountryCode()
		}
		bag.Errorf(diagnostics.Position{}, "LINT005",
			"the definition %s/%s carries no source", d.GetKind(), country)
	}
}

// atlasPath is the document that states, for every country that matters to the
// roadmap, which identifier it issues and whether this library covers it.
const atlasPath = "docs/identifier-atlas.md"

// lintAtlas refuses a definition the atlas does not mention.
//
// The atlas answers "what does this library actually support", and it is the
// page a reader lands on before any other. It is written by hand because most
// of it is about the world rather than about this repository, and a hand
// written page that tracks generated artifacts drifts - the engine PROVENANCE
// headers proved that, sitting three versions behind their own lock. So the
// half that can be checked is checked: a rule that ships without an entry is
// a rule nobody can find.
//
// The check is deliberately shallow, and it is worth being precise about how
// shallow: it asks whether the document names the identifier somewhere and the
// jurisdiction somewhere, not whether it names them in the same row, and not
// whether what it says about them is true. It catches the omission of a whole
// definition, which is the failure that actually happens. Judging prose is not
// something a linter should pretend to do.
func lintAtlas(bag *diagnostics.Bag, result *buildResult) {
	raw, err := os.ReadFile(atlasPath)
	if err != nil {
		// Running from somewhere other than the repository root is not itself
		// an error; the CI always runs from the root, which is what matters.
		return
	}
	checkAtlas(bag, result, string(raw))
}

// checkAtlas holds the decision, separated from reading the file so it can be
// tested against a document rather than against the repository.
func checkAtlas(bag *diagnostics.Bag, result *buildResult, text string) {
	// The atlas writes identifier names the way a reader says them: VAT rather
	// than vat, "Company number" rather than company_number, D-U-N-S rather
	// than duns. Folding case and dropping the separators makes the three
	// spellings meet without asking the document to be written in kind names.
	folded := foldName(text)
	for _, d := range result.rules.Bundle.GetIdentifiers() {
		country := globalCountry
		if d.CountryCode != nil {
			country = d.GetCountryCode()
		}
		// A country is covered either by its own row or by a family row that
		// names it, so the kind alone is not enough to conclude.
		if strings.Contains(folded, foldName(d.GetKind())) && mentionsCountry(text, country) {
			continue
		}
		bag.Errorf(diagnostics.Position{File: atlasPath}, "LINT006",
			"the definition %s/%s is missing from %s", d.GetKind(), country, atlasPath)
	}
}

// foldName lower cases and removes the separators that distinguish an
// identifier name written for a reader from the same name written as a kind.
func foldName(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToLower(s) {
		if r == ' ' || r == '-' || r == '_' || r == '.' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// mentionsCountry reports whether the atlas names the jurisdiction, by code or
// by the English name the tables use.
func mentionsCountry(text, code string) bool {
	if code == globalCountry {
		return true
	}
	if strings.Contains(text, "`"+code+"`") {
		return true
	}
	if name, ok := atlasCountryNames[code]; ok {
		return strings.Contains(text, name)
	}
	return false
}

// atlasCountryNames maps every jurisdiction carrying a definition to the name
// the atlas tables use for it. A definition for a jurisdiction absent here
// fails the lint, which is the point: adding a country means placing it on the
// map.
var atlasCountryNames = map[string]string{
	"AT": "Austria", "CN": "China", "JP": "Japan", "BR": "Brazil", "BE": "Belgium", "BG": "Bulgaria", "CY": "Cyprus",
	"CZ": "Czechia", "DE": "Germany", "DK": "Denmark", "EE": "Estonia",
	"EL": "Greece", "ES": "Spain", "FI": "Finland", "FR": "France",
	"GB": "United Kingdom", "GR": "Greece", "HR": "Croatia", "HU": "Hungary",
	"IE": "Ireland", "IS": "Iceland", "IT": "Italy", "LI": "Liechtenstein",
	"LT": "Lithuania", "LU": "Luxembourg", "LV": "Latvia", "MT": "Malta",
	"NL": "Netherlands", "NO": "Norway", "PL": "Poland", "PT": "Portugal",
	"RO": "Romania", "SE": "Sweden", "SI": "Slovenia", "SK": "Slovakia",
	"US": "United States", "XI": "Northern Ireland",
}

// lintStructuralSeparators refuses a canonicalizer that removes a character its
// own format then requires.
//
// The EUID rules found this the hard way: three canonicalizers were copied from
// SIRET, which strips dots, while the dot is what separates the register from
// the number in an EUID. Every value lost its boundary and failed on the
// separator assertion. The conformance cases caught it, but only because those
// countries had cases; the next one added without them would not be caught at
// all. This makes the contradiction a compile error instead.
func lintStructuralSeparators(bag *diagnostics.Bag, result *buildResult) {
	bundle := result.rules.Bundle
	programs := map[uint32]*irv1.Program{}
	for _, p := range bundle.GetPrograms() {
		programs[p.GetId()] = p
	}

	collect := func(id uint32, kind func(*irv1.Node) (string, bool)) map[rune]bool {
		out := map[rune]bool{}
		p := programs[id]
		if p == nil {
			return out
		}
		for _, n := range p.GetNodes() {
			if text, ok := kind(n); ok {
				for _, r := range text {
					out[r] = true
				}
			}
		}
		return out
	}

	removed := func(n *irv1.Node) (string, bool) {
		op := n.GetCanonicalizationOperation()
		if op.GetKind() != irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_CHARS {
			return "", false
		}
		return op.GetText(), true
	}
	required := func(n *irv1.Node) (string, bool) {
		op := n.GetPredicateOperation()
		if op.GetKind() != irv1.PredicateOpKind_PREDICATE_OP_KIND_CONTAINS {
			return "", false
		}
		return op.GetText(), true
	}

	for _, d := range bundle.GetIdentifiers() {
		stripped := collect(d.GetCanonicalizationProgram(), removed)
		if len(stripped) == 0 {
			continue
		}
		country := globalCountry
		if d.CountryCode != nil {
			country = d.GetCountryCode()
		}
		for r := range collect(d.GetFormatProgram(), required) {
			if stripped[r] {
				bag.Errorf(diagnostics.Position{}, "LINT005",
					"%s/%s: the canonicalizer removes %q while its format requires the value to contain it",
					d.GetKind(), country, string(r))
			}
		}
	}
}
