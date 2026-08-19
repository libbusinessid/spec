package typecheck_test

import (
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/typecheck"
)

func TestCheckedNodeType(t *testing.T) {
	unit := mustCheck(t, `
format "a" "b" {
  checks = [require(is_empty(subject()), "empty", "a.b.empty")]
}
`)
	program := unit.BySymbol["format.a.b"]
	if program.Root.Type() != irv1.ValueType_VALUE_TYPE_ASSERTION {
		t.Fatalf("unexpected root type %v", program.Root.Type())
	}
	predicate := program.Root.Inputs[0].Inputs[0]
	if predicate.Type() != irv1.ValueType_VALUE_TYPE_BOOLEAN {
		t.Fatalf("unexpected predicate type %v", predicate.Type())
	}
	if predicate.Inputs[0].Type() != irv1.ValueType_VALUE_TYPE_STRING {
		t.Fatalf("unexpected operand type %v", predicate.Inputs[0].Type())
	}
}

func TestReasonCodeName(t *testing.T) {
	if got := typecheck.ReasonCodeName(irv1.ReasonCode_REASON_CODE_NOT_RUN_FORMAT_INVALID); got != "not_run_format_invalid" {
		t.Fatalf("unexpected name %q", got)
	}
	if got := typecheck.ReasonCodeName(irv1.ReasonCode_REASON_CODE_OK); got != "ok" {
		t.Fatalf("unexpected name %q", got)
	}
}
