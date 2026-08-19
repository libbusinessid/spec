// Package version exposes the compiler identity stamped into every manifest.
package version

// Compiler is the semantic version of businessidc. It is bumped whenever the
// emitted bytes, the manifest or a generated document can change.
const Compiler = "1.0.0"

// Name is the tool name reported by `businessidc version`.
const Name = "businessidc"
