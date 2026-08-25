package typecheck

import (
	"sort"
	"unicode/utf8"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/ast"
	"github.com/entid-org/spec/internal/diagnostics"
	"github.com/entid-org/spec/internal/features"
	"github.com/entid-org/spec/internal/limits"
	"github.com/entid-org/spec/internal/linker"
)

// Diagnostic codes emitted by the typechecker.
const (
	CodeUnknownFunction = "TYPE001"
	CodeArity           = "TYPE002"
	CodeTypeMismatch    = "TYPE003"
	CodeBadConstant     = "TYPE004"
	CodeBounds          = "TYPE005"
	CodeUnknownCapture  = "TYPE006"
	CodeContext         = "TYPE007"
	CodeReasonCode      = "TYPE008"
	CodeDuplicate       = "TYPE009"
	CodeLimit           = "TYPE010"
	CodeStaticProof     = "TYPE011"
)

// unknownLength marks a string node whose maximum length is not statically
// provable.
const unknownLength = -1

// Check resolves every declaration of the symbol table into a checked program.
func Check(table *linker.Table) (*Unit, *diagnostics.Bag) {
	bag := diagnostics.New()
	c := &checker{table: table, bag: bag}
	unit := &Unit{BySymbol: map[string]*Program{}}

	for _, symbol := range sortedKeys(table.Canonicalizers) {
		c.addProgram(unit, c.checkCanonicalizer(table.Canonicalizers[symbol]))
	}
	for _, symbol := range sortedKeys(table.Formats) {
		c.addProgram(unit, c.checkFormat(table.Formats[symbol]))
	}
	for _, symbol := range sortedKeys(table.Checksums) {
		c.addProgram(unit, c.checkChecksum(table.Checksums[symbol]))
	}
	c.checkCanonicalizerUsage(unit)
	return unit, bag
}

func (c *checker) addProgram(unit *Unit, p *Program) {
	if p == nil {
		return
	}
	unit.Programs = append(unit.Programs, p)
	unit.BySymbol[p.Symbol] = p
}

type checker struct {
	table *linker.Table
	bag   *diagnostics.Bag
}

// ctx carries the language restrictions of the program being checked.
type ctx struct {
	symbol        string
	kind          irv1.ProgramKind
	allowSubject  bool
	allowCaptures bool
	captures      map[string]*Node
	nodes         int
}

func (c *checker) checkCanonicalizer(d *ast.Canonicalizer) *Program {
	cx := &ctx{symbol: d.Symbol(), kind: irv1.ProgramKind_PROGRAM_KIND_CANONICALIZATION}
	seq, ok := features.LookupOp(features.CategoryCanonicalization,
		int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_SEQUENCE))
	if !ok {
		return nil
	}
	root := &Node{Op: seq, Pos: d.Position, MaxLen: unknownLength}
	for _, step := range d.Steps {
		n := c.expr(cx, step, irv1.ValueType_VALUE_TYPE_CANONICALIZATION_STEP)
		if n == nil {
			continue
		}
		root.Inputs = append(root.Inputs, n)
	}
	c.checkNodeBudget(cx, d.Position)
	return &Program{Symbol: d.Symbol(), Kind: cx.kind, Root: root, Pos: d.Position}
}

func (c *checker) checkFormat(d *ast.Format) *Program {
	cx := &ctx{
		symbol:        d.Symbol(),
		kind:          irv1.ProgramKind_PROGRAM_KIND_FORMAT,
		allowSubject:  true,
		allowCaptures: true,
		captures:      map[string]*Node{},
	}
	p := &Program{Symbol: d.Symbol(), Kind: cx.kind, Pos: d.Position}
	if d.Subject != nil {
		p.Subject = c.expr(cx, d.Subject, irv1.ValueType_VALUE_TYPE_STRING)
	}
	if len(d.Captures) > limits.MaxCapturesPerFormat {
		c.bag.Errorf(d.Position, CodeLimit, "%s declares %d captures, the limit is %d",
			d.Symbol(), len(d.Captures), limits.MaxCapturesPerFormat)
	}
	for _, capture := range d.Captures {
		if _, dup := cx.captures[capture.Name]; dup {
			c.bag.Errorf(capture.Position, CodeDuplicate, "duplicate capture %q", capture.Name)
			continue
		}
		n := c.expr(cx, capture.Value, irv1.ValueType_VALUE_TYPE_STRING)
		if n == nil {
			continue
		}
		cx.captures[capture.Name] = n
		p.Captures = append(p.Captures, Capture{Name: capture.Name, Node: n, Pos: capture.Position})
	}

	seq, _ := features.LookupOp(features.CategoryAssertion,
		int32(irv1.AssertionOpKind_ASSERTION_OP_KIND_SEQUENCE))
	root := &Node{Op: seq, Pos: d.Position, MaxLen: unknownLength}
	for _, check := range d.Checks {
		n := c.expr(cx, check, irv1.ValueType_VALUE_TYPE_ASSERTION)
		if n == nil {
			continue
		}
		root.Inputs = append(root.Inputs, n)
	}
	callOp, _ := features.LookupOp(features.CategoryCall, int32(irv1.CallOpKind_CALL_OP_KIND_FORMAT))
	for _, use := range d.Uses {
		target, ok := c.table.Formats[use.Rule.String()]
		if !ok {
			c.bag.Errorf(use.Rule.Position, CodeUnknownFunction, "unknown format %q", use.Rule.String())
			continue
		}
		input := c.expr(cx, use.Input, irv1.ValueType_VALUE_TYPE_STRING)
		if input == nil {
			continue
		}
		root.Inputs = append(root.Inputs, &Node{
			Op:         callOp,
			Inputs:     []*Node{input},
			Pos:        use.Position,
			CallTarget: target.Symbol(),
			MaxLen:     unknownLength,
		})
	}
	if len(root.Inputs) == 0 {
		c.bag.Errorf(d.Position, CodeArity, "%s declares no check", d.Symbol())
		return nil
	}
	p.Root = root
	c.checkNodeBudget(cx, d.Position)
	return p
}

func (c *checker) checkChecksum(d *ast.Checksum) *Program {
	cx := &ctx{
		symbol:       d.Symbol(),
		kind:         irv1.ProgramKind_PROGRAM_KIND_CHECKSUM,
		allowSubject: true,
	}
	p := &Program{Symbol: d.Symbol(), Kind: cx.kind, Pos: d.Position}
	if d.Subject != nil {
		p.Subject = c.expr(cx, d.Subject, irv1.ValueType_VALUE_TYPE_STRING)
	}
	root := c.expr(cx, d.Rule, irv1.ValueType_VALUE_TYPE_CHECKSUM_OUTCOME)
	if root == nil {
		return nil
	}
	if root.Op.Category == features.CategoryChecksum &&
		root.Op.Code == int32(irv1.ChecksumOpKind_CHECKSUM_OP_KIND_WHEN) {
		c.bag.Suggestf(root.Pos, CodeContext, "wrap the branch in choose(...)",
			"when_checksum is only accepted as a direct branch of choose")
		return nil
	}
	p.Root = root
	c.checkNodeBudget(cx, d.Position)
	return p
}

func (c *checker) checkNodeBudget(cx *ctx, pos diagnostics.Position) {
	if cx.nodes > limits.MaxNodesPerProgram {
		c.bag.Errorf(pos, CodeLimit, "%s uses %d nodes, the limit is %d",
			cx.symbol, cx.nodes, limits.MaxNodesPerProgram)
	}
}

// checkCanonicalizerUsage applies the restrictions that depend on how a
// canonicalizer is referenced.
func (c *checker) checkCanonicalizerUsage(unit *Unit) {
	for _, d := range c.table.Dispatchers {
		if d.PreCanonicalizer == nil {
			continue
		}
		p, ok := unit.BySymbol[d.PreCanonicalizer.String()]
		if !ok {
			continue
		}
		for _, step := range p.Root.Inputs {
			if !preCanonicalizationAllowed(step.Op.Code) {
				c.bag.Suggestf(step.Pos, CodeContext,
					"a routing pre-canonicalizer only accepts trim_whitespace, remove_whitespace, uppercase_ascii and remove_chars",
					"%s is not allowed in the pre-canonicalizer of dispatcher %q", step.Op.HCLName(), d.Kind)
			}
		}
	}
	for _, id := range c.table.IdentifierOrder {
		if id.Canonicalizer == nil || !id.Global {
			continue
		}
		p, ok := unit.BySymbol[id.Canonicalizer.String()]
		if !ok {
			continue
		}
		var walk func(n *Node)
		walk = func(n *Node) {
			if n.Op.Category == features.CategoryCanonicalization &&
				n.Op.Code == int32(irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_PREPEND_COUNTRY_IF_MISSING) {
				c.bag.Errorf(n.Pos, CodeContext,
					"prepend_country_if_missing cannot be used by the GLOBAL definition %s", id.Symbol())
			}
			for _, in := range n.Inputs {
				walk(in)
			}
		}
		walk(p.Root)
	}
}

func preCanonicalizationAllowed(code int32) bool {
	switch irv1.CanonicalizationOpKind(code) {
	case irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_TRIM_WHITESPACE,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_WHITESPACE,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_UPPERCASE_ASCII,
		irv1.CanonicalizationOpKind_CANONICALIZATION_OP_KIND_REMOVE_CHARS:
		return true
	default:
		return false
	}
}

func sortedKeys[T any](m map[string]T) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func stringPtr(s string) *string { return &s }
func uint32Ptr(v uint32) *uint32 { return &v }
func int64Ptr(v int64) *int64    { return &v }
func sortedUnique(s string) string {
	runes := []rune(s)
	sort.Slice(runes, func(i, j int) bool { return runes[i] < runes[j] })
	out := make([]rune, 0, len(runes))
	for i, r := range runes {
		if i > 0 && runes[i-1] == r {
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }
