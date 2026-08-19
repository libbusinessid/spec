// Package ast holds the typed abstract syntax tree of the LibBusinessID HCL
// language. The AST is produced by internal/hcllang, resolved by
// internal/linker and internal/typecheck, then lowered by internal/lower.
//
// The AST never carries an HCL expression: every construct is a closed, typed
// node with an exact source position.
package ast

import "github.com/libbusinessid/spec/internal/diagnostics"

// Expr is a source expression of the language.
type Expr interface {
	// Pos returns the source position of the expression.
	Pos() diagnostics.Position
	// exprKind returns the short label of the node and keeps the interface
	// closed to this package.
	exprKind() string
}

// CallExpr is a function call such as `slice(value(), 0, 2)`.
type CallExpr struct {
	Name     string
	Args     []Expr
	Position diagnostics.Position
}

// Pos implements Expr.
func (e *CallExpr) Pos() diagnostics.Position { return e.Position }
func (e *CallExpr) exprKind() string          { return "call" }

// StringLit is a quoted literal without any interpolation.
type StringLit struct {
	Value    string
	Position diagnostics.Position
}

// Pos implements Expr.
func (e *StringLit) Pos() diagnostics.Position { return e.Position }
func (e *StringLit) exprKind() string          { return "string" }

// IntLit is an integer literal.
type IntLit struct {
	Value    int64
	Position diagnostics.Position
}

// Pos implements Expr.
func (e *IntLit) Pos() diagnostics.Position { return e.Position }
func (e *IntLit) exprKind() string          { return "integer" }

// BoolLit is a boolean literal.
type BoolLit struct {
	Value    bool
	Position diagnostics.Position
}

// Pos implements Expr.
func (e *BoolLit) Pos() diagnostics.Position { return e.Position }
func (e *BoolLit) exprKind() string          { return "boolean" }

// ListExpr is a bracketed list such as `["A", "B"]`.
type ListExpr struct {
	Items    []Expr
	Position diagnostics.Position
}

// Pos implements Expr.
func (e *ListExpr) Pos() diagnostics.Position { return e.Position }
func (e *ListExpr) exprKind() string          { return "list" }

// RefExpr is a dotted symbolic reference such as `format.fr.siren` or
// `capture.registration`.
type RefExpr struct {
	Parts    []string
	Position diagnostics.Position
}

// Pos implements Expr.
func (e *RefExpr) Pos() diagnostics.Position { return e.Position }
func (e *RefExpr) exprKind() string          { return "reference" }

// String renders the reference as written in the source.
func (e *RefExpr) String() string {
	out := ""
	for i, p := range e.Parts {
		if i > 0 {
			out += "."
		}
		out += p
	}
	return out
}

// Canonicalizer is a `canonicalizer "<namespace>" "<name>"` declaration.
type Canonicalizer struct {
	Namespace string
	Name      string
	Steps     []Expr
	Position  diagnostics.Position
}

// Symbol returns the fully qualified symbol name.
func (d *Canonicalizer) Symbol() string { return "canonicalizer." + d.Namespace + "." + d.Name }

// Capture is a `capture "<name>"` block of a format declaration.
type Capture struct {
	Name     string
	Value    Expr
	Position diagnostics.Position
}

// UseFormat is a `use_format` block of a format declaration.
type UseFormat struct {
	Rule     *RefExpr
	Input    Expr
	Position diagnostics.Position
}

// Format is a `format "<namespace>" "<name>"` declaration.
type Format struct {
	Namespace string
	Name      string
	Subject   Expr
	Captures  []*Capture
	Checks    []Expr
	Uses      []*UseFormat
	Position  diagnostics.Position
}

// Symbol returns the fully qualified symbol name.
func (d *Format) Symbol() string { return "format." + d.Namespace + "." + d.Name }

// Checksum is a `checksum "<namespace>" "<name>"` declaration.
type Checksum struct {
	Namespace string
	Name      string
	Subject   Expr
	Rule      Expr
	Position  diagnostics.Position
}

// Symbol returns the fully qualified symbol name.
func (d *Checksum) Symbol() string { return "checksum." + d.Namespace + "." + d.Name }

// CountryAlias maps a country token to a canonical ISO 3166-1 alpha-2 code.
type CountryAlias struct {
	Alias       string
	CountryCode string
	Position    diagnostics.Position
}

// DispatchTarget is a `target` block of a dispatcher.
type DispatchTarget struct {
	Global                        bool
	CountryCode                   string
	AcceptedPrefixes              []string
	HasCanonicalPrefix            bool
	CanonicalPrefix               string
	Identifier                    *RefExpr
	AllowUnprefixedWithoutCountry bool
	Position                      diagnostics.Position
	CountryPosition               diagnostics.Position
	PrefixPositions               []diagnostics.Position
}

// Dispatcher is a `dispatcher "<kind>"` declaration.
type Dispatcher struct {
	Kind             string
	Aliases          []string
	AliasPositions   []diagnostics.Position
	PreCanonicalizer *RefExpr
	CountryAliases   []*CountryAlias
	Targets          []*DispatchTarget
	Position         diagnostics.Position
}

// Symbol returns the fully qualified symbol name.
func (d *Dispatcher) Symbol() string { return "dispatcher." + d.Kind }

// Source is a `source` block of an identifier declaration.
type Source struct {
	ID             string
	URL            string
	Authority      string
	Title          string
	AccessedAt     string
	Jurisdiction   string
	Language       string
	Notes          string
	LicenseOrTerms string
	HasArchiveURL  bool
	ArchiveURL     string
	Position       diagnostics.Position
}

// NoChecksum is the explicit declaration that a definition has no checksum.
type NoChecksum struct {
	ReasonCode string
	Notes      string
	Position   diagnostics.Position
}

// Identifier is an `identifier "<kind>" "<country_or_GLOBAL>"` declaration.
type Identifier struct {
	Kind           string
	Global         bool
	CountryCode    string
	Canonicalizer  *RefExpr
	Format         *RefExpr
	Checksum       *RefExpr
	NoChecksum     *NoChecksum
	DefaultProfile string
	Sources        []*Source
	Position       diagnostics.Position
}

// Symbol returns the fully qualified symbol name.
func (d *Identifier) Symbol() string {
	country := "GLOBAL"
	if !d.Global {
		country = d.CountryCode
	}
	return "identifier." + d.Kind + "." + country
}

// File is one parsed `*.hcl` source file.
type File struct {
	// Path is the POSIX relative path of the file inside the rules root.
	Path           string
	Canonicalizers []*Canonicalizer
	Formats        []*Format
	Checksums      []*Checksum
	Dispatchers    []*Dispatcher
	Identifiers    []*Identifier
}

// Unit is the whole compilation unit: every rule file, ordered by path.
type Unit struct {
	Files []*File
}

// Canonicalizers returns every canonicalizer declaration in file order.
func (u *Unit) Canonicalizers() []*Canonicalizer {
	var out []*Canonicalizer
	for _, f := range u.Files {
		out = append(out, f.Canonicalizers...)
	}
	return out
}

// Formats returns every format declaration in file order.
func (u *Unit) Formats() []*Format {
	var out []*Format
	for _, f := range u.Files {
		out = append(out, f.Formats...)
	}
	return out
}

// Checksums returns every checksum declaration in file order.
func (u *Unit) Checksums() []*Checksum {
	var out []*Checksum
	for _, f := range u.Files {
		out = append(out, f.Checksums...)
	}
	return out
}

// Dispatchers returns every dispatcher declaration in file order.
func (u *Unit) Dispatchers() []*Dispatcher {
	var out []*Dispatcher
	for _, f := range u.Files {
		out = append(out, f.Dispatchers...)
	}
	return out
}

// Identifiers returns every identifier declaration in file order.
func (u *Unit) Identifiers() []*Identifier {
	var out []*Identifier
	for _, f := range u.Files {
		out = append(out, f.Identifiers...)
	}
	return out
}

// ExprKind returns the short label of an expression, used by the diagnostics
// of the compiler stages.
func ExprKind(e Expr) string {
	if e == nil {
		return "none"
	}
	return e.exprKind()
}

// Walk calls fn for e and every sub-expression, in pre-order.
func Walk(e Expr, fn func(Expr)) {
	if e == nil {
		return
	}
	fn(e)
	switch n := e.(type) {
	case *CallExpr:
		for _, a := range n.Args {
			Walk(a, fn)
		}
	case *ListExpr:
		for _, item := range n.Items {
			Walk(item, fn)
		}
	}
}
