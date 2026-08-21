package artifact

import (
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/limits"
)

// A DAG whose every node reads the previous one twice expands exponentially
// when a generator inlines repeated operands, while passing all twenty four
// load checks: the node count is bounded, the graph is acyclic, the depth is
// fine. Nothing saw it, and the bundle is a denial of service against the
// generator rather than against the engine.
//
// The TypeScript engine hit this the moment it stopped interpreting, and bounded
// it on its own side. A bound each generator invents is a bound two generators
// disagree on, so it belongs here.
func TestExpansionIsBounded(t *testing.T) {
	// Sixty doubling nodes reach 2^60 instances; the bound is 100000.
	doubling := func(n int) *irv1.Program {
		p := &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT}
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
			}},
		})
		for i := 1; i <= n; i++ {
			p.Nodes = append(p.Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				InputNodes: []uint32{uint32(i - 1), uint32(i - 1)},
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_CONCAT,
				}},
			})
		}
		return p
	}

	for name, tc := range map[string]struct {
		nodes int
		want  string
	}{
		"a graph a generator can emit": {8, ""},
		"a graph that would explode":   {60, "instances"},
	} {
		t.Run(name, func(t *testing.T) {
			err := checkExpansion(doubling(tc.nodes))
			if tc.want == "" {
				if err != nil {
					t.Fatalf("expected acceptance, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("a program expanding past %d instances must be refused",
					limits.MaxStepsPerValidation)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected an error mentioning %q, got %v", tc.want, err)
			}
		})
	}
}
