package artifact

import (
	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
)

// checkSubjectNode refuses a subject defined in terms of itself.
//
// A program may name a node whose value becomes the subject of a top level
// invocation. If that node's subtree reads SUBJECT, the subject is defined by
// itself: a generator emitting the subtree recurses forever, and an interpreter
// exhausts its evaluation budget. No reading of the IR makes such a bundle
// usable, and none of the other checks looked - the subject node was verified
// for range and for type, and its subtree was never walked.
//
// Found by an engine that refused it on its own rather than assuming the
// specification had thought of it.
func checkSubjectNode(p *irv1.Program) error {
	if p.SubjectNode == nil {
		return nil
	}
	nodes := p.GetNodes()
	seen := make(map[uint32]bool, len(nodes))

	var walk func(uint32) bool
	walk = func(i uint32) bool {
		if int(i) >= len(nodes) || seen[i] {
			return false
		}
		seen[i] = true
		if op := nodes[i].GetStringOperation(); op != nil &&
			op.GetKind() == irv1.StringOpKind_STRING_OP_KIND_SUBJECT {
			return true
		}
		for _, in := range nodes[i].GetInputNodes() {
			if walk(in) {
				return true
			}
		}
		return false
	}

	if walk(p.GetSubjectNode()) {
		return invalidf(
			"program %d builds its subject node from the subject it defines",
			p.GetId())
	}
	return nil
}
