package typecheck

import (
	"sort"

	"github.com/entid-org/spec/internal/features"
)

// argKind classifies one surface argument of a language function.
type argKind int

const (
	// argOperand is a nested expression that becomes an IR operand.
	argOperand argKind = iota
	// argVariadicOperand repeats argOperand until the end of the call.
	argVariadicOperand
	// argInt is an integer literal bound to an unsigned parameter.
	argInt
	// argModulus is an integer literal bound to the modulus parameter.
	argModulus
	// argConstant is a signed integer literal an outcome is compared against.
	argConstant
	// argString is a string literal bound to a text parameter.
	argString
	// argOptionalString is a trailing optional string literal.
	argOptionalString
	// argCharList is a list of single code point string literals folded into
	// the text parameter as a character set.
	argCharList
	// argIntList is a list of integer literals.
	argIntList
	// argStringList is a list of string literals.
	argStringList
	// argEnum is a string literal naming an enum value.
	argEnum
	// argChecksumRef is a checksum symbol reference.
	argChecksumRef
	// argReasonCode is a string literal naming a reason code.
	argReasonCode
)

// argSlot binds one surface argument to an IR parameter.
type argSlot struct {
	kind  argKind
	param features.Param
}

// signature is the surface calling convention of one catalog operation.
type signature struct {
	op   features.Op
	args []argSlot
}

// signatures maps every surface function name of the language to its catalog
// operation and calling convention. It is built once from the frozen catalog.
var signatures = buildSignatures()

func register(table map[string]signature, category features.Category, code int32, args ...argSlot) {
	op, ok := features.LookupOp(category, code)
	if !ok {
		panic("typecheck: unknown catalog operation")
	}
	name := op.HCLName()
	if name == "" {
		panic("typecheck: catalog operation has no surface name")
	}
	if _, dup := table[name]; dup {
		panic("typecheck: duplicate surface name " + name)
	}
	table[name] = signature{op: op, args: args}
}

// SurfaceNames returns every function name of the language, sorted.
func SurfaceNames() []string {
	out := make([]string, 0, len(signatures))
	for name := range signatures {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func operand() argSlot  { return argSlot{kind: argOperand} }
func variadic() argSlot { return argSlot{kind: argVariadicOperand} }
func intArg(p features.Param) argSlot {
	return argSlot{kind: argInt, param: p}
}
func modulusArg() argSlot  { return argSlot{kind: argModulus, param: features.ParamModulus} }
func constantArg() argSlot { return argSlot{kind: argConstant, param: features.ParamConstant} }
func strArg(p features.Param) argSlot {
	return argSlot{kind: argString, param: p}
}
func optStrArg(p features.Param) argSlot {
	return argSlot{kind: argOptionalString, param: p}
}
func charListArg() argSlot { return argSlot{kind: argCharList, param: features.ParamText} }
func intListArg(p features.Param) argSlot {
	return argSlot{kind: argIntList, param: p}
}
func strListArg(p features.Param) argSlot {
	return argSlot{kind: argStringList, param: p}
}
func enumArg(p features.Param) argSlot {
	return argSlot{kind: argEnum, param: p}
}
func checksumRefArg() argSlot { return argSlot{kind: argChecksumRef} }
func reasonArg() argSlot      { return argSlot{kind: argReasonCode, param: features.ParamReasonCode} }
