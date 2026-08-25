// Package limits holds the normative structural and arithmetic limits of the
// LibEntID IR v1. Every value here is normative and reproduced in
// docs/ir.md. Engines may raise an internal limit, never lower it.
package limits

// Structural limits of the rule bundle (spec.md 7.5).
const (
	// MaxBundleBytes is the maximum size of a serialized rule bundle.
	MaxBundleBytes = 16 * 1024 * 1024
	// MaxIdentifiers is the maximum number of identifier definitions.
	MaxIdentifiers = 10_000
	// MaxTotalNodes is the maximum number of nodes across all programs.
	MaxTotalNodes = 500_000
	// MaxNodesPerProgram is the maximum number of nodes of a single program.
	MaxNodesPerProgram = 4_096
	// MaxCallDepth is the maximum static program call depth.
	MaxCallDepth = 32
	// MaxConstantBytes is the maximum size of a constant string in UTF-8 bytes.
	MaxConstantBytes = 4_096
	// MaxInputBytes is the maximum accepted raw user input in UTF-8 bytes.
	MaxInputBytes = 1_024
	// MaxStepsPerValidation is the evaluation budget of a public operation.
	MaxStepsPerValidation = 100_000
	// CodePointsPerStep is the number of produced code points charged as one
	// evaluation step. Together with MaxStepsPerValidation it bounds the total
	// number of code points a single operation can materialize, and therefore
	// the memory a hostile bundle can make an engine allocate.
	CodePointsPerStep = 64
	// MaxCapturesPerFormat is the maximum number of captures of a format rule.
	MaxCapturesPerFormat = 128
)

// Arithmetic limits (spec.md 7.5).
const (
	// MinModulus is the smallest accepted modulus or complement base.
	MinModulus = 2
	// MaxModulus is the largest accepted modulus or complement base.
	MaxModulus = 1_000_000_000
	// MaxWeightMagnitude is the largest accepted absolute value of a weight.
	MaxWeightMagnitude = 1_000_000
	// MinWeights is the smallest accepted number of weights.
	MinWeights = 1
	// MaxWeights is the largest accepted number of weights.
	MaxWeights = 256

	// MaxAlphabetRunes bounds the alphabet of CHAR_MAPPING_CUSTOM_ALPHABET.
	MaxAlphabetRunes = 256
	// MinRemainderValues is the smallest accepted remainder table size.
	MinRemainderValues = 1
	// MaxRemainderValues is the largest accepted remainder table size.
	MaxRemainderValues = 1_000_000
	// MaxIndex is the largest accepted index or slice bound.
	MaxIndex = 4_096

	// MinConstant and MaxConstant bound the literal COMPARE_CONSTANT compares
	// against. The range matches what a checked integer expression can produce,
	// so a comparison can never be written against a value no expression could
	// reach.
	// MaxRulesVersionBytes bounds the version string, which engines carry into
	// generated sources, manifests and logs.
	MaxRulesVersionBytes = 64

	MinConstant = -1_000_000_000
	MaxConstant = 1_000_000_000
	// MinConcatOperands is the smallest accepted number of concat operands.
	MinConcatOperands = 1
	// MaxConcatOperands is the largest accepted number of concat operands.
	MaxConcatOperands = 256
	// MaxDigitsToIntegerLength is the largest provable digit length that still
	// fits in a signed 64 bit integer.
	MaxDigitsToIntegerLength = 18
)

// Conformance bundle limits (spec.md 8.4).
const (
	// MaxConformanceBundleBytes is the maximum size of a conformance bundle.
	MaxConformanceBundleBytes = 64 * 1024 * 1024
	// MaxConformanceCases is the maximum number of conformance cases.
	MaxConformanceCases = 1_000_000
	// MaxDescriptionBytes is the maximum size of a case description.
	MaxDescriptionBytes = 4_096
)

// Identifier and dispatcher token limits (spec.md 6.11).
const (
	// MaxKindLength is the maximum length of a canonical kind or kind alias.
	MaxKindLength = 64
	// MinPrefixLength is the minimum length of a dispatch prefix.
	MinPrefixLength = 1
	// MaxPrefixLength is the maximum length of a dispatch prefix.
	MaxPrefixLength = 8
)
