package artifact

import (
	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
)

// Ruleset is a fully validated bundle, indexed for execution.
type Ruleset struct {
	// Bundle is the decoded message. It is never mutated after validation.
	Bundle *irv1.RuleBundle
	// ProgramByID indexes the programs.
	ProgramByID map[uint32]*irv1.Program
	// DefinitionByID indexes the identifier definitions.
	DefinitionByID map[uint32]*irv1.IdentifierDefinition
	// DispatcherByKind indexes the dispatchers by canonical kind and by alias.
	DispatcherByKind map[string]*irv1.IdentifierDispatcher
}

// RulesVersion returns the business version of the bundle.
func (r *Ruleset) RulesVersion() string { return r.Bundle.GetRulesVersion() }

// FormatVersion returns the structural version of the bundle.
func (r *Ruleset) FormatVersion() uint32 { return r.Bundle.GetFormatVersion() }

// RequiredFeatures returns the capability IDs required by the bundle.
func (r *Ruleset) RequiredFeatures() []uint32 {
	out := make([]uint32, len(r.Bundle.GetRequiredFeatureIds()))
	copy(out, r.Bundle.GetRequiredFeatureIds())
	return out
}
