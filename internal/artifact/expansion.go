package artifact

import (
	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/limits"
)

// checkExpansion refuses a program that no generator could emit.
//
// A generator that inlines an operand where the graph reads it more than once
// produces one instance per path to a node, not one per node. The node count is
// bounded and the graph is acyclic, so every other load check passes - and a DAG
// whose every node reads the previous one twice still reaches 2^n instances.
// Such a bundle is a denial of service against the generator rather than against
// the engine, and nothing else here would see it.
//
// The count starts at the roots a generator emits from and follows operands. A
// node no root reaches is emitted by nobody and counts for nothing: the bound is
// on what a generator produces, and a generator does not emit dead code. Two
// engines disagreed on exactly this, one counting every node and one counting
// the reachable ones, and answered differently on the same bundle.
//
// The bound is the evaluation budget: a generated program may not carry more
// instances than an interpreter would have taken steps to run it once.
func checkExpansion(p *irv1.Program) error {
	nodes := p.GetNodes()
	if len(nodes) == 0 {
		return nil
	}
	const ceiling = int64(limits.MaxStepsPerValidation)

	// counts[i] is what emitting node i costs, computed bottom up. The
	// arithmetic saturates rather than wrapping: a chain two hundred levels deep
	// reaches 2^201, and an accumulator that overflows lands on a small number
	// that passes. The overflow is the shape of the attack, not an edge case.
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
	}

	for _, root := range emissionRoots(p) {
		if int(root) >= len(counts) {
			// Out of range roots are refused by the root check.
			continue
		}
		if counts[root] > ceiling {
			return invalidf(
				"program %d expands to more than %d operation instances once repeated "+
					"operands are inlined; a generator cannot emit it",
				p.GetId(), ceiling)
		}
	}
	return nil
}

// emissionRoots are the nodes a generator emits from: the program root and every
// capture. Everything else is reached through them or not at all.
func emissionRoots(p *irv1.Program) []uint32 {
	roots := make([]uint32, 0, 1+len(p.GetCaptures()))
	roots = append(roots, p.GetRootNode())
	for _, c := range p.GetCaptures() {
		roots = append(roots, c.GetNode())
	}
	return roots
}
