package runner

import (
	"testing"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/conformance/v1"
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	testeev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/testee/v1"
)

func validationCase(country *string) *conformancev1.ConformanceCase {
	return &conformancev1.ConformanceCase{
		Id:          "c1",
		Kind:        "siren",
		CountryCode: country,
		Operation:   conformancev1.Operation_OPERATION_VALIDATE,
		Expected: &conformancev1.ExpectedOutcome{Value: &conformancev1.ExpectedOutcome_ValidationReport{
			ValidationReport: &conformancev1.ExpectedValidationReport{
				Kind:           "siren",
				CanonicalValue: "552100554",
				CountryCode:    country,
				Format:         &conformancev1.ExpectedStep{Status: conformancev1.StepStatus_STEP_STATUS_VALID},
				Checksum:       &conformancev1.ExpectedStep{Status: conformancev1.StepStatus_STEP_STATUS_VALID},
			},
		}},
	}
}

func validationResponse(canonical string) *testeev1.TesteeResponse {
	return &testeev1.TesteeResponse{CaseId: "c1", Result: &testeev1.TesteeResponse_ValidationReport{
		ValidationReport: &testeev1.ObservedValidationReport{
			Kind:           "siren",
			CanonicalValue: canonical,
			CountryCode:    proto.String("FR"),
			Format:         &testeev1.ObservedStep{Status: conformancev1.StepStatus_STEP_STATUS_VALID},
			Checksum:       &testeev1.ObservedStep{Status: conformancev1.StepStatus_STEP_STATUS_VALID},
		},
	}}
}

func TestAcceptsAnExactMatch(t *testing.T) {
	if d := compare(validationCase(proto.String("FR")), validationResponse("552100554")); len(d) != 0 {
		t.Fatalf("an exact match must produce no diff, got %v", d)
	}
}

func TestDetectsAWrongCanonicalValue(t *testing.T) {
	d := compare(validationCase(proto.String("FR")), validationResponse("552100555"))
	if len(d) != 1 || d[0].Field != "canonicalValue" {
		t.Fatalf("a wrong canonical value must be reported, got %v", d)
	}
}

func TestDetectsAWrongStepStatus(t *testing.T) {
	resp := validationResponse("552100554")
	resp.GetValidationReport().Checksum.Status = conformancev1.StepStatus_STEP_STATUS_INVALID
	d := compare(validationCase(proto.String("FR")), resp)
	if len(d) != 1 || d[0].Field != "checksum.status" {
		t.Fatalf("a wrong checksum status must be reported, got %v", d)
	}
}

func TestDetectsAWrongReasonCode(t *testing.T) {
	c := validationCase(proto.String("FR"))
	c.GetExpected().GetValidationReport().Format.ReasonCode = irv1.ReasonCode_REASON_CODE_INVALID_LENGTH
	d := compare(c, validationResponse("552100554"))
	if len(d) != 1 || d[0].Field != "format.reasonCode" {
		t.Fatalf("a wrong reason code must be reported, got %v", d)
	}
}

// An absent country code must not be treated as equal to an empty or present
// one: that laxity is exactly how a non conformant engine passes.
func TestDistinguishesAbsentFromPresentCountryCode(t *testing.T) {
	d := compare(validationCase(nil), validationResponse("552100554"))
	if len(d) != 1 || d[0].Field != "countryCode" {
		t.Fatalf("an unexpected country code must be reported, got %v", d)
	}
}

func TestAFailureIsNeverConformant(t *testing.T) {
	resp := &testeev1.TesteeResponse{CaseId: "c1", Result: &testeev1.TesteeResponse_Failure{
		Failure: &testeev1.TesteeFailure{Kind: testeev1.FailureKind_FAILURE_KIND_UNSUPPORTED_OPERATION, Detail: "todo"},
	}}
	if d := compare(validationCase(proto.String("FR")), resp); len(d) == 0 {
		t.Fatal("an unsupported operation must be reported as a difference, never skipped")
	}
}

func TestAMissingResultIsADifference(t *testing.T) {
	if d := compare(validationCase(proto.String("FR")), &testeev1.TesteeResponse{CaseId: "c1"}); len(d) == 0 {
		t.Fatal("an empty response must be reported")
	}
}

// Answering a validation case with a canonicalization result must not be
// mistaken for a match on the fields the two shapes share.
func TestDetectsAResultOfTheWrongShape(t *testing.T) {
	resp := &testeev1.TesteeResponse{CaseId: "c1", Result: &testeev1.TesteeResponse_Canonicalization{
		Canonicalization: &testeev1.ObservedCanonicalization{Kind: "siren", CanonicalValue: "552100554"},
	}}
	if d := compare(validationCase(proto.String("FR")), resp); len(d) == 0 {
		t.Fatal("a result of the wrong shape must be reported")
	}
}

func TestLoadRuleset(t *testing.T) {
	c := &conformancev1.ConformanceCase{
		Id:                  "l1",
		Operation:           conformancev1.Operation_OPERATION_LOAD_RULESET,
		ExpectedEngineError: proto.String("invalid_ruleset"),
	}
	t.Run("refused with the right error", func(t *testing.T) {
		resp := &testeev1.TesteeResponse{CaseId: "l1", Result: &testeev1.TesteeResponse_Load{
			Load: &testeev1.ObservedLoad{Accepted: false, EngineError: "invalid_ruleset"},
		}}
		if d := compare(c, resp); len(d) != 0 {
			t.Fatalf("got %v", d)
		}
	})
	t.Run("accepting a hostile bundle is a difference", func(t *testing.T) {
		resp := &testeev1.TesteeResponse{CaseId: "l1", Result: &testeev1.TesteeResponse_Load{
			Load: &testeev1.ObservedLoad{Accepted: true},
		}}
		if d := compare(c, resp); len(d) == 0 {
			t.Fatal("accepting a bundle that must be refused is a difference")
		}
	})
	t.Run("refused with the wrong error type", func(t *testing.T) {
		resp := &testeev1.TesteeResponse{CaseId: "l1", Result: &testeev1.TesteeResponse_Load{
			Load: &testeev1.ObservedLoad{Accepted: false, EngineError: "incompatible_ruleset"},
		}}
		d := compare(c, resp)
		if len(d) != 1 || d[0].Field != "engineError" {
			t.Fatalf("the error type is part of the contract, got %v", d)
		}
	})
}

func canonicalCase() *conformancev1.ConformanceCase {
	return &conformancev1.ConformanceCase{
		Id:        "k1",
		Kind:      "vat",
		Operation: conformancev1.Operation_OPERATION_CANONICALIZE,
		Expected: &conformancev1.ExpectedOutcome{Value: &conformancev1.ExpectedOutcome_Canonicalization{
			Canonicalization: &conformancev1.ExpectedCanonicalization{
				Kind:           "vat",
				CanonicalValue: "FR12345678901",
				CountryCode:    proto.String("FR"),
				Status:         conformancev1.StepStatus_STEP_STATUS_VALID,
			},
		}},
	}
}

func canonicalResponse() *testeev1.TesteeResponse {
	return &testeev1.TesteeResponse{CaseId: "k1", Result: &testeev1.TesteeResponse_Canonicalization{
		Canonicalization: &testeev1.ObservedCanonicalization{
			Kind:           "vat",
			CanonicalValue: "FR12345678901",
			CountryCode:    proto.String("FR"),
			Status:         conformancev1.StepStatus_STEP_STATUS_VALID,
		},
	}}
}

func TestCanonicalization(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		if d := compare(canonicalCase(), canonicalResponse()); len(d) != 0 {
			t.Fatalf("got %v", d)
		}
	})
	t.Run("wrong kind", func(t *testing.T) {
		r := canonicalResponse()
		r.GetCanonicalization().Kind = "siren"
		if d := compare(canonicalCase(), r); len(d) != 1 || d[0].Field != "kind" {
			t.Fatalf("got %v", d)
		}
	})
	t.Run("wrong status", func(t *testing.T) {
		r := canonicalResponse()
		r.GetCanonicalization().Status = conformancev1.StepStatus_STEP_STATUS_INVALID
		if d := compare(canonicalCase(), r); len(d) != 1 || d[0].Field != "status" {
			t.Fatalf("got %v", d)
		}
	})
	t.Run("wrong reason code", func(t *testing.T) {
		r := canonicalResponse()
		r.GetCanonicalization().ReasonCode = irv1.ReasonCode_REASON_CODE_INVALID_LENGTH
		if d := compare(canonicalCase(), r); len(d) != 1 || d[0].Field != "reasonCode" {
			t.Fatalf("got %v", d)
		}
	})
	t.Run("answered with a validation report", func(t *testing.T) {
		if d := compare(canonicalCase(), validationResponse("x")); len(d) == 0 {
			t.Fatal("a result of the wrong shape must be reported")
		}
	})
	t.Run("missing country code", func(t *testing.T) {
		r := canonicalResponse()
		r.GetCanonicalization().CountryCode = nil
		if d := compare(canonicalCase(), r); len(d) != 1 || d[0].Field != "countryCode" {
			t.Fatalf("an absent country code must be reported, got %v", d)
		}
	})
}

// A step the engine did not report at all must be a difference, not a match on
// the zero value.
func TestAnAbsentStepIsADifference(t *testing.T) {
	r := validationResponse("552100554")
	r.GetValidationReport().Checksum = nil
	if d := compare(validationCase(proto.String("FR")), r); len(d) != 1 || d[0].Field != "checksum" {
		t.Fatalf("got %v", d)
	}
}

func TestACaseWithoutExpectationIsReported(t *testing.T) {
	c := validationCase(proto.String("FR"))
	c.Expected = nil
	if d := compare(c, validationResponse("552100554")); len(d) != 1 || d[0].Field != "expected" {
		t.Fatalf("got %v", d)
	}
}

func TestLoadRulesetThatMustBeAccepted(t *testing.T) {
	c := &conformancev1.ConformanceCase{Id: "l2", Operation: conformancev1.Operation_OPERATION_LOAD_RULESET}
	t.Run("accepted", func(t *testing.T) {
		r := &testeev1.TesteeResponse{CaseId: "l2", Result: &testeev1.TesteeResponse_Load{
			Load: &testeev1.ObservedLoad{Accepted: true},
		}}
		if d := compare(c, r); len(d) != 0 {
			t.Fatalf("got %v", d)
		}
	})
	t.Run("refused when it should load", func(t *testing.T) {
		r := &testeev1.TesteeResponse{CaseId: "l2", Result: &testeev1.TesteeResponse_Load{
			Load: &testeev1.ObservedLoad{Accepted: false, EngineError: "invalid_ruleset"},
		}}
		if d := compare(c, r); len(d) != 1 || d[0].Field != "accepted" {
			t.Fatalf("got %v", d)
		}
	})
	t.Run("answered with the wrong shape", func(t *testing.T) {
		if d := compare(c, canonicalResponse()); len(d) == 0 {
			t.Fatal("a load case answered with a canonicalization must be reported")
		}
	})
}

func TestDiffString(t *testing.T) {
	d := Diff{CaseID: "c1", Field: "kind", Want: "siren", Got: "vat"}
	if got := d.String(); got != "c1: kind want=siren got=vat" {
		t.Fatalf("got %q", got)
	}
}
