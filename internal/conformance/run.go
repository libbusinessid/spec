package conformance

import (
	"fmt"
	"strings"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/reference"
)

// Failure describes one conformance case the interpreter did not satisfy.
type Failure struct {
	CaseID  string
	Field   string
	Got     string
	Want    string
	Message string
}

// String renders the failure for a human.
func (f Failure) String() string {
	if f.Message != "" {
		return fmt.Sprintf("%s: %s", f.CaseID, f.Message)
	}
	return fmt.Sprintf("%s: %s got %q, want %q", f.CaseID, f.Field, f.Got, f.Want)
}

// Report summarizes a conformance run.
type Report struct {
	Total    int
	Passed   int
	Failures []Failure
}

// Run executes every case of the bundle against the reference interpreter.
//
// The interpreter only verifies the reviewed expectations: it never produces
// them.
func Run(engine *reference.Engine, bundle *conformancev1.ConformanceBundle) Report {
	report := Report{Total: len(bundle.GetCases())}
	for _, c := range bundle.GetCases() {
		failures := runCase(engine, c)
		if len(failures) == 0 {
			report.Passed++
			continue
		}
		report.Failures = append(report.Failures, failures...)
	}
	return report
}

func runCase(engine *reference.Engine, c *conformancev1.ConformanceCase) []Failure {
	if c.GetOperation() == conformancev1.Operation_OPERATION_LOAD_RULESET {
		return runLoadCase(c)
	}
	in := reference.Input{Kind: c.GetKind(), Value: c.GetInput()}
	if c.CountryCode != nil {
		country := c.GetCountryCode()
		in.CountryCode = &country
	}
	opts := reference.Options{Profile: reference.Profile(c.GetProfile())}
	if c.GetOperation() == conformancev1.Operation_OPERATION_CANONICALIZE {
		got, err := engine.Canonicalize(in, opts)
		if err != nil {
			return []Failure{{CaseID: c.GetId(), Message: "engine error: " + err.Error()}}
		}
		return compareCanonicalization(c, got)
	}
	var (
		got reference.ValidationReport
		err error
	)
	switch c.GetOperation() {
	case conformancev1.Operation_OPERATION_VALIDATE_FORMAT:
		got, err = engine.ValidateFormat(in, opts)
	case conformancev1.Operation_OPERATION_VALIDATE_CHECKSUM:
		got, err = engine.ValidateChecksum(in, opts)
	default:
		got, err = engine.Validate(in, opts)
	}
	if err != nil {
		return []Failure{{CaseID: c.GetId(), Message: "engine error: " + err.Error()}}
	}
	return compareReport(c, got)
}

func runLoadCase(c *conformancev1.ConformanceCase) []Failure {
	_, err := artifact.LoadRuleset(c.GetRulesPayload())
	if err == nil {
		return []Failure{{CaseID: c.GetId(), Field: "engineError", Got: "none", Want: c.GetExpectedEngineError()}}
	}
	var kind string
	if typed, ok := err.(*artifact.Error); ok { //nolint:errorlint // the loader returns this exact type
		kind = string(typed.Kind)
	} else {
		kind = "unknown"
	}
	if kind != c.GetExpectedEngineError() {
		return []Failure{{
			CaseID: c.GetId(), Field: "engineError",
			Got: kind + " (" + err.Error() + ")", Want: c.GetExpectedEngineError(),
		}}
	}
	return nil
}

func compareCanonicalization(c *conformancev1.ConformanceCase, got reference.CanonicalizationResult) []Failure {
	want := c.GetExpected().GetCanonicalization()
	if want == nil {
		return []Failure{{CaseID: c.GetId(), Message: "the case does not carry a canonicalization expectation"}}
	}
	var out []Failure
	add := func(field, g, w string) {
		if g != w {
			out = append(out, Failure{CaseID: c.GetId(), Field: field, Got: g, Want: w})
		}
	}
	add("kind", got.Kind, want.GetKind())
	add("inputValue", got.InputValue, want.GetInputValue())
	add("canonicalValue", got.CanonicalValue, want.GetCanonicalValue())
	add("countryCode", optional(got.CountryCode), optionalProto(want.CountryCode))
	add("profile", string(got.Profile), want.GetProfile())
	add("rulesVersion", got.RulesVersion, want.GetRulesVersion())
	add("formatVersion", fmt.Sprint(got.FormatVersion), fmt.Sprint(want.GetFormatVersion()))
	add("status", string(got.Status), statusName(want.GetStatus()))
	add("reasonCode", reasonName(got.ReasonCode), reasonName(want.GetReasonCode()))
	add("messageKey", optional(got.MessageKey), optionalProto(want.MessageKey))
	return out
}

func compareReport(c *conformancev1.ConformanceCase, got reference.ValidationReport) []Failure {
	want := c.GetExpected().GetValidationReport()
	if want == nil {
		return []Failure{{CaseID: c.GetId(), Message: "the case does not carry a validation expectation"}}
	}
	var out []Failure
	add := func(field, g, w string) {
		if g != w {
			out = append(out, Failure{CaseID: c.GetId(), Field: field, Got: g, Want: w})
		}
	}
	add("kind", got.Kind, want.GetKind())
	add("inputValue", got.InputValue, want.GetInputValue())
	add("canonicalValue", got.CanonicalValue, want.GetCanonicalValue())
	add("countryCode", optional(got.CountryCode), optionalProto(want.CountryCode))
	add("profile", string(got.Profile), want.GetProfile())
	add("rulesVersion", got.RulesVersion, want.GetRulesVersion())
	add("formatVersion", fmt.Sprint(got.FormatVersion), fmt.Sprint(want.GetFormatVersion()))
	out = append(out, compareStep(c.GetId(), "format", got.Format, want.GetFormat())...)
	out = append(out, compareStep(c.GetId(), "checksum", got.Checksum, want.GetChecksum())...)
	return out
}

func compareStep(id, label string, got reference.StepResult, want *conformancev1.ExpectedStep) []Failure {
	var out []Failure
	add := func(field, g, w string) {
		if g != w {
			out = append(out, Failure{CaseID: id, Field: label + "." + field, Got: g, Want: w})
		}
	}
	add("level", string(got.Level), levelName(want.GetLevel()))
	add("status", string(got.Status), statusName(want.GetStatus()))
	add("reasonCode", reasonName(got.ReasonCode), reasonName(want.GetReasonCode()))
	// A message key is only compared when the expectation declares one.
	if want.MessageKey != nil {
		add("messageKey", optional(got.MessageKey), want.GetMessageKey())
	}
	return out
}

func optional(s *string) string {
	if s == nil {
		return "<absent>"
	}
	return *s
}

func optionalProto(s *string) string {
	if s == nil {
		return "<absent>"
	}
	return *s
}

func statusName(s conformancev1.StepStatus) string {
	return strings.ToLower(strings.TrimPrefix(conformancev1.StepStatus_name[int32(s)], "STEP_STATUS_"))
}

func levelName(l conformancev1.ValidationLevel) string {
	return strings.ToLower(strings.TrimPrefix(conformancev1.ValidationLevel_name[int32(l)], "VALIDATION_LEVEL_"))
}

func reasonName(r irv1.ReasonCode) string {
	return strings.ToLower(strings.TrimPrefix(irv1.ReasonCode_name[int32(r)], "REASON_CODE_"))
}
