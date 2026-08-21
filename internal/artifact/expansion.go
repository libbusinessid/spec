package artifact

import (
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/limits"
)

// checkExpansion refuses a program that no generator could emit.
//
// A generator that inlines an operand where the graph reads it more than once
// produces one instance per path to a node, not one per node. The node count is
// bounded and the graph is acyclic, so every load check passes - and a DAG whose
// every node reads the previous one twice still reaches 2^n instances. Such a
// bundle is a denial of service against the generator rather than against the
// engine, and nothing else here would see it.
//
// The bound is the evaluation budget: a generated program may not carry more
// instances than an interpreter would have taken steps to run it once. A
// generator that shares repeated operands instead of inlining them is free to,
// provided it keeps the short circuit of ALL, ANY and the assertion sequence.
func checkExpansion(p *irv1.Program) error {
	nodes := p.GetNodes()
	if len(nodes) == 0 {
		return nil
	}
	const ceiling = int64(limits.MaxStepsPerValidation)
	counts := make([]int64, len(nodes))
	for i, n := range nodes {
		total := int64(1)
		for _, in := range n.GetInputNodes() {
			if int(in) >= i {
				// Forward references are refused elsewhere; stopping here keeps
				// this check total whatever order it runs in.
				continue
			}
			total += counts[in]
			if total > ceiling {
				total = ceiling + 1
				break
			}
		}
		counts[i] = total
		if total > ceiling {
			return invalidf(
				"program %d expands to more than %d operation instances once repeated "+
					"operands are inlined; a generator cannot emit it",
				p.GetId(), ceiling)
		}
	}
	return nil
}
