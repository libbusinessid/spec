package artifact

import (
	"fmt"
	"strings"
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/features"
)

// A reader of the coverage document should see a sentence when nothing is
// unused, not an empty table. That branch is the one the project is working
// towards, so it is worth knowing it renders.
func TestRenderUnusedOperations(t *testing.T) {
	render := func(bundle *irv1.RuleBundle) string {
		var sb strings.Builder
		renderUnusedOperations(func(format string, args ...any) {
			fmt.Fprintf(&sb, format+"\n", args...)
		}, bundle)
		return sb.String()
	}

	t.Run("nothing exercised", func(t *testing.T) {
		out := render(&irv1.RuleBundle{})
		if !strings.Contains(out, "| Operation |") {
			t.Fatalf("expected a table of every operation, got:\n%s", out)
		}
		if strings.Contains(out, "None.") {
			t.Fatalf("an empty bundle exercises nothing, so nothing is used:\n%s", out)
		}
	})

	t.Run("everything exercised", func(t *testing.T) {
		out := render(&irv1.RuleBundle{Programs: []*irv1.Program{{
			Id: 1, Nodes: nodeForEveryOperation(),
		}}})
		if !strings.Contains(out, "None.") {
			t.Fatalf("expected the section to say None., got:\n%s", out)
		}
		if strings.Contains(out, "| Operation |") {
			t.Fatalf("an empty list must not print a table header:\n%s", out)
		}
	})
}

// nodeForEveryOperation builds one node per catalogued operation, which is what
// a bundle exercising the whole IR would look like.
func nodeForEveryOperation() []*irv1.Node {
	nodes := make([]*irv1.Node, 0, len(features.Ops()))
	for _, op := range features.Ops() {
		n := &irv1.Node{}
		switch op.Category {
		case features.CategoryString:
			n.Operation = &irv1.Node_StringOperation{
				StringOperation: &irv1.StringOperation{Kind: irv1.StringOpKind(op.Code)}}
		case features.CategoryInteger:
			n.Operation = &irv1.Node_IntegerOperation{
				IntegerOperation: &irv1.IntegerOperation{Kind: irv1.IntegerOpKind(op.Code)}}
		case features.CategoryPredicate:
			n.Operation = &irv1.Node_PredicateOperation{
				PredicateOperation: &irv1.PredicateOperation{Kind: irv1.PredicateOpKind(op.Code)}}
		case features.CategoryCanonicalization:
			n.Operation = &irv1.Node_CanonicalizationOperation{
				CanonicalizationOperation: &irv1.CanonicalizationOperation{
					Kind: irv1.CanonicalizationOpKind(op.Code)}}
		case features.CategoryAssertion:
			n.Operation = &irv1.Node_AssertionOperation{
				AssertionOperation: &irv1.AssertionOperation{Kind: irv1.AssertionOpKind(op.Code)}}
		case features.CategoryChecksum:
			n.Operation = &irv1.Node_ChecksumOperation{
				ChecksumOperation: &irv1.ChecksumOperation{Kind: irv1.ChecksumOpKind(op.Code)}}
		case features.CategoryCall:
			n.Operation = &irv1.Node_CallOperation{
				CallOperation: &irv1.CallOperation{Kind: irv1.CallOpKind(op.Code)}}
		default:
			continue
		}
		nodes = append(nodes, n)
	}
	return nodes
}
