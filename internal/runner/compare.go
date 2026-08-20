package runner

import (
	"fmt"

	conformancev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/conformance/v1"
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	testeev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/testee/v1"
)

// Diff is one field on which an engine departed from the corpus.
type Diff struct {
	CaseID string
	Field  string
	Want   string
	Got    string
}

func (d Diff) String() string {
	return fmt.Sprintf("%s: %s want=%s got=%s", d.CaseID, d.Field, d.Want, d.Got)
}

// compare confronts one response with the expectation of its case.
//
// Every difference is reported, including a missing or misshapen result. An
// engine is conformant only when this returns nothing: there is no tolerance,
// no skipping and no partial credit.
func compare(c *conformancev1.ConformanceCase, resp *testeev1.TesteeResponse, formatStatusOnly bool) []Diff {
	var out []Diff
	add := func(field, want, got string) {
		out = append(out, Diff{CaseID: c.GetId(), Field: field, Want: want, Got: got})
	}

	if f := resp.GetFailure(); f != nil {
		add("result", "an observation", fmt.Sprintf("failure %s: %s", f.GetKind(), f.GetDetail()))
		return out
	}

	if c.GetOperation() == conformancev1.Operation_OPERATION_LOAD_RULESET {
		load := resp.GetLoad()
		if load == nil {
			add("result", "a load outcome", shapeOf(resp))
			return out
		}
		wantErr := c.GetExpectedEngineError()
		if wantErr == "" {
			if !load.GetAccepted() {
				add("accepted", "true", "false")
			}
			return out
		}
		if load.GetAccepted() {
			add("accepted", "false", "true")
			return out
		}
		if load.GetEngineError() != wantErr {
			add("engineError", wantErr, load.GetEngineError())
		}
		return out
	}

	switch want := c.GetExpected().GetValue().(type) {
	case *conformancev1.ExpectedOutcome_Canonicalization:
		got := resp.GetCanonicalization()
		if got == nil {
			add("result", "a canonicalization", shapeOf(resp))
			return out
		}
		w := want.Canonicalization
		cmpString(add, "kind", w.GetKind(), got.GetKind())
		cmpString(add, "canonicalValue", w.GetCanonicalValue(), got.GetCanonicalValue())
		cmpOptional(add, "countryCode", w.CountryCode, got.CountryCode)
		cmpStatus(add, "status", w.GetStatus(), got.GetStatus())
		cmpReason(add, "reasonCode", w.GetReasonCode(), got.GetReasonCode())
	case *conformancev1.ExpectedOutcome_ValidationReport:
		got := resp.GetValidationReport()
		if got == nil {
			add("result", "a validation report", shapeOf(resp))
			return out
		}
		w := want.ValidationReport
		if formatStatusOnly {
			// A register sweep may assert exactly one thing, because exactly
			// one thing is what the issuer's register establishes: this
			// identifier exists, therefore it is valid. The register says
			// nothing about the canonical form of the value, and nothing about
			// its checksum. Filling those in would mean computing them with the
			// very interpreter under test, which is the one thing a conformance
			// expectation may never be derived from.
			cmpStatus(add, "format.status", w.GetFormat().GetStatus(), got.GetFormat().GetStatus())
			return out
		}
		cmpString(add, "kind", w.GetKind(), got.GetKind())
		cmpString(add, "canonicalValue", w.GetCanonicalValue(), got.GetCanonicalValue())
		cmpOptional(add, "countryCode", w.CountryCode, got.CountryCode)
		cmpStep(add, "format", w.GetFormat(), got.GetFormat())
		cmpStep(add, "checksum", w.GetChecksum(), got.GetChecksum())
	default:
		add("expected", "a reviewed expectation", "the case carries none")
	}
	return out
}

func cmpStep(add func(string, string, string), name string, want *conformancev1.ExpectedStep, got *testeev1.ObservedStep) {
	if got == nil {
		add(name, want.GetStatus().String(), "absent")
		return
	}
	cmpStatus(add, name+".status", want.GetStatus(), got.GetStatus())
	cmpReason(add, name+".reasonCode", want.GetReasonCode(), got.GetReasonCode())
}

func cmpString(add func(string, string, string), field, want, got string) {
	if want != got {
		add(field, want, got)
	}
}

func cmpStatus(add func(string, string, string), field string, want, got conformancev1.StepStatus) {
	if want != got {
		add(field, want.String(), got.String())
	}
}

func cmpReason(add func(string, string, string), field string, want, got irv1.ReasonCode) {
	if want != got {
		add(field, want.String(), got.String())
	}
}

// cmpOptional keeps absence distinct from emptiness. Treating the two as equal
// is the laxity that lets a non conformant engine pass.
func cmpOptional(add func(string, string, string), field string, want, got *string) {
	switch {
	case want == nil && got == nil:
	case want == nil:
		add(field, "absent", *got)
	case got == nil:
		add(field, *want, "absent")
	case *want != *got:
		add(field, *want, *got)
	}
}

func shapeOf(resp *testeev1.TesteeResponse) string {
	switch {
	case resp.GetCanonicalization() != nil:
		return "a canonicalization"
	case resp.GetValidationReport() != nil:
		return "a validation report"
	case resp.GetLoad() != nil:
		return "a load outcome"
	default:
		return "nothing"
	}
}
