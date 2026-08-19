package artifact

import (
	"os"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/ast"
	"github.com/libbusinessid/spec/internal/diagnostics"
	"github.com/libbusinessid/spec/internal/hcllang"
	"github.com/libbusinessid/spec/internal/linker"
	"github.com/libbusinessid/spec/internal/lower"
	"github.com/libbusinessid/spec/internal/typecheck"
)

// CompileOptions controls a rule compilation.
type CompileOptions struct {
	// RulesVersion is the business version stamped into the bundle.
	RulesVersion string
	// Optimize enables structural deduplication of identical sub-graphs.
	Optimize bool
}

// CompileResult holds every artifact produced by a rule compilation.
type CompileResult struct {
	// Unit is the parsed compilation unit.
	Unit *ast.Unit
	// Table is the resolved symbol table.
	Table *linker.Table
	// Checked is the typed program graph.
	Checked *typecheck.Unit
	// Bundle is the emitted rule bundle.
	Bundle *irv1.RuleBundle
	// Bytes is the deterministic serialization of the bundle.
	Bytes []byte
	// SourceDigest is the canonical digest of the rule sources.
	SourceDigest [32]byte
	// Ruleset is the bundle re-loaded through the defensive validator.
	Ruleset *Ruleset
	// Files lists the compiled sources in canonical order.
	Files []hcllang.SourceFile
}

// CompileRules runs the whole rule pipeline: discovery, parsing, linking, type
// checking, lowering, digesting, serialization and defensive re-validation.
//
// The compiler always reloads its own output through LoadRuleset, so a bundle
// that the engines would refuse can never be published.
func CompileRules(rulesDir string, opts CompileOptions) (*CompileResult, *diagnostics.Bag) {
	bag := diagnostics.New()
	files, err := hcllang.Discover(rulesDir)
	if err != nil {
		bag.Errorf(diagnostics.Position{File: rulesDir}, "CLI001", "cannot read the rules directory: %v", err)
		return nil, bag
	}
	if len(files) == 0 {
		bag.Errorf(diagnostics.Position{File: rulesDir}, "CLI002", "no rule file found")
		return nil, bag
	}
	contents := make(map[string][]byte, len(files))
	unit, parseBag := hcllang.ParseUnit(files, func(sf hcllang.SourceFile) ([]byte, error) {
		data, err := os.ReadFile(sf.AbsPath)
		if err == nil {
			contents[sf.RelPath] = data
		}
		return data, err
	})
	bag.Extend(parseBag)
	if bag.HasErrors() {
		return nil, bag
	}
	table, linkBag := linker.Link(unit)
	bag.Extend(linkBag)
	checked, typeBag := typecheck.Check(table)
	bag.Extend(typeBag)
	if bag.HasErrors() {
		return nil, bag
	}
	bundle, lowerBag := lower.Lower(table, checked, lower.Options{
		RulesVersion: opts.RulesVersion,
		Optimize:     opts.Optimize,
	})
	bag.Extend(lowerBag)
	if bag.HasErrors() {
		return nil, bag
	}

	entries := make([]SourceEntry, 0, len(files))
	for _, f := range files {
		entries = append(entries, SourceEntry{Path: "rules/" + f.RelPath, Content: contents[f.RelPath]})
	}
	digest, err := SourceDigest(RulesDomain, entries)
	if err != nil {
		bag.Errorf(diagnostics.Position{}, "CLI003", "cannot compute the source digest: %v", err)
		return nil, bag
	}
	bundle.SourceDigest = digest[:]

	raw, err := Marshal(bundle)
	if err != nil {
		bag.Errorf(diagnostics.Position{}, "CLI004", "cannot serialize the bundle: %v", err)
		return nil, bag
	}
	ruleset, err := LoadRuleset(raw)
	if err != nil {
		bag.Errorf(diagnostics.Position{}, "CLI005", "the compiler produced a bundle the engines would refuse: %v", err)
		return nil, bag
	}
	return &CompileResult{
		Unit:         unit,
		Table:        table,
		Checked:      checked,
		Bundle:       bundle,
		Bytes:        raw,
		SourceDigest: digest,
		Ruleset:      ruleset,
		Files:        files,
	}, bag
}
