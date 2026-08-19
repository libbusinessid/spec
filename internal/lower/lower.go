// Package lower converts the checked program graph into the Protobuf IR.
//
// Every identifier, ordering and node index produced here is a pure function
// of the sorted, fully qualified symbol names and of the syntactic order of the
// operands. No Go map iteration can influence the result.
package lower

import (
	"sort"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/ast"
	"github.com/libbusinessid/spec/internal/diagnostics"
	"github.com/libbusinessid/spec/internal/features"
	"github.com/libbusinessid/spec/internal/limits"
	"github.com/libbusinessid/spec/internal/linker"
	"github.com/libbusinessid/spec/internal/optimize"
	"github.com/libbusinessid/spec/internal/typecheck"
)

// Diagnostic codes emitted by the lowering stage.
const (
	CodeLimit     = "IR001"
	CodeInternal  = "IR002"
	CodeDuplicate = "IR003"
)

// FormatVersion is the structural version of the IR emitted by this compiler.
const FormatVersion uint32 = 1

// Options controls the optional lowering behaviours.
type Options struct {
	// RulesVersion is the business version of the emitted bundle.
	RulesVersion string
	// Optimize enables structural deduplication of identical sub-graphs.
	Optimize bool
}

// Lower builds the rule bundle from the checked compilation unit.
func Lower(table *linker.Table, unit *typecheck.Unit, opts Options) (*irv1.RuleBundle, *diagnostics.Bag) {
	bag := diagnostics.New()
	l := &lowerer{table: table, unit: unit, bag: bag, opts: opts, used: features.NewSet()}
	bundle := l.build()
	return bundle, bag
}

type lowerer struct {
	table *linker.Table
	unit  *typecheck.Unit
	bag   *diagnostics.Bag
	opts  Options
	used  *features.Set

	programID    map[string]uint32
	identifierID map[string]uint32
	totalNodes   int
}

func (l *lowerer) build() *irv1.RuleBundle {
	l.assignProgramIDs()
	l.assignIdentifierIDs()

	bundle := &irv1.RuleBundle{
		FormatVersion: FormatVersion,
		RulesVersion:  l.opts.RulesVersion,
	}
	l.used.Add(features.CoreGraphV1, features.ProfilesV1)

	symbols := make([]string, 0, len(l.programID))
	for s := range l.programID {
		symbols = append(symbols, s)
	}
	sort.Strings(symbols)
	programs := make([]*irv1.Program, 0, len(symbols))
	for _, symbol := range symbols {
		programs = append(programs, l.lowerProgram(l.unit.BySymbol[symbol]))
	}
	sort.Slice(programs, func(i, j int) bool { return programs[i].GetId() < programs[j].GetId() })
	bundle.Programs = programs

	bundle.Identifiers = l.lowerIdentifiers()
	bundle.Dispatchers = l.lowerDispatchers()
	if l.totalNodes > limits.MaxTotalNodes {
		l.bag.Errorf(diagnostics.Position{}, CodeLimit,
			"the bundle holds %d nodes, the limit is %d", l.totalNodes, limits.MaxTotalNodes)
	}
	bundle.RequiredFeatureIds = l.used.Sorted()
	return bundle
}

func (l *lowerer) assignProgramIDs() {
	symbols := make([]string, 0, len(l.unit.Programs))
	for _, p := range l.unit.Programs {
		symbols = append(symbols, p.Symbol)
	}
	sort.Strings(symbols)
	// The checked unit indexes its programs by symbol, so the names are unique
	// by construction.
	l.programID = make(map[string]uint32, len(symbols))
	for i, s := range symbols {
		l.programID[s] = uint32(i + 1)
	}
}

func (l *lowerer) assignIdentifierIDs() {
	symbols := make([]string, 0, len(l.table.IdentifierOrder))
	for _, id := range l.table.IdentifierOrder {
		symbols = append(symbols, id.Symbol())
	}
	sort.Strings(symbols)
	l.identifierID = make(map[string]uint32, len(symbols))
	for i, s := range symbols {
		l.identifierID[s] = uint32(i + 1)
	}
	if len(symbols) > limits.MaxIdentifiers {
		l.bag.Errorf(diagnostics.Position{}, CodeLimit,
			"the bundle holds %d identifiers, the limit is %d", len(symbols), limits.MaxIdentifiers)
	}
}

// emitter turns a checked node graph into a topologically ordered node list.
type emitter struct {
	l      *lowerer
	nodes  []*irv1.Node
	byNode map[*typecheck.Node]uint32
	byKey  map[string]uint32
	pos    diagnostics.Position
}

func (e *emitter) emit(n *typecheck.Node) uint32 {
	if idx, ok := e.byNode[n]; ok {
		return idx
	}
	inputs := make([]uint32, 0, len(n.Inputs))
	for _, in := range n.Inputs {
		inputs = append(inputs, e.emit(in))
	}
	node := e.l.buildNode(n, inputs)
	if e.l.opts.Optimize {
		key := structuralKey(node)
		if idx, ok := e.byKey[key]; ok {
			e.byNode[n] = idx
			return idx
		}
		e.byKey[key] = uint32(len(e.nodes))
	}
	idx := uint32(len(e.nodes))
	e.nodes = append(e.nodes, node)
	e.byNode[n] = idx
	return idx
}

func (l *lowerer) lowerProgram(p *typecheck.Program) *irv1.Program {
	e := &emitter{l: l, byNode: map[*typecheck.Node]uint32{}, byKey: map[string]uint32{}, pos: p.Pos}
	out := &irv1.Program{Id: l.programID[p.Symbol], Kind: p.Kind}
	if p.Subject != nil {
		idx := e.emit(p.Subject)
		out.SubjectNode = &idx
	}
	for _, capture := range p.Captures {
		out.Captures = append(out.Captures, &irv1.Capture{Name: capture.Name, Node: e.emit(capture.Node)})
	}
	out.RootNode = e.emit(p.Root)
	out.Nodes = e.nodes
	if len(out.Nodes) > limits.MaxNodesPerProgram {
		l.bag.Errorf(p.Pos, CodeLimit, "%s holds %d nodes, the limit is %d",
			p.Symbol, len(out.Nodes), limits.MaxNodesPerProgram)
	}
	l.totalNodes += len(out.Nodes)
	if len(out.Captures) > 0 {
		l.used.Add(features.CapturesAndCallsV1)
	}
	return out
}

func (l *lowerer) buildNode(n *typecheck.Node, inputs []uint32) *irv1.Node {
	l.used.Add(n.Op.Features...)
	node := &irv1.Node{OutputType: n.Op.Output, InputNodes: inputs}
	switch n.Op.Category {
	case features.CategoryString:
		op := &irv1.StringOperation{Kind: irv1.StringOpKind(n.Op.Code)}
		op.Text = n.Text
		op.Start = n.Start
		op.End = n.End
		node.Operation = &irv1.Node_StringOperation{StringOperation: op}
	case features.CategoryInteger:
		op := &irv1.IntegerOperation{Kind: irv1.IntegerOpKind(n.Op.Code)}
		op.Modulus = n.Modulus
		op.Weights = n.Weights
		op.Alignment = n.Alignment
		op.Mapping = n.Mapping
		op.RemainderValues = n.Remainders
		node.Operation = &irv1.Node_IntegerOperation{IntegerOperation: op}
	case features.CategoryPredicate:
		op := &irv1.PredicateOperation{Kind: irv1.PredicateOpKind(n.Op.Code)}
		op.Text = n.Text
		op.Values = n.Values
		op.Lengths = n.Lengths
		op.Length = n.Length
		op.MinLength = n.MinLength
		op.MaxLength = n.MaxLength
		op.Index = n.Index
		node.Operation = &irv1.Node_PredicateOperation{PredicateOperation: op}
	case features.CategoryCanonicalization:
		op := &irv1.CanonicalizationOperation{Kind: irv1.CanonicalizationOpKind(n.Op.Code)}
		op.Text = n.Text
		op.Replacement = n.Replacement
		op.Index = n.Index
		op.Length = n.Length
		node.Operation = &irv1.Node_CanonicalizationOperation{CanonicalizationOperation: op}
	case features.CategoryAssertion:
		op := &irv1.AssertionOperation{Kind: irv1.AssertionOpKind(n.Op.Code)}
		op.ReasonCode = n.ReasonCode
		op.MessageKey = n.MessageKey
		node.Operation = &irv1.Node_AssertionOperation{AssertionOperation: op}
	case features.CategoryChecksum:
		op := &irv1.ChecksumOperation{Kind: irv1.ChecksumOpKind(n.Op.Code)}
		op.Index = n.Index
		op.Start = n.Start
		op.End = n.End
		op.ReasonCode = n.ReasonCode
		op.MessageKey = n.MessageKey
		op.Constant = n.Constant
		node.Operation = &irv1.Node_ChecksumOperation{ChecksumOperation: op}
	case features.CategoryCall:
		id, ok := l.programID[n.CallTarget]
		if !ok {
			l.bag.Errorf(n.Pos, CodeInternal, "unresolved call target %q", n.CallTarget)
		}
		node.Operation = &irv1.Node_CallOperation{CallOperation: &irv1.CallOperation{
			Kind:      irv1.CallOpKind(n.Op.Code),
			ProgramId: id,
		}}
		l.used.Add(features.CapturesAndCallsV1)
	}
	return node
}

func (l *lowerer) lowerIdentifiers() []*irv1.IdentifierDefinition {
	out := make([]*irv1.IdentifierDefinition, 0, len(l.table.IdentifierOrder))
	for _, id := range l.table.IdentifierOrder {
		def := &irv1.IdentifierDefinition{
			Id:             l.identifierID[id.Symbol()],
			Kind:           id.Kind,
			DefaultProfile: id.DefaultProfile,
		}
		if !id.Global {
			country := id.CountryCode
			def.CountryCode = &country
		}
		if id.Canonicalizer != nil {
			def.CanonicalizationProgram = l.programID[id.Canonicalizer.String()]
		}
		if id.Format != nil {
			def.FormatProgram = l.programID[id.Format.String()]
		}
		switch {
		case id.Checksum != nil:
			pid := l.programID[id.Checksum.String()]
			def.ChecksumProgram = &pid
		case id.NoChecksum != nil:
			code, ok := reasonCodeByName(id.NoChecksum.ReasonCode)
			if ok {
				def.AbsentChecksumReason = &code
			}
			l.used.Add(features.ChecksumTristateV1)
		}
		def.Sources = lowerSources(id.Sources)
		if len(def.GetSources()) > 0 {
			l.used.Add(features.ProvenanceV1)
		}
		out = append(out, def)
	}
	sort.SliceStable(out, func(i, j int) bool { return identifierBefore(out[i], out[j]) })
	l.rejectDuplicateIdentifiers(out)
	return out
}

func (l *lowerer) rejectDuplicateIdentifiers(defs []*irv1.IdentifierDefinition) {
	for i := 1; i < len(defs); i++ {
		a, b := defs[i-1], defs[i]
		if a.GetKind() == b.GetKind() && a.GetCountryCode() == b.GetCountryCode() &&
			(a.CountryCode == nil) == (b.CountryCode == nil) {
			l.bag.Errorf(diagnostics.Position{}, CodeDuplicate,
				"two definitions share the sort key %s/%s", a.GetKind(), a.GetCountryCode())
		}
	}
}

// sourceTier maps the authored tier onto the wire enum. An unknown value is
// rejected by the linker, so the fallback is unreachable in practice and stays
// unspecified rather than guessing a tier.
func sourceTier(v string) irv1.SourceTier {
	switch v {
	case "primary":
		return irv1.SourceTier_SOURCE_TIER_PRIMARY
	case "secondary":
		return irv1.SourceTier_SOURCE_TIER_SECONDARY
	default:
		return irv1.SourceTier_SOURCE_TIER_UNSPECIFIED
	}
}

func lowerSources(sources []*ast.Source) []*irv1.Source {
	out := make([]*irv1.Source, 0, len(sources))
	for _, s := range sources {
		src := &irv1.Source{
			Id:             s.ID,
			Url:            s.URL,
			Authority:      s.Authority,
			Title:          s.Title,
			AccessedAt:     s.AccessedAt,
			Jurisdiction:   s.Jurisdiction,
			Language:       s.Language,
			Notes:          s.Notes,
			LicenseOrTerms: s.LicenseOrTerms,
			Tier:           sourceTier(s.Tier),
		}
		if s.HasArchiveURL {
			archive := s.ArchiveURL
			src.ArchiveUrl = &archive
		}
		out = append(out, src)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GetId() < out[j].GetId() })
	return out
}

func (l *lowerer) lowerDispatchers() []*irv1.IdentifierDispatcher {
	out := make([]*irv1.IdentifierDispatcher, 0, len(l.table.Dispatchers))
	for _, d := range l.table.Dispatchers {
		l.used.Add(features.IdentifierDispatchV1)
		disp := &irv1.IdentifierDispatcher{Kind: d.Kind}
		aliases := append([]string(nil), d.Aliases...)
		sort.Strings(aliases)
		disp.KindAliases = aliases
		if d.PreCanonicalizer != nil {
			disp.PreCanonicalizationProgram = l.programID[d.PreCanonicalizer.String()]
		}
		for _, ca := range d.CountryAliases {
			disp.CountryAliases = append(disp.CountryAliases,
				&irv1.CountryAlias{Alias: ca.Alias, CountryCode: ca.CountryCode})
		}
		sort.SliceStable(disp.CountryAliases, func(i, j int) bool {
			return disp.CountryAliases[i].GetAlias() < disp.CountryAliases[j].GetAlias()
		})
		for _, t := range d.Targets {
			target := &irv1.DispatchTarget{
				AllowUnprefixedWithoutCountry: t.AllowUnprefixedWithoutCountry,
			}
			if !t.Global {
				country := t.CountryCode
				target.CountryCode = &country
			}
			prefixes := append([]string(nil), t.AcceptedPrefixes...)
			sort.Strings(prefixes)
			target.AcceptedPrefixes = prefixes
			if t.HasCanonicalPrefix {
				prefix := t.CanonicalPrefix
				target.CanonicalPrefix = &prefix
			}
			if t.Identifier != nil {
				target.IdentifierDefinitionId = l.identifierID[t.Identifier.String()]
			}
			disp.Targets = append(disp.Targets, target)
		}
		sort.SliceStable(disp.Targets, func(i, j int) bool {
			return targetBefore(disp.Targets[i], disp.Targets[j])
		})
		out = append(out, disp)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].GetKind() < out[j].GetKind() })
	for i := 1; i < len(out); i++ {
		if out[i].GetKind() == out[i-1].GetKind() {
			l.bag.Errorf(diagnostics.Position{}, CodeDuplicate,
				"two dispatchers share the kind %q", out[i].GetKind())
		}
	}
	return out
}

func reasonCodeByName(name string) (irv1.ReasonCode, bool) {
	upper := make([]byte, 0, len(name))
	for i := 0; i < len(name); i++ {
		c := name[i]
		if c >= 'a' && c <= 'z' {
			c -= 'a' - 'A'
		}
		upper = append(upper, c)
	}
	v, ok := irv1.ReasonCode_value["REASON_CODE_"+string(upper)]
	if !ok || v == 0 {
		return irv1.ReasonCode_REASON_CODE_UNSPECIFIED, false
	}
	return irv1.ReasonCode(v), true
}

func structuralKey(node *irv1.Node) string { return optimize.StructuralKey(node) }

// identifierBefore is the normative serialization order of definitions: by
// kind, then GLOBAL first, then country code.
func identifierBefore(a, b *irv1.IdentifierDefinition) bool {
	if a.GetKind() != b.GetKind() {
		return a.GetKind() < b.GetKind()
	}
	switch {
	case a.CountryCode == nil && b.CountryCode == nil:
		return false
	case a.CountryCode == nil:
		return true
	case b.CountryCode == nil:
		return false
	default:
		return a.GetCountryCode() < b.GetCountryCode()
	}
}

// targetBefore is the normative serialization order of dispatch targets:
// GLOBAL first, then country code.
func targetBefore(a, b *irv1.DispatchTarget) bool {
	switch {
	case a.CountryCode == nil && b.CountryCode == nil:
		return false
	case a.CountryCode == nil:
		return true
	case b.CountryCode == nil:
		return false
	default:
		return a.GetCountryCode() < b.GetCountryCode()
	}
}

// newBag exists so that the internal tests can build a lowerer without the
// whole pipeline.
func newBag() *diagnostics.Bag { return diagnostics.New() }
