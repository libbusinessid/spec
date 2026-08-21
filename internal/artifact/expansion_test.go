package artifact

import (
	"os"
	"path/filepath"
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
			p := doubling(tc.nodes)
			p.RootNode = uint32(len(p.GetNodes()) - 1)
			err := checkExpansion(p)
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

// The bound is on what a generator emits, and a generator does not emit dead
// code. A chain no root reaches costs nothing, however deep it goes.
//
// Two engines disagreed here before it was written down: one counted every
// node, the other counted the reachable ones, and they answered differently on
// the same bundle. The hostile fixture was refused by both - for different
// reasons - so the conformance case could not tell them apart.
func TestExpansionFollowsTheRoots(t *testing.T) {
	p := &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT}
	p.Nodes = append(p.Nodes, &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
			Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
		}},
	})
	// The root sits at index 1 and reads only the subject.
	p.Nodes = append(p.Nodes, &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
		InputNodes: []uint32{0},
		Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
			Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS,
		}},
	})
	p.RootNode = 1
	// Sixty doubling nodes that nothing reaches.
	prev := uint32(0)
	for range 60 {
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			InputNodes: []uint32{prev, prev},
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_CONCAT,
			}},
		})
		prev = uint32(len(p.Nodes) - 1)
	}
	if err := checkExpansion(p); err != nil {
		t.Fatalf("a chain no root reaches is emitted by nobody: %v", err)
	}

	// Point the root at the top of that chain and it is emitted after all.
	p.RootNode = prev
	if err := checkExpansion(p); err == nil {
		t.Fatal("a chain the root reaches must be counted")
	}
}

// The three questions two engines answered differently, each pinned so none of
// them comes back.
func TestExpansionCountsWhatAGeneratorEmits(t *testing.T) {
	// chain builds n doubling nodes above node 0 and returns the top index.
	chain := func(p *irv1.Program, n int) uint32 {
		prev := uint32(0)
		for range n {
			p.Nodes = append(p.Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				InputNodes: []uint32{prev, prev},
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_CONCAT,
				}},
			})
			prev = uint32(len(p.Nodes) - 1)
		}
		return prev
	}
	base := func() *irv1.Program {
		return &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT,
			Nodes: []*irv1.Node{{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
				}},
			}}}
	}

	t.Run("the roots are summed, not checked apart", func(t *testing.T) {
		// Three captures the root never reaches, each comfortably under the
		// ceiling on its own. Checking each apart would accept the program while
		// a generator emits all three.
		p := base()
		p.RootNode = 0
		for range 2 {
			top := chain(p, 15)
			p.Captures = append(p.Captures, &irv1.Capture{Name: "c", Node: top})
			p.Nodes = append(p.Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
				}},
			})
		}
		if err := checkExpansion(p); err == nil {
			t.Fatal("two roots under the ceiling still exceed it together")
		}
	})

	t.Run("a capture the root reaches is not charged twice", func(t *testing.T) {
		// One chain, the root on top of it, and a capture pointing inside it.
		// Charging the capture again would double a program a generator emits
		// once.
		p := base()
		top := chain(p, 15)
		p.RootNode = top
		p.Captures = append(p.Captures, &irv1.Capture{Name: "c", Node: top - 1})
		if err := checkExpansion(p); err != nil {
			t.Fatalf("a capture inside the root's expression is emitted once: %v", err)
		}
	})

	t.Run("the subject node is emitted too", func(t *testing.T) {
		p := base()
		p.RootNode = 0
		top := chain(p, 20)
		subject := top
		p.SubjectNode = &subject
		if err := checkExpansion(p); err == nil {
			t.Fatal("a generator emits the subject node's subtree as well")
		}
	})
}

// TestExpansionProfileOfTheShippedBundle publishes the numbers two engines
// compare against each other.
//
// No conformance case can establish that two implementations count the same
// thing, because no rule anyone would write comes near the ceiling: every
// reading agrees on every real bundle. Comparing the profile is what made the
// disagreements visible - one engine reporting 3204 instances where another
// reported 3069 found that it was charging captures the root already reached.
//
// It also holds the invariant, so a drift shows up without anyone remembering
// to look.
func TestExpansionProfileOfTheShippedBundle(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "dist",
		"businessid-rules-"+strings.TrimSpace(readVersion(t))+".binpb"))
	if err != nil {
		t.Skip("no compiled bundle in dist; run businessidc compile first")
	}
	rs, err := LoadRuleset(raw)
	if err != nil {
		t.Fatalf("the shipped bundle must load: %v", err)
	}
	bundle := rs.Bundle
	var total, worst int64
	var worstID uint32
	for _, p := range bundle.GetPrograms() {
		costs := make([]int64, len(p.GetNodes()))
		for i, n := range p.GetNodes() {
			c := int64(1)
			for _, in := range n.GetInputNodes() {
				if int(in) < i {
					c = saturatingAdd(c, costs[in])
				}
			}
			costs[i] = c
		}
		var emitted int64
		for _, root := range emissionRoots(p, costs) {
			emitted = saturatingAdd(emitted, costs[root])
		}
		total += emitted
		if emitted > worst {
			worst, worstID = emitted, p.GetId()
		}
		if emitted > int64(limits.MaxStepsPerValidation) {
			t.Fatalf("program %d exceeds the budget at %d instances", p.GetId(), emitted)
		}
	}
	t.Logf("%s: %d programs, %d instances in total, worst program %d at %d, budget %d",
		bundle.GetRulesVersion(), len(bundle.GetPrograms()), total, worstID, worst,
		limits.MaxStepsPerValidation)
}

func readVersion(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "RULES_VERSION"))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// A capture reached by another capture is not a second emission either, and the
// order the captures are listed in must not change the answer.
//
// The TypeScript engine found this: it excluded a capture only when the program
// root reached it, not when another root did. Ours excludes it whichever root
// reaches it, but only if that root was seen first - and the capture list is in
// no particular order. Taking the captures from the highest index down settles
// it, since an operand always sits at a lower index than the node reading it.
func TestExpansionIgnoresTheOrderCapturesAreListedIn(t *testing.T) {
	build := func(order []int) *irv1.Program {
		p := &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT,
			Nodes: []*irv1.Node{{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
				}},
			}},
		}
		// A detached chain: node 1 reads node 0, node 2 reads node 1.
		for i := 1; i <= 2; i++ {
			p.Nodes = append(p.Nodes, &irv1.Node{
				OutputType: irv1.ValueType_VALUE_TYPE_STRING,
				InputNodes: []uint32{uint32(i - 1)},
				Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
					Kind: irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM,
				}},
			})
		}
		// The root is the bare subject and reaches neither capture.
		p.RootNode = 0
		for _, n := range order {
			p.Captures = append(p.Captures, &irv1.Capture{Name: "c", Node: uint32(n)})
		}
		return p
	}

	cost := func(p *irv1.Program) int64 {
		costs := make([]int64, len(p.GetNodes()))
		for i, n := range p.GetNodes() {
			c := int64(1)
			for _, in := range n.GetInputNodes() {
				if int(in) < i {
					c = saturatingAdd(c, costs[in])
				}
			}
			costs[i] = c
		}
		var total int64
		for _, r := range emissionRoots(p, costs) {
			total = saturatingAdd(total, costs[r])
		}
		return total
	}

	// Capture 2 reaches capture 1, so emitting capture 2 emits both. The answer
	// is the same whichever order they appear in.
	high := cost(build([]int{2, 1}))
	low := cost(build([]int{1, 2}))
	if high != low {
		t.Fatalf("the capture order changed the count: %d listed high first, %d low first",
			high, low)
	}
}
