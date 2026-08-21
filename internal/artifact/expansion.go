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
// What is counted is what a generator emits, which settles three questions that
// two engines answered differently before they were written down:
//
//   - it starts at the emission roots and follows operands, so a node no root
//     reaches costs nothing: a generator does not emit dead code;
//   - the roots are the program root, the subject node when the program declares
//     one, and every capture the root does not already reach. A capture the root
//     reaches is emitted inside the root's expression, and counting its subtree
//     again would charge it twice;
//   - the costs of those roots are summed, because a generator emits all of
//     them. Checking each root separately would let a program carry any number
//     of roots just below the ceiling.
//
// A CALL counts as one instance: the callee is a separate program, emitted once
// and reached by a function call, bounded on its own.
//
// The arithmetic saturates rather than wrapping. A chain two hundred levels deep
// reaches 2^201, and an accumulator that overflows lands on a small number that
// passes: the overflow is the shape of the attack, not an edge case.
func checkExpansion(p *irv1.Program) error {
	nodes := p.GetNodes()
	if len(nodes) == 0 {
		return nil
	}
	const ceiling = int64(limits.MaxStepsPerValidation)

	costs := make([]int64, len(nodes))
	for i, n := range nodes {
		total := int64(1)
		for _, in := range n.GetInputNodes() {
			if int(in) >= i {
				// Forward references are refused elsewhere; stopping here keeps
				// this check total whatever order it runs in.
				continue
			}
			total = saturatingAdd(total, costs[in])
		}
		costs[i] = total
	}

	var emitted int64
	for _, root := range emissionRoots(p, costs) {
		emitted = saturatingAdd(emitted, costs[root])
	}
	if emitted > ceiling {
		return invalidf(
			"program %d expands to more than %d operation instances once repeated "+
				"operands are inlined; a generator cannot emit it",
			p.GetId(), ceiling)
	}
	return nil
}

// emissionRoots are the nodes a generator emits from, each counted once.
func emissionRoots(p *irv1.Program, costs []int64) []uint32 {
	inRange := func(i uint32) bool { return int(i) < len(costs) }

	roots := make([]uint32, 0, 2+len(p.GetCaptures()))
	if inRange(p.GetRootNode()) {
		roots = append(roots, p.GetRootNode())
	}
	if p.SubjectNode != nil && inRange(p.GetSubjectNode()) {
		roots = append(roots, p.GetSubjectNode())
	}

	// A capture the root already reaches is emitted inside the root's
	// expression; only the ones it misses add anything.
	reached := reachableFrom(p, roots)
	for _, c := range p.GetCaptures() {
		if inRange(c.GetNode()) && !reached[c.GetNode()] {
			roots = append(roots, c.GetNode())
			for n := range reachableFrom(p, []uint32{c.GetNode()}) {
				reached[n] = true
			}
		}
	}
	return roots
}

// reachableFrom walks operands from the given nodes.
func reachableFrom(p *irv1.Program, from []uint32) map[uint32]bool {
	seen := make(map[uint32]bool, len(p.GetNodes()))
	var walk func(uint32)
	walk = func(i uint32) {
		if seen[i] || int(i) >= len(p.GetNodes()) {
			return
		}
		seen[i] = true
		for _, in := range p.GetNodes()[i].GetInputNodes() {
			walk(in)
		}
	}
	for _, i := range from {
		walk(i)
	}
	return seen
}

// saturatingAdd stops at the budget instead of wrapping. A chain two hundred
// levels deep reaches 2^201, and an accumulator that overflows lands on a small
// number that passes: the overflow is the shape of the attack, not an edge case.
func saturatingAdd(a, b int64) int64 {
	const ceiling = int64(limits.MaxStepsPerValidation)
	if a > ceiling-b {
		return ceiling + 1
	}
	return a + b
}
