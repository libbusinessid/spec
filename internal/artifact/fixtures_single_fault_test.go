package artifact

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/limits"
)

// The fixture that exercises the left_pad bound must be invalid for that bound
// alone.
//
// It placed the padding node in the format program and rooted the format
// program there, so a canonicalization operation was refused for being in a
// format program and the length of the case's name never counted. Bring the
// length under the bound and the bundle stayed refused, for a reason the case
// is not about: an engine with no bound on left_pad passed it.
//
// Fifth fixture of this family, after message_key, subject_node_circular and
// program_expansion twice. Measured by the Swift engine, as three of the others
// were measured by engines rather than here.
func TestTheLeftPadFixtureIsInvalidForThatReasonAlone(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bundles", "left_pad_length.binpb"))
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	if _, err = LoadRuleset(raw); err == nil {
		t.Fatal("the fixture loads; it is supposed to be refused")
	}
	// The refusal must name the over long padding itself, not some other thing
	// that happens to be wrong: that is the whole point of the property below.
	if !strings.Contains(err.Error(), strconv.Itoa(limits.MaxIndex+1)) {
		t.Fatalf("the fixture must be refused for its padding of %d, got %v", limits.MaxIndex+1, err)
	}

	// Repair the one fault the case is named for, and nothing else.
	var bundle irv1.RuleBundle
	if err = proto.Unmarshal(raw, &bundle); err != nil {
		t.Fatalf("decoding the fixture: %v", err)
	}
	repaired := 0
	for _, p := range bundle.GetPrograms() {
		for _, n := range p.GetNodes() {
			op := n.GetCanonicalizationOperation()
			if op == nil || op.GetLength() <= limits.MaxIndex {
				continue
			}
			op.Length = proto.Uint32(4)
			repaired++
		}
	}
	if repaired != 1 {
		t.Fatalf("repaired %d over long paddings, want exactly 1", repaired)
	}
	sound, err := proto.Marshal(&bundle)
	if err != nil {
		t.Fatalf("re-encoding: %v", err)
	}
	if _, err = LoadRuleset(sound); err != nil {
		t.Fatalf("with its padding under the bound the fixture is still refused: %v\n"+
			"The case cannot distinguish an engine that bounds left_pad from one that "+
			"does not: both answer invalid_ruleset, for different reasons.", err)
	}
}
