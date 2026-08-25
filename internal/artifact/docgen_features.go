package artifact

import (
	"fmt"
	"sort"
	"strings"

	"github.com/entid-org/spec/internal/features"
)

// RenderFeaturesDoc renders docs/features.md, the frozen content of every
// capability ID.
func RenderFeaturesDoc() []byte {
	var b strings.Builder
	w := func(format string, args ...any) { fmt.Fprintf(&b, format+"\n", args...) }

	w("%s", generatedBanner)
	w("")
	w("# LibEntID capabilities v1")
	w("")
	w("A capability ID designates an exact and frozen set of operations, fields, bounds")
	w("and semantics. That set can never be widened or reinterpreted. IDs are never")
	w("renumbered and never reused. A new operation, a new variant of an operation or")
	w("any observable change always receives a new ID, even when it is conceptually")
	w("close to an existing capability.")
	w("")
	w("An engine publishes the list of capability IDs it implements and refuses to load")
	w("a bundle declaring a single unknown ID.")
	w("")
	w("## Registry")
	w("")
	w("| ID | Name | Content |")
	w("|---:|---|---|")
	for _, c := range features.All() {
		w("| %d | `%s` | %s |", c.ID, c.Name, c.Summary)
	}
	w("")

	byCapability := map[uint32][]features.Op{}
	for _, op := range features.Ops() {
		for _, id := range op.Features {
			byCapability[id] = append(byCapability[id], op)
		}
	}
	for _, c := range features.All() {
		w("## %d - `%s`", c.ID, c.Name)
		w("")
		w("%s.", titleASCII(c.Summary))
		w("")
		w("Frozen content:")
		w("")
		for _, item := range c.Content {
			w("- %s", item)
		}
		w("")
		ops := byCapability[c.ID]
		if len(ops) == 0 {
			w("This capability declares no operation of its own: it is required by the")
			w("bundle level constructs listed above.")
			w("")
			continue
		}
		sort.Slice(ops, func(i, j int) bool { return ops[i].Symbol < ops[j].Symbol })
		w("Operations requiring this capability:")
		w("")
		for _, op := range ops {
			w("- `%s`", op.Symbol)
		}
		w("")
	}
	return []byte(b.String())
}
