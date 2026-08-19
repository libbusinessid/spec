package artifact

import "fmt"

// ErrorKind classifies a bundle rejection exactly as the engine contract does.
type ErrorKind string

// The two normative bundle rejection kinds.
const (
	// ErrIncompatible is reported when the bundle is well formed but requires a
	// format version or a capability this runtime does not implement.
	ErrIncompatible ErrorKind = "incompatible_ruleset"
	// ErrInvalid is reported when the bundle is malformed, out of limits or
	// violates an IR invariant.
	ErrInvalid ErrorKind = "invalid_ruleset"
)

// Error is the typed rejection of a rule bundle.
type Error struct {
	Kind   ErrorKind
	Detail string
}

// Error implements the error interface.
func (e *Error) Error() string { return string(e.Kind) + ": " + e.Detail }

func invalidf(format string, args ...any) *Error {
	return &Error{Kind: ErrInvalid, Detail: fmt.Sprintf(format, args...)}
}

func incompatiblef(format string, args ...any) *Error {
	return &Error{Kind: ErrIncompatible, Detail: fmt.Sprintf(format, args...)}
}
