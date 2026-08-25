package reference_test

import (
	"os"
	"path/filepath"
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/features"
	"github.com/entid-org/spec/internal/reference"
)

// probeEngine compiles the synthetic rule set exercising every V1 operation.
func probeEngine(t *testing.T, optimize bool) (*reference.Engine, *artifact.Ruleset) {
	t.Helper()
	dir := t.TempDir()
	source, err := os.ReadFile(filepath.Join("testdata", "opcodes.hcl"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "opcodes.hcl"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	result, bag := artifact.CompileRules(dir, artifact.CompileOptions{
		RulesVersion: "2026.08.0", Optimize: optimize,
	})
	if bag.HasErrors() {
		for _, d := range bag.Sorted() {
			t.Error(d.String())
		}
		t.FailNow()
	}
	return reference.NewEngineFromRuleset(result.Ruleset), result.Ruleset
}

// TestEveryOperationIsExercised proves that the probe rule set covers every
// concrete operation of the catalog, so that the interpreter dispatch and the
// bundle validator are fully exercised.
func TestEveryOperationIsExercised(t *testing.T) {
	_, rules := probeEngine(t, false)
	seen := map[string]bool{}
	for _, p := range rules.Bundle.GetPrograms() {
		for _, n := range p.GetNodes() {
			seen[opSymbol(t, n)] = true
		}
	}
	var missing []string
	for _, op := range features.Ops() {
		if !seen[op.Symbol] {
			missing = append(missing, op.Symbol)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("the probe rule set does not exercise %v", missing)
	}
}

func opSymbol(t *testing.T, n *irv1.Node) string {
	t.Helper()
	switch op := n.GetOperation().(type) {
	case *irv1.Node_StringOperation:
		return op.StringOperation.GetKind().String()
	case *irv1.Node_IntegerOperation:
		return op.IntegerOperation.GetKind().String()
	case *irv1.Node_PredicateOperation:
		return op.PredicateOperation.GetKind().String()
	case *irv1.Node_CanonicalizationOperation:
		return op.CanonicalizationOperation.GetKind().String()
	case *irv1.Node_AssertionOperation:
		return op.AssertionOperation.GetKind().String()
	case *irv1.Node_ChecksumOperation:
		return op.ChecksumOperation.GetKind().String()
	case *irv1.Node_CallOperation:
		return op.CallOperation.GetKind().String()
	default:
		t.Fatal("a node without operation reached the bundle")
		return ""
	}
}

func TestProbeCanonicalizationExercisesEveryStep(t *testing.T) {
	e, _ := probeEngine(t, false)
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"conditional branch applies every step", " zz12 ", "P99XX012S"},
		{"conditional branch does not apply", "XX01234567", "XX01234567"},
		{"prefix is added when missing", "01234567", "XX01234567"},
		{"second accepted prefix is kept", "YY01234567", "YY01234567"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := e.Canonicalize(reference.Input{Kind: "probe", Value: tc.value}, reference.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if got.CanonicalValue != tc.want {
				t.Fatalf("got %q, want %q", got.CanonicalValue, tc.want)
			}
		})
	}
}

func TestProbeChecksumBranches(t *testing.T) {
	e, _ := probeEngine(t, false)
	// The probe checksum chains every checksum operation through any_check, so
	// a value is valid as soon as one branch matches and unsupported when the
	// arithmetic is indeterminate.
	report, err := e.Validate(reference.Input{Kind: "probe", Value: "XX01234567"}, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Format.Status != reference.StatusValid {
		t.Fatalf("unexpected format result %+v", report.Format)
	}
	if report.Checksum.Status != reference.StatusValid && report.Checksum.Status != reference.StatusInvalid {
		t.Fatalf("unexpected checksum status %q", report.Checksum.Status)
	}
}

func TestOptimizedAndUnoptimizedBundlesAgree(t *testing.T) {
	plain, plainRules := probeEngine(t, false)
	optimized, optimizedRules := probeEngine(t, true)
	if len(optimizedRules.Bundle.GetPrograms()) != len(plainRules.Bundle.GetPrograms()) {
		t.Fatal("deduplication must not change the number of programs")
	}
	optimizedNodes, plainNodes := 0, 0
	for _, p := range optimizedRules.Bundle.GetPrograms() {
		optimizedNodes += len(p.GetNodes())
	}
	for _, p := range plainRules.Bundle.GetPrograms() {
		plainNodes += len(p.GetNodes())
	}
	if optimizedNodes > plainNodes {
		t.Fatalf("deduplication produced more nodes: %d > %d", optimizedNodes, plainNodes)
	}
	values := []string{
		"", " ", "zz12", "XX01234567", "YY01234567", "01234567", "XX0123456", "XX012345678",
		"XXABCDEFGH", "xx-01-23-45-67", "XX0123456A", "0", "XX00000000",
	}
	for _, value := range values {
		for _, profile := range []reference.Profile{reference.ProfileCompatible, reference.ProfileStrictCurrent} {
			in := reference.Input{Kind: "probe", Value: value}
			opts := reference.Options{Profile: profile}
			a, err := plain.Validate(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			b, err := optimized.Validate(in, opts)
			if err != nil {
				t.Fatal(err)
			}
			if !sameReport(a, b) {
				t.Fatalf("deduplication changed the result for %q: %+v vs %+v", value, a, b)
			}
		}
	}
}

func TestProbeRejectionsCoverEveryAssertion(t *testing.T) {
	e, _ := probeEngine(t, false)
	tests := []struct {
		name  string
		value string
		key   string
	}{
		{"length", "XX", "probe.length"},
		{"length_in", "XX012345678901234", "probe.length_in"},
		{"digits", "XX0123456A", "probe.digits"},
		{"body length", "XX0123456787", "probe.body.length"},
		{"ends_with", "XX012345678", "probe.ends"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			report, err := e.ValidateFormat(reference.Input{Kind: "probe", Value: tc.value}, reference.Options{})
			if err != nil {
				t.Fatal(err)
			}
			if report.Format.Status != reference.StatusInvalid {
				t.Fatalf("expected an invalid format, got %+v", report.Format)
			}
			if report.Format.MessageKey == nil || *report.Format.MessageKey != tc.key {
				t.Fatalf("expected the message key %q, got %v", tc.key, show(report.Format.MessageKey))
			}
		})
	}
}
