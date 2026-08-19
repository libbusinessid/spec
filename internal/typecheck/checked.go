// Package typecheck resolves every expression of the LibBusinessID language
// into a fully typed operation graph. No dynamic type survives this stage: the
// result is directly lowerable to the Protobuf IR.
package typecheck

import (
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/diagnostics"
	"github.com/libbusinessid/spec/internal/features"
)

// Node is a checked operation of a program.
type Node struct {
	// Op is the catalog entry of the operation.
	Op features.Op
	// Inputs are the checked operands, in syntactic order.
	Inputs []*Node
	// Pos anchors diagnostics and golden output to the source.
	Pos diagnostics.Position

	Text        *string
	Replacement *string
	MessageKey  *string
	Start       *uint32
	End         *uint32
	Index       *uint32
	Length      *uint32
	MinLength   *uint32
	MaxLength   *uint32
	Lengths     []uint32
	Values      []string
	Modulus     *int64
	Constant    *int64
	Weights     []int64
	Alignment   *irv1.WeightAlignment
	Mapping     *irv1.CharMapping
	Remainders  []int64
	ReasonCode  *irv1.ReasonCode

	// CallTarget is the callee symbol of a call operation.
	CallTarget string

	// MaxLen is the statically proven maximum number of code points of a
	// string node, or -1 when the bound is unknown.
	MaxLen int
}

// Type returns the static output type of the node.
func (n *Node) Type() irv1.ValueType { return n.Op.Output }

// Capture is a named node of a format program.
type Capture struct {
	Name string
	Node *Node
	Pos  diagnostics.Position
}

// Program is a checked program ready for lowering.
type Program struct {
	// Symbol is the fully qualified declaration symbol.
	Symbol string
	Kind   irv1.ProgramKind
	Root   *Node
	// Subject is the node producing subject() at top level, or nil.
	Subject  *Node
	Captures []Capture
	Pos      diagnostics.Position
}

// Unit is the checked compilation unit.
type Unit struct {
	// Programs are ordered by symbol.
	Programs []*Program
	// BySymbol indexes the programs.
	BySymbol map[string]*Program
}
