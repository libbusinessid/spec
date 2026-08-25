package artifact

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/features"
)

// CoverageInput carries everything the generated documentation needs.
type CoverageInput struct {
	Ruleset     *Ruleset
	Conformance *conformancev1.ConformanceBundle
}

// KindCoverageOf computes the per kind coverage table of the manifest.
func KindCoverageOf(in CoverageInput) []KindCoverage {
	byKind := map[string]*KindCoverage{}
	for _, d := range in.Ruleset.Bundle.GetIdentifiers() {
		entry, ok := byKind[d.GetKind()]
		if !ok {
			entry = &KindCoverage{Kind: d.GetKind()}
			byKind[d.GetKind()] = entry
		}
		country := globalCountry
		if d.CountryCode != nil {
			country = d.GetCountryCode()
		}
		entry.Countries = append(entry.Countries, country)
		if d.ChecksumProgram != nil {
			entry.WithChecksum++
		} else {
			entry.WithoutChecksum++
		}
	}
	if in.Conformance != nil {
		for _, c := range in.Conformance.GetCases() {
			if entry, ok := byKind[c.GetKind()]; ok {
				entry.ConformanceCases++
			}
		}
	}
	out := make([]KindCoverage, 0, len(byKind))
	for _, entry := range byKind {
		sort.Strings(entry.Countries)
		out = append(out, *entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out
}

// RenderCoverageDoc renders docs/generated/coverage.md.
func RenderCoverageDoc(in CoverageInput) []byte {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }
	bundle := in.Ruleset.Bundle

	w("%s", generatedBanner)
	w("")
	w("# Rule coverage")
	w("")
	w("Rules version `%s`, IR format version `%d`.", bundle.GetRulesVersion(), bundle.GetFormatVersion())
	w("")
	renderDefinitionMatrix(w, bundle)
	renderDispatchTables(w, bundle)
	renderAlgorithms(w, bundle)
	renderUnusedOperations(w, bundle)
	renderMissingChecksums(w, bundle)
	renderCapabilities(w, bundle)
	renderProvenance(w, bundle)
	if in.Conformance != nil {
		renderConformanceStatistics(w, in.Conformance)
	}
	return []byte(b.String())
}

type writeLine func(format string, args ...any)

func renderDefinitionMatrix(w writeLine, bundle *irv1.RuleBundle) {
	w("## Country by kind matrix")
	w("")
	w("| Kind | Country | Canonicalizer | Format | Checksum | Default profile | Sources |")
	w("|---|---|---:|---:|---:|---|---:|")
	for _, d := range bundle.GetIdentifiers() {
		country := globalCountry
		if d.CountryCode != nil {
			country = d.GetCountryCode()
		}
		checksum := "-"
		if d.ChecksumProgram != nil {
			checksum = fmt.Sprintf("%d", d.GetChecksumProgram())
		}
		w("| `%s` | `%s` | %d | %d | %s | `%s` | %d |",
			d.GetKind(), country, d.GetCanonicalizationProgram(), d.GetFormatProgram(),
			checksum, d.GetDefaultProfile(), len(d.GetSources()))
	}
	w("")
}

func renderDispatchTables(w writeLine, bundle *irv1.RuleBundle) {
	w("## Dispatch tables")
	w("")
	for _, disp := range bundle.GetDispatchers() {
		w("### `%s`", disp.GetKind())
		w("")
		aliases := "none"
		if len(disp.GetKindAliases()) > 0 {
			aliases = "`" + strings.Join(disp.GetKindAliases(), "`, `") + "`"
		}
		w("Kind aliases: %s.", aliases)
		w("")
		if len(disp.GetCountryAliases()) > 0 {
			parts := make([]string, 0, len(disp.GetCountryAliases()))
			for _, ca := range disp.GetCountryAliases() {
				parts = append(parts, fmt.Sprintf("`%s` to `%s`", ca.GetAlias(), ca.GetCountryCode()))
			}
			w("Country aliases: %s.", strings.Join(parts, ", "))
			w("")
		}
		w("| Country | Accepted prefixes | Canonical prefix | Implicit |")
		w("|---|---|---|---|")
		for _, t := range disp.GetTargets() {
			country := globalCountry
			if t.CountryCode != nil {
				country = t.GetCountryCode()
			}
			prefixes := "-"
			if len(t.GetAcceptedPrefixes()) > 0 {
				prefixes = "`" + strings.Join(t.GetAcceptedPrefixes(), "`, `") + "`"
			}
			canonical := "-"
			if t.CanonicalPrefix != nil {
				canonical = "`" + t.GetCanonicalPrefix() + "`"
			}
			implicit := "no"
			if t.GetAllowUnprefixedWithoutCountry() {
				implicit = "yes"
			}
			w("| `%s` | %s | %s | %s |", country, prefixes, canonical, implicit)
		}
		w("")
	}
}

func renderAlgorithms(w writeLine, bundle *irv1.RuleBundle) {
	w("## Algorithms in use")
	w("")
	w("| Operation | Programs |")
	w("|---|---:|")
	usage := map[string]int{}
	for _, p := range bundle.GetPrograms() {
		seen := map[string]bool{}
		for _, n := range p.GetNodes() {
			shape, err := shapeOf(n)
			if err != nil {
				continue
			}
			op, ok := features.LookupOp(shape.category, shape.code)
			if !ok || seen[op.Symbol] {
				continue
			}
			seen[op.Symbol] = true
			usage[op.Symbol]++
		}
	}
	symbols := make([]string, 0, len(usage))
	for symbol := range usage {
		symbols = append(symbols, symbol)
	}
	sort.Strings(symbols)
	for _, symbol := range symbols {
		w("| `%s` | %d |", symbol, usage[symbol])
	}
	w("")
}

// renderUnusedOperations lists what the IR defines and no rule emits.
//
// An engine compiles the bundle into native code, so it never generates an
// operation no rule carries - but it still has to implement one, in case a
// later bundle does, and nothing in the corpus proves that implementation. The
// count is the size of that untested surface, and printing it is what keeps it
// from growing unnoticed: prefix_in sat here fully implemented across the whole
// toolchain while a rule spelled out forty one starts_with by hand.
func renderUnusedOperations(w writeLine, bundle *irv1.RuleBundle) {
	used := map[string]bool{}
	for _, p := range bundle.GetPrograms() {
		for _, n := range p.GetNodes() {
			shape, err := shapeOf(n)
			if err != nil {
				continue
			}
			if op, ok := features.LookupOp(shape.category, shape.code); ok {
				used[op.Symbol] = true
			}
		}
	}
	var unused []string
	for _, op := range features.Ops() {
		if !used[op.Symbol] {
			unused = append(unused, op.Symbol)
		}
	}
	sort.Strings(unused)

	w("## Operations no rule exercises")
	w("")
	w("%d of %d operations. Every engine implements these and no conformance case",
		len(unused), len(features.Ops()))
	w("proves that implementation, so the list is the untested surface of the IR.")
	w("It shrinks when a rule needs one of them, and it must not grow silently.")
	w("")
	if len(unused) == 0 {
		w("None.")
		w("")
		return
	}
	w("| Operation |")
	w("|---|")
	for _, symbol := range unused {
		w("| `%s` |", symbol)
	}
	w("")
}

func renderMissingChecksums(w writeLine, bundle *irv1.RuleBundle) {
	w("## Rules without a published checksum")
	w("")
	found := false
	for _, d := range bundle.GetIdentifiers() {
		if d.ChecksumProgram != nil {
			continue
		}
		found = true
		country := globalCountry
		if d.CountryCode != nil {
			country = d.GetCountryCode()
		}
		w("- `%s` / `%s`: reports `%s`.", d.GetKind(), country,
			strings.ToLower(strings.TrimPrefix(
				irv1.ReasonCode_name[int32(d.GetAbsentChecksumReason())], "REASON_CODE_")))
	}
	if !found {
		w("Every definition carries a checksum program.")
	}
	w("")
}

func renderCapabilities(w writeLine, bundle *irv1.RuleBundle) {
	w("## Required capabilities")
	w("")
	w("| ID | Name | Content |")
	w("|---:|---|---|")
	for _, id := range bundle.GetRequiredFeatureIds() {
		c, ok := features.Lookup(id)
		if !ok {
			continue
		}
		w("| %d | `%s` | %s |", c.ID, c.Name, c.Summary)
	}
	w("")
}

type sourceRow struct{ id, authority, jurisdiction, accessed, terms, url string }

func renderProvenance(w writeLine, bundle *irv1.RuleBundle) {
	w("## Provenance")
	w("")
	w("| Source | Authority | Jurisdiction | Accessed | Terms |")
	w("|---|---|---|---|---|")
	seen := map[string]bool{}
	var rows []sourceRow
	for _, d := range bundle.GetIdentifiers() {
		for _, s := range d.GetSources() {
			if seen[s.GetId()] {
				continue
			}
			seen[s.GetId()] = true
			rows = append(rows, sourceRow{s.GetId(), s.GetAuthority(), s.GetJurisdiction(),
				s.GetAccessedAt(), s.GetLicenseOrTerms(), s.GetUrl()})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].id < rows[j].id })
	for _, r := range rows {
		w("| [`%s`](%s) | %s | %s | %s | %s |", r.id, r.url, r.authority, r.jurisdiction, r.accessed, r.terms)
	}
	w("")
}

func renderConformanceStatistics(w writeLine, suite *conformancev1.ConformanceBundle) {
	w("## Conformance statistics")
	w("")
	w("Total cases: **%d**.", len(suite.GetCases()))
	w("")
	w("| Kind | Cases |")
	w("|---|---:|")
	byKind := map[string]int{}
	for _, c := range suite.GetCases() {
		key := c.GetKind()
		if key == "" {
			key = "(loader)"
		}
		byKind[key]++
	}
	for _, k := range sortedCountKeys(byKind) {
		w("| `%s` | %d |", k, byKind[k])
	}
	w("")
	w("| Tag | Cases |")
	w("|---|---:|")
	for _, tag := range sortedTagCounts(suite) {
		w("| `%s` | %d |", tag.name, tag.count)
	}
	w("")
	w("| Data classification | Cases |")
	w("|---|---:|")
	byClass := map[string]int{}
	for _, c := range suite.GetCases() {
		byClass[c.GetDataClassification()]++
	}
	for _, k := range sortedCountKeys(byClass) {
		w("| `%s` | %d |", k, byClass[k])
	}
	w("")
}

func sortedCountKeys(counts map[string]int) []string {
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type tagCount struct {
	name  string
	count int
}

func sortedTagCounts(bundle *conformancev1.ConformanceBundle) []tagCount {
	counts := map[string]int{}
	for _, c := range bundle.GetCases() {
		for _, tag := range c.GetTags() {
			counts[tag]++
		}
	}
	out := make([]tagCount, 0, len(counts))
	for name, count := range counts {
		out = append(out, tagCount{name, count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// TagCounts returns the conformance tag histogram for the manifest.
func TagCounts(bundle *conformancev1.ConformanceBundle) map[string]int {
	out := map[string]int{}
	for _, t := range sortedTagCounts(bundle) {
		out[t.name] = t.count
	}
	return out
}

// ProvenanceFigures are the counts the provenance note quotes.
//
// They were read out of the rendered coverage.md by a shell script, with sed
// patterns matching prose that nothing forced to stay stable. Counting the
// bundle directly is both simpler and one fewer thing that drifts silently.
type ProvenanceFigures struct {
	Definitions  int
	Nodes        int
	Capabilities int
	UsedOps      int
	TotalOps     int
}

// FiguresOf counts what the provenance note states, from the bundle itself.
func FiguresOf(bundle *irv1.RuleBundle) ProvenanceFigures {
	out := ProvenanceFigures{
		Definitions:  len(bundle.GetIdentifiers()),
		Capabilities: len(bundle.GetRequiredFeatureIds()),
		TotalOps:     len(features.Ops()),
	}
	used := map[string]bool{}
	for _, p := range bundle.GetPrograms() {
		out.Nodes += len(p.GetNodes())
		for _, n := range p.GetNodes() {
			shape, err := shapeOf(n)
			if err != nil {
				continue
			}
			if op, ok := features.LookupOp(shape.category, shape.code); ok {
				used[op.Symbol] = true
			}
		}
	}
	out.UsedOps = len(used)
	return out
}

// ProvenanceInput is everything one engine's provenance note is made of.
type ProvenanceInput struct {
	Commit    string
	Version   string
	Stability string
	Figures   ProvenanceFigures
	Body      []byte
	Notes     []byte
}

// RenderProvenance assembles the note an engine ships as spec/PROVENANCE.md.
func RenderProvenance(in ProvenanceInput) []byte {
	body := string(in.Body)
	for placeholder, value := range map[string]int{
		"{{DEFINITIONS}}":  in.Figures.Definitions,
		"{{NODES}}":        in.Figures.Nodes,
		"{{CAPABILITIES}}": in.Figures.Capabilities,
		"{{USED_OPS}}":     in.Figures.UsedOps,
		"{{TOTAL_OPS}}":    in.Figures.TotalOps,
	} {
		body = strings.ReplaceAll(body, placeholder, strconv.Itoa(value))
	}
	var b strings.Builder
	b.WriteString("# Where these files come from, and what to build\n\n")
	b.WriteString("Copied from `github.com/entid-org/spec` at commit\n")
	fmt.Fprintf(&b, "`%s`, rules version\n`%s`, stability `%s`.\n\n",
		in.Commit, in.Version, in.Stability)
	b.WriteString(body)
	b.WriteString("\n")
	b.Write(in.Notes)
	return []byte(b.String())
}
