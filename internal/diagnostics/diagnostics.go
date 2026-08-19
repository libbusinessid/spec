// Package diagnostics provides the stable diagnostic model shared by every
// stage of businessidc: file, line, column, stable code, message and optional
// suggestion. Diagnostics are collected, sorted deterministically and rendered
// either for humans on stderr or as machine readable JSON.
package diagnostics

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// Severity classifies a diagnostic.
type Severity string

// The supported severities.
const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Position is a one based source position. Line zero means the diagnostic is
// attached to the whole file, and an empty file means it has no source anchor.
type Position struct {
	File   string `json:"file,omitempty"`
	Line   int    `json:"line,omitempty"`
	Column int    `json:"column,omitempty"`
}

// String renders the position as `file:line:column`.
func (p Position) String() string {
	switch {
	case p.File == "":
		return ""
	case p.Line <= 0:
		return p.File
	case p.Column <= 0:
		return fmt.Sprintf("%s:%d", p.File, p.Line)
	default:
		return fmt.Sprintf("%s:%d:%d", p.File, p.Line, p.Column)
	}
}

// Diagnostic is one problem reported by the compiler.
type Diagnostic struct {
	Severity   Severity `json:"severity"`
	Code       string   `json:"code"`
	Message    string   `json:"message"`
	Position   Position `json:"position,omitzero"`
	Suggestion string   `json:"suggestion,omitempty"`
}

// String renders a single line human readable diagnostic.
func (d Diagnostic) String() string {
	var b strings.Builder
	if pos := d.Position.String(); pos != "" {
		b.WriteString(pos)
		b.WriteString(": ")
	}
	b.WriteString(string(d.Severity))
	b.WriteString(" [")
	b.WriteString(d.Code)
	b.WriteString("] ")
	b.WriteString(d.Message)
	if d.Suggestion != "" {
		b.WriteString("\n  suggestion: ")
		b.WriteString(d.Suggestion)
	}
	return b.String()
}

// Bag collects diagnostics of a single compiler run.
type Bag struct {
	items []Diagnostic
}

// New returns an empty bag.
func New() *Bag { return &Bag{} }

// Add appends a diagnostic.
func (b *Bag) Add(d Diagnostic) {
	if d.Severity == "" {
		d.Severity = SeverityError
	}
	b.items = append(b.items, d)
}

// Errorf appends an error diagnostic.
func (b *Bag) Errorf(pos Position, code, format string, args ...any) {
	b.Add(Diagnostic{Severity: SeverityError, Code: code, Message: fmt.Sprintf(format, args...), Position: pos})
}

// Suggestf appends an error diagnostic carrying a suggestion.
func (b *Bag) Suggestf(pos Position, code, suggestion, format string, args ...any) {
	b.Add(Diagnostic{
		Severity:   SeverityError,
		Code:       code,
		Message:    fmt.Sprintf(format, args...),
		Position:   pos,
		Suggestion: suggestion,
	})
}

// Warnf appends a warning diagnostic.
func (b *Bag) Warnf(pos Position, code, format string, args ...any) {
	b.Add(Diagnostic{Severity: SeverityWarning, Code: code, Message: fmt.Sprintf(format, args...), Position: pos})
}

// Extend appends every diagnostic of another bag.
func (b *Bag) Extend(other *Bag) {
	if other == nil {
		return
	}
	b.items = append(b.items, other.items...)
}

// HasErrors reports whether at least one error was collected.
func (b *Bag) HasErrors() bool {
	for _, d := range b.items {
		if d.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Len returns the number of collected diagnostics.
func (b *Bag) Len() int { return len(b.items) }

// Sorted returns the diagnostics ordered by file, line, column, code then
// message, so that two runs over the same inputs always report the same order.
func (b *Bag) Sorted() []Diagnostic {
	out := make([]Diagnostic, len(b.items))
	copy(out, b.items)
	sort.SliceStable(out, func(i, j int) bool {
		a, c := out[i], out[j]
		switch {
		case a.Position.File != c.Position.File:
			return a.Position.File < c.Position.File
		case a.Position.Line != c.Position.Line:
			return a.Position.Line < c.Position.Line
		case a.Position.Column != c.Position.Column:
			return a.Position.Column < c.Position.Column
		case a.Code != c.Code:
			return a.Code < c.Code
		default:
			return a.Message < c.Message
		}
	})
	return out
}

// WriteText renders every diagnostic for humans.
func (b *Bag) WriteText(w io.Writer) error {
	for _, d := range b.Sorted() {
		if _, err := fmt.Fprintln(w, d.String()); err != nil {
			return err
		}
	}
	return nil
}

// WriteJSON renders every diagnostic as a machine readable document.
func (b *Bag) WriteJSON(w io.Writer) error {
	items := b.Sorted()
	if items == nil {
		items = []Diagnostic{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(struct {
		Diagnostics []Diagnostic `json:"diagnostics"`
	}{items})
}

// Err returns a non nil error when the bag holds at least one error.
func (b *Bag) Err() error {
	if !b.HasErrors() {
		return nil
	}
	return Error{Diagnostics: b.Sorted()}
}

// Error is the error type carrying a set of diagnostics.
type Error struct {
	Diagnostics []Diagnostic
}

// Error implements the error interface.
func (e Error) Error() string {
	n := 0
	for _, d := range e.Diagnostics {
		if d.Severity == SeverityError {
			n++
		}
	}
	if n <= 1 {
		return e.first()
	}
	return fmt.Sprintf("%d errors, first: %s", n, e.first())
}

func (e Error) first() string {
	for _, d := range e.Diagnostics {
		if d.Severity == SeverityError {
			return d.String()
		}
	}
	return "no diagnostic"
}
