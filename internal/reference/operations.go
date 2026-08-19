package reference

import (
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/limits"
)

// profileOf resolves the effective profile of an operation.
func profileOf(opts Options, definition *irv1.IdentifierDefinition) Profile {
	if opts.Profile != "" {
		return opts.Profile
	}
	if definition != nil && definition.GetDefaultProfile() != "" {
		return Profile(definition.GetDefaultProfile())
	}
	return ProfileCompatible
}

// Canonicalize runs only the input limit, the dispatch and both
// canonicalization phases.
func (e *Engine) Canonicalize(in Input, opts Options) (CanonicalizationResult, error) {
	profile := opts.Profile
	if profile == "" {
		profile = ProfileCompatible
	}
	result := CanonicalizationResult{
		Kind:           lowerASCII(trimASCII(in.Kind)),
		InputValue:     in.Value,
		CanonicalValue: in.Value,
		CountryCode:    in.CountryCode,
		Profile:        profile,
		RulesVersion:   e.rules.RulesVersion(),
		FormatVersion:  e.rules.FormatVersion(),
	}
	if len(in.Value) > limits.MaxInputBytes {
		result.Status = StatusUnsupported
		result.ReasonCode = irv1.ReasonCode_REASON_CODE_INPUT_TOO_LONG
		return result, nil
	}
	m := &machine{rules: e.rules}
	outcome, err := e.dispatch(m, in, profile)
	if err != nil {
		return CanonicalizationResult{}, err
	}
	result.Kind = outcome.kind
	result.CanonicalValue = outcome.canonicalValue
	result.CountryCode = outcome.countryCode
	if outcome.definition != nil {
		result.Profile = profileOf(opts, outcome.definition)
	}
	switch outcome.reason {
	case irv1.ReasonCode_REASON_CODE_OK:
		result.Status = StatusValid
		result.ReasonCode = irv1.ReasonCode_REASON_CODE_OK
	case irv1.ReasonCode_REASON_CODE_COUNTRY_MISMATCH:
		result.Status = StatusInvalid
		result.ReasonCode = irv1.ReasonCode_REASON_CODE_COUNTRY_MISMATCH
	default:
		result.Status = StatusUnsupported
		result.ReasonCode = outcome.reason
	}
	return result, nil
}

// operation selects how far the pipeline runs.
type operation int

const (
	opValidateFormat operation = iota
	opValidate
)

// Validate runs dispatch, canonicalization, format then checksum.
func (e *Engine) Validate(in Input, opts Options) (ValidationReport, error) {
	return e.run(in, opts, opValidate)
}

// ValidateFormat runs dispatch, canonicalization and format only. After a valid
// format the checksum step is `not_run`/`not_requested`.
func (e *Engine) ValidateFormat(in Input, opts Options) (ValidationReport, error) {
	return e.run(in, opts, opValidateFormat)
}

// ValidateChecksum returns exactly the same report as Validate: the format
// always acts as a guard and the separate entry point exists only for API
// readability.
func (e *Engine) ValidateChecksum(in Input, opts Options) (ValidationReport, error) {
	return e.run(in, opts, opValidate)
}

//nolint:funlen // the normative pipeline is one linear sequence.
func (e *Engine) run(in Input, opts Options, op operation) (ValidationReport, error) {
	profile := opts.Profile
	if profile == "" {
		profile = ProfileCompatible
	}
	report := ValidationReport{
		Kind:           lowerASCII(trimASCII(in.Kind)),
		InputValue:     in.Value,
		CanonicalValue: in.Value,
		CountryCode:    in.CountryCode,
		Profile:        profile,
		RulesVersion:   e.rules.RulesVersion(),
		FormatVersion:  e.rules.FormatVersion(),
	}
	// 1: the raw input limit is a safety bound, never a business proof.
	if len(in.Value) > limits.MaxInputBytes {
		report.Format = StepResult{
			Level:      LevelFormat,
			Status:     StatusUnsupported,
			ReasonCode: irv1.ReasonCode_REASON_CODE_INPUT_TOO_LONG,
		}
		report.Checksum = notRun(irv1.ReasonCode_REASON_CODE_NOT_RUN_FORMAT_UNSUPPORTED)
		return report, nil
	}

	m := &machine{rules: e.rules}
	outcome, err := e.dispatch(m, in, profile)
	if err != nil {
		return ValidationReport{}, err
	}
	report.Kind = outcome.kind
	report.CanonicalValue = outcome.canonicalValue
	report.CountryCode = outcome.countryCode

	// 3 and 4: dispatch failures.
	switch {
	case outcome.reason == irv1.ReasonCode_REASON_CODE_COUNTRY_MISMATCH:
		report.Format = StepResult{
			Level:      LevelFormat,
			Status:     StatusInvalid,
			ReasonCode: irv1.ReasonCode_REASON_CODE_COUNTRY_MISMATCH,
		}
		report.Checksum = notRun(irv1.ReasonCode_REASON_CODE_NOT_RUN_FORMAT_INVALID)
		return report, nil
	case outcome.reason != irv1.ReasonCode_REASON_CODE_OK:
		report.Format = StepResult{Level: LevelFormat, Status: StatusUnsupported, ReasonCode: outcome.reason}
		report.Checksum = notRun(irv1.ReasonCode_REASON_CODE_NOT_RUN_FORMAT_UNSUPPORTED)
		return report, nil
	}

	definition := outcome.definition
	report.Profile = profileOf(opts, definition)
	base := &frame{
		value:   outcome.canonicalValue,
		country: definition.CountryCode,
		profile: report.Profile,
		target:  outcome.target,
	}

	// 5: the format runs on the canonical value.
	formatProgram, err := m.program(definition.GetFormatProgram())
	if err != nil {
		return ValidationReport{}, err
	}
	formatOut, err := m.RunFormat(formatProgram, base)
	if err != nil {
		return ValidationReport{}, err
	}
	if !formatOut.ok {
		report.Format = StepResult{
			Level:      LevelFormat,
			Status:     StatusInvalid,
			ReasonCode: formatOut.reason,
			MessageKey: formatOut.messageKey,
		}
		report.Checksum = notRun(irv1.ReasonCode_REASON_CODE_NOT_RUN_FORMAT_INVALID)
		return report, nil
	}
	report.Format = StepResult{
		Level:      LevelFormat,
		Status:     StatusValid,
		ReasonCode: irv1.ReasonCode_REASON_CODE_OK,
	}
	if op == opValidateFormat {
		report.Checksum = notRun(irv1.ReasonCode_REASON_CODE_NOT_REQUESTED)
		return report, nil
	}

	// 8: no checksum program at all.
	if definition.ChecksumProgram == nil {
		report.Checksum = StepResult{
			Level:      LevelChecksum,
			Status:     StatusUnsupported,
			ReasonCode: definition.GetAbsentChecksumReason(),
		}
		return report, nil
	}

	// 9: the checksum only ever runs on a validated format.
	checksumProgram, err := m.program(definition.GetChecksumProgram())
	if err != nil {
		return ValidationReport{}, err
	}
	checksumOut, err := m.RunChecksum(checksumProgram, base)
	if err != nil {
		return ValidationReport{}, err
	}
	if checksumOut.notApplicable {
		return ValidationReport{}, enginef("a checksum program returned a non applicable branch")
	}
	report.Checksum = StepResult{
		Level:      LevelChecksum,
		Status:     checksumOut.status,
		ReasonCode: checksumOut.reason,
		MessageKey: checksumOut.messageKey,
	}
	return report, nil
}

func notRun(reason irv1.ReasonCode) StepResult {
	return StepResult{Level: LevelChecksum, Status: StatusNotRun, ReasonCode: reason}
}

// RegistryLookup is the registry entry point. V1 ships no provider, so the
// result is always `registry_not_configured` when none is supplied.
func (e *Engine) RegistryLookup(in Input, provider RegistryProvider, opts Options) (RegistryResult, error) {
	report, err := e.Validate(in, opts)
	if err != nil {
		return RegistryResult{}, err
	}
	if provider == nil {
		return RegistryResult{
			Status:         RegistryUnsupported,
			CanonicalValue: report.CanonicalValue,
			ReasonCode:     irv1.ReasonCode_REASON_CODE_REGISTRY_NOT_CONFIGURED,
		}, nil
	}
	if !provider.Supports(report.Kind, report.CountryCode) {
		return RegistryResult{
			Status:         RegistryUnsupported,
			CanonicalValue: report.CanonicalValue,
			ReasonCode:     irv1.ReasonCode_REASON_CODE_REGISTRY_NOT_CONFIGURED,
		}, nil
	}
	return provider.Lookup(report.CanonicalValue, in)
}
