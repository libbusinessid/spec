package conformance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/conformance"
	"github.com/entid-org/spec/internal/reference"
)

func pilotEngine(t *testing.T) *reference.Engine {
	t.Helper()
	result, bag := artifact.CompileRules(filepath.Join(repoRoot(t), "rules"),
		artifact.CompileOptions{RulesVersion: "2026.08.0", Optimize: true})
	if bag.HasErrors() {
		t.Fatalf("the rules do not compile: %v", bag.Sorted())
	}
	return reference.NewEngineFromRuleset(result.Ruleset)
}

func validationCase(id string, mutate func(*conformancev1.ExpectedValidationReport)) *conformancev1.ConformanceCase {
	expected := &conformancev1.ExpectedValidationReport{
		Kind: "siren", InputValue: "012345674", CanonicalValue: "012345674",
		CountryCode: strPointer("FR"), Profile: "compatible", RulesVersion: "2026.08.0", FormatVersion: 1,
		Format: &conformancev1.ExpectedStep{
			Level:      conformancev1.ValidationLevel_VALIDATION_LEVEL_FORMAT,
			Status:     conformancev1.StepStatus_STEP_STATUS_VALID,
			ReasonCode: irv1.ReasonCode_REASON_CODE_OK,
		},
		Checksum: &conformancev1.ExpectedStep{
			Level:      conformancev1.ValidationLevel_VALIDATION_LEVEL_CHECKSUM,
			Status:     conformancev1.StepStatus_STEP_STATUS_VALID,
			ReasonCode: irv1.ReasonCode_REASON_CODE_OK,
		},
	}
	if mutate != nil {
		mutate(expected)
	}
	return &conformancev1.ConformanceCase{
		Id: id, Kind: "siren", Input: "012345674", Profile: "compatible",
		Operation: conformancev1.Operation_OPERATION_VALIDATE,
		Expected: &conformancev1.ExpectedOutcome{Value: &conformancev1.ExpectedOutcome_ValidationReport{
			ValidationReport: expected,
		}},
	}
}

func strPointer(v string) *string { return &v }

func TestRunReportsEveryMismatchedField(t *testing.T) {
	engine := pilotEngine(t)
	bundle := &conformancev1.ConformanceBundle{Cases: []*conformancev1.ConformanceCase{
		validationCase("wrong-kind", func(e *conformancev1.ExpectedValidationReport) { e.Kind = "vat" }),
		validationCase("wrong-canonical", func(e *conformancev1.ExpectedValidationReport) { e.CanonicalValue = "x" }),
		validationCase("wrong-country", func(e *conformancev1.ExpectedValidationReport) { e.CountryCode = nil }),
		validationCase("wrong-profile", func(e *conformancev1.ExpectedValidationReport) { e.Profile = "strict_current" }),
		validationCase("wrong-rules-version", func(e *conformancev1.ExpectedValidationReport) { e.RulesVersion = "2020.01.0" }),
		validationCase("wrong-format-version", func(e *conformancev1.ExpectedValidationReport) { e.FormatVersion = 9 }),
		validationCase("wrong-input", func(e *conformancev1.ExpectedValidationReport) { e.InputValue = "x" }),
		validationCase("wrong-message-key", func(e *conformancev1.ExpectedValidationReport) {
			key := "nope"
			e.Format.MessageKey = &key
		}),
		validationCase("wrong-level", func(e *conformancev1.ExpectedValidationReport) {
			e.Checksum.Level = conformancev1.ValidationLevel_VALIDATION_LEVEL_REGISTRY
		}),
	}}
	report := conformance.Run(engine, bundle)
	if report.Passed != 0 || len(report.Failures) < 9 {
		t.Fatalf("expected every case to fail: %+v", report)
	}
	fields := map[string]bool{}
	for _, f := range report.Failures {
		fields[f.Field] = true
		if !strings.Contains(f.String(), f.CaseID) {
			t.Fatalf("the failure must name the case: %q", f.String())
		}
	}
	for _, want := range []string{
		"kind", "canonicalValue", "countryCode", "profile", "rulesVersion",
		"formatVersion", "inputValue", "format.messageKey", "checksum.level",
	} {
		if !fields[want] {
			t.Fatalf("the field %q was not compared", want)
		}
	}
}

func TestRunComparesCanonicalizationCases(t *testing.T) {
	engine := pilotEngine(t)
	expected := &conformancev1.ExpectedCanonicalization{
		Kind: "siren", InputValue: "012345674", CanonicalValue: "nope",
		Profile: "compatible", RulesVersion: "2026.08.0", FormatVersion: 1,
		Status: conformancev1.StepStatus_STEP_STATUS_VALID, ReasonCode: irv1.ReasonCode_REASON_CODE_OK,
	}
	bundle := &conformancev1.ConformanceBundle{Cases: []*conformancev1.ConformanceCase{{
		Id: "canon", Kind: "siren", Input: "012345674", Profile: "compatible",
		Operation: conformancev1.Operation_OPERATION_CANONICALIZE,
		Expected: &conformancev1.ExpectedOutcome{Value: &conformancev1.ExpectedOutcome_Canonicalization{
			Canonicalization: expected,
		}},
	}}}
	report := conformance.Run(engine, bundle)
	if len(report.Failures) == 0 {
		t.Fatal("the canonical value mismatch must be reported")
	}
}

func TestRunRejectsAMissingExpectation(t *testing.T) {
	engine := pilotEngine(t)
	bundle := &conformancev1.ConformanceBundle{Cases: []*conformancev1.ConformanceCase{
		{
			Id: "no-expectation", Kind: "siren", Input: "012345674", Profile: "compatible",
			Operation: conformancev1.Operation_OPERATION_VALIDATE,
		},
		{
			Id: "no-canonicalization", Kind: "siren", Input: "012345674", Profile: "compatible",
			Operation: conformancev1.Operation_OPERATION_CANONICALIZE,
		},
	}}
	report := conformance.Run(engine, bundle)
	if len(report.Failures) != 2 {
		t.Fatalf("expected two failures, got %+v", report.Failures)
	}
	for _, f := range report.Failures {
		if !strings.Contains(f.String(), "expectation") {
			t.Fatalf("unexpected failure %q", f.String())
		}
	}
}

func TestRunHandlesEveryOperation(t *testing.T) {
	engine := pilotEngine(t)
	for _, op := range []conformancev1.Operation{
		conformancev1.Operation_OPERATION_VALIDATE,
		conformancev1.Operation_OPERATION_VALIDATE_CHECKSUM,
	} {
		bundle := &conformancev1.ConformanceBundle{Cases: []*conformancev1.ConformanceCase{
			validationCase("op", nil),
		}}
		bundle.Cases[0].Operation = op
		report := conformance.Run(engine, bundle)
		if report.Passed != 1 {
			t.Fatalf("operation %v failed: %+v", op, report.Failures)
		}
	}
	formatCase := validationCase("format", func(e *conformancev1.ExpectedValidationReport) {
		e.Checksum.Status = conformancev1.StepStatus_STEP_STATUS_NOT_RUN
		e.Checksum.ReasonCode = irv1.ReasonCode_REASON_CODE_NOT_REQUESTED
	})
	formatCase.Operation = conformancev1.Operation_OPERATION_VALIDATE_FORMAT
	report := conformance.Run(engine, &conformancev1.ConformanceBundle{
		Cases: []*conformancev1.ConformanceCase{formatCase},
	})
	if report.Passed != 1 {
		t.Fatalf("validate_format failed: %+v", report.Failures)
	}
}

func TestRunLoadRulesetCases(t *testing.T) {
	engine := pilotEngine(t)
	valid, err := os.ReadFile(filepath.Join(repoRoot(t), "testdata", "bundles", "minimal_valid.binpb"))
	if err != nil {
		t.Fatal(err)
	}
	truncated, err := os.ReadFile(filepath.Join(repoRoot(t), "testdata", "bundles", "truncated.binpb"))
	if err != nil {
		t.Fatal(err)
	}
	expectedInvalid := "invalid_ruleset"
	expectedIncompatible := "incompatible_ruleset"
	bundle := &conformancev1.ConformanceBundle{Cases: []*conformancev1.ConformanceCase{
		{
			Id: "accepted", Operation: conformancev1.Operation_OPERATION_LOAD_RULESET,
			RulesPayload: valid, ExpectedEngineError: &expectedInvalid,
		},
		{
			Id: "wrong-kind", Operation: conformancev1.Operation_OPERATION_LOAD_RULESET,
			RulesPayload: truncated, ExpectedEngineError: &expectedIncompatible,
		},
		{
			Id: "correct", Operation: conformancev1.Operation_OPERATION_LOAD_RULESET,
			RulesPayload: truncated, ExpectedEngineError: &expectedInvalid,
		},
	}}
	report := conformance.Run(engine, bundle)
	if report.Passed != 1 || len(report.Failures) != 2 {
		t.Fatalf("unexpected report %+v", report)
	}
	if report.Failures[0].Got != "none" {
		t.Fatalf("an accepted bundle must be reported: %+v", report.Failures[0])
	}
	if !strings.Contains(report.Failures[1].Got, "invalid_ruleset") {
		t.Fatalf("the actual kind must be reported: %+v", report.Failures[1])
	}
}

func TestFailureStringWithAMessage(t *testing.T) {
	f := conformance.Failure{CaseID: "id", Message: "boom"}
	if f.String() != "id: boom" {
		t.Fatalf("unexpected rendering %q", f.String())
	}
}
