package artifact_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// knownUnusedOperations are the operations the IR defines and no rule emits.
//
// Every engine implements these, and no conformance case proves that
// implementation: an engine compiles the bundle into native code, so it never
// generates an operation no rule carries, yet it must still handle one in case
// a later bundle does. This list is that untested surface, written down.
//
// It is a list rather than a count so both directions are caught. Adding an
// operation nothing uses widens what every engine owes with no evidence behind
// it, and the capability freeze makes that permanent once a release exists.
// Removing one is the good direction, and the test still fails, because the
// list should say what is true.
//
// prefix_in is why this exists. It sat here fully implemented across proto,
// catalogue, type checker and interpreter while the UK company number rule
// spelled out forty one starts_with by hand, and nothing said so.
var knownUnusedOperations = []string{
	"CANONICALIZATION_OP_KIND_APPEND",
	"CANONICALIZATION_OP_KIND_LEFT_PAD",
	"CANONICALIZATION_OP_KIND_PREPEND",
	"PREDICATE_OP_KIND_ENDS_WITH",
	"PREDICATE_OP_KIND_EQUALS",
	"PREDICATE_OP_KIND_LENGTH_IN",
	"STRING_OP_KIND_CONCAT",
	"STRING_OP_KIND_CONSTANT",
	"STRING_OP_KIND_COUNTRY_CODE",
	"STRING_OP_KIND_SLICE_TO",
	"STRING_OP_KIND_STRIP_PREFIX",
}

func TestUnusedOperationsAreTheOnesWeKnowAbout(t *testing.T) {
	unused := unusedFromCoverageDoc(t)

	for _, got := range unused {
		if !slices.Contains(knownUnusedOperations, got) {
			t.Errorf("%s is defined and no rule emits it.\n"+
				"Every engine has to implement it and no conformance case proves that "+
				"implementation.\nEither write a rule that needs it, or add it to "+
				"knownUnusedOperations and say why it earns its place.", got)
		}
	}
	for _, want := range knownUnusedOperations {
		if !slices.Contains(unused, want) {
			t.Errorf("%s is listed as unused and a rule now emits it.\n"+
				"Remove it from knownUnusedOperations: the list should say what is true.", want)
		}
	}
	if t.Failed() {
		t.Logf("the coverage document currently lists: %s", strings.Join(unused, ", "))
	}
}

// unusedFromCoverageDoc reads the list out of the generated coverage document,
// which is the same page a reader of this project sees and which
// check-generated keeps in step with the rules.
func unusedFromCoverageDoc(t *testing.T) []string {
	t.Helper()
	path := filepath.Join("..", "..", "docs", "generated", "coverage.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	text := string(raw)
	const heading = "## Operations no rule exercises"
	i := strings.Index(text, heading)
	if i < 0 {
		t.Fatalf("%s carries no %q section", path, heading)
	}
	section := text[i+len(heading):]
	if j := strings.Index(section, "\n## "); j >= 0 {
		section = section[:j]
	}
	out := regexp.MustCompile(`\| `+"`"+`([A-Z_0-9]+)`+"`"+` \|`).FindAllStringSubmatch(section, -1)
	symbols := make([]string, 0, len(out))
	for _, m := range out {
		symbols = append(symbols, m[1])
	}
	slices.Sort(symbols)
	return symbols
}
