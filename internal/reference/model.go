// Package reference implements the internal reference interpreter of the
// EntID IR.
//
// Its only purpose is to verify the specification, the compiled bundles and the
// reviewed conformance expectations. It is deliberately written for readability
// and defensive checking, and it is never published as an engine: each
// production engine stays an independent, idiomatic implementation so that a
// defect is unlikely to be reproduced identically.
package reference

import irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"

// StepStatus is the outcome of one validation step.
type StepStatus string

// The normative step statuses.
const (
	StatusValid       StepStatus = "valid"
	StatusInvalid     StepStatus = "invalid"
	StatusUnsupported StepStatus = "unsupported"
	StatusNotRun      StepStatus = "not_run"
)

// Level names the step a result belongs to.
type Level string

// The normative validation levels.
const (
	LevelFormat   Level = "format"
	LevelChecksum Level = "checksum"
	LevelRegistry Level = "registry"
)

// Profile is a validation profile.
type Profile string

// The normative validation profiles.
const (
	ProfileCompatible    Profile = "compatible"
	ProfileStrictCurrent Profile = "strict_current"
)

// Input is a value submitted for canonicalization or validation.
type Input struct {
	Kind        string
	Value       string
	CountryCode *string
}

// Options carries the caller supplied validation options.
type Options struct {
	Profile Profile
}

// StepResult is the outcome of one step.
type StepResult struct {
	Level      Level
	Status     StepStatus
	ReasonCode irv1.ReasonCode
	MessageKey *string
}

// ValidationReport is the immutable result of a validation operation.
type ValidationReport struct {
	Kind           string
	InputValue     string
	CanonicalValue string
	CountryCode    *string
	Profile        Profile
	RulesVersion   string
	FormatVersion  uint32
	Format         StepResult
	Checksum       StepResult
}

// CanonicalizationResult is the immutable result of a canonicalization.
type CanonicalizationResult struct {
	Kind           string
	InputValue     string
	CanonicalValue string
	CountryCode    *string
	Profile        Profile
	RulesVersion   string
	FormatVersion  uint32
	Status         StepStatus
	ReasonCode     irv1.ReasonCode
	MessageKey     *string
}

// RegistryStatus is the outcome of a registry lookup.
type RegistryStatus string

// The registry statuses of the V1 interface.
const (
	RegistryFound                  RegistryStatus = "found"
	RegistryNotFound               RegistryStatus = "not_found"
	RegistryInactive               RegistryStatus = "inactive"
	RegistryUnsupported            RegistryStatus = "unsupported"
	RegistryTemporarilyUnavailable RegistryStatus = "temporarily_unavailable"
)

// RegistryResult is the outcome of a registry lookup. V1 ships no provider.
type RegistryResult struct {
	Status         RegistryStatus
	ProviderID     string
	CheckedAt      string
	CanonicalValue string
	ReasonCode     irv1.ReasonCode
	Metadata       map[string]string
}

// RegistryProvider is the registry abstraction. No concrete provider exists in
// V1 and no engine performs a network call during validation.
type RegistryProvider interface {
	// Supports reports whether the provider covers a kind and country.
	Supports(kind string, countryCode *string) bool
	// Lookup resolves a canonical identifier, or fails technically.
	Lookup(canonical string, input Input) (RegistryResult, error)
}
