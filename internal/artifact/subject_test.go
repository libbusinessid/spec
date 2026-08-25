package artifact

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
)

// A subject node whose subtree reads SUBJECT defines the subject in terms of
// itself. No reading of the IR makes such a bundle usable: a generator emitting
// it recurses forever, an interpreter exhausts its budget.
//
// The Swift engine refused it on its own and asked for it to be written down.
// Nothing in the twenty five checks covered it: the subject node is checked for
// range and for type, and its subtree is never walked.
func TestSubjectNodeMayNotReadTheSubject(t *testing.T) {
	build := func(subjectReadsSubject bool) *irv1.Program {
		p := &irv1.Program{Id: 1, Kind: irv1.ProgramKind_PROGRAM_KIND_FORMAT}
		// 0: SUBJECT
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SUBJECT,
			}},
		})
		// 1: a string the subject node can be built from
		inputs := []uint32{}
		if subjectReadsSubject {
			inputs = []uint32{0}
		}
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_STRING,
			InputNodes: inputs,
			Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
				Kind: irv1.StringOpKind_STRING_OP_KIND_SLICE_FROM,
			}},
		})
		subject := uint32(1)
		p.SubjectNode = &subject
		// 2: an assertion rooting the program
		p.Nodes = append(p.Nodes, &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_BOOLEAN,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_PredicateOperation{PredicateOperation: &irv1.PredicateOperation{
				Kind: irv1.PredicateOpKind_PREDICATE_OP_KIND_ASCII_DIGITS,
			}},
		})
		p.RootNode = 2
		return p
	}

	if err := checkSubjectNode(build(false)); err != nil {
		t.Fatalf("a subject node that does not read the subject is fine: %v", err)
	}
	err := checkSubjectNode(build(true))
	if err == nil {
		t.Fatal("a subject node defined in terms of the subject must be refused")
	}
	if !strings.Contains(err.Error(), "subject") {
		t.Fatalf("the error must name what is circular, got %v", err)
	}
}

// The fixture that exercises the clause must be invalid for that clause alone.
//
// Three engines decoded subject_node_circular.binpb and reported the same
// thing: it declares subject_node without declaring capability 11, which
// features.md section 11 freezes that field into, so check 25 refuses it on its
// own. Both checks answer invalid_ruleset, so an engine that never implemented
// the clause passes the case anyway - which is the engine the case exists to
// catch.
//
// This is the third fixture of that family, after message_key and
// program_expansion. The test states the property those lacked: remove the
// fault the case is named for, and nothing else must be wrong.
func TestTheCircularSubjectFixtureIsInvalidForThatReasonAlone(t *testing.T) {
	raw := circularSubjectFixture(t)

	if _, err := LoadRuleset(raw); err == nil {
		t.Fatal("the fixture loads; it is supposed to be refused")
	}

	// Repair the circularity and nothing else: the subject node stays, so the
	// capability it uses stays used. Clearing the node instead would remove both
	// faults at once and prove nothing, which is how the first version of this
	// test passed against the very fixture it was written to indict.
	bundle := decodeBundle(t, raw)
	if repaired := soundenSubjectNodes(bundle); repaired != 1 {
		t.Fatalf("repaired %d subject nodes, want exactly 1", repaired)
	}
	if _, err := LoadRuleset(encodeBundle(t, bundle)); err != nil {
		t.Fatalf("with a well founded subject node the fixture is still refused: %v\n"+
			"The case cannot distinguish an engine that implements check 15 from one "+
			"that does not: both answer invalid_ruleset, for different reasons.", err)
	}
}

// features.md section 11 freezes Program.subject_node into CAPTURES_AND_CALLS_V1,
// so a bundle declaring a subject node without declaring capability 11 uses a
// capability it never declared, and check 25 must refuse it.
//
// This loader derived capability 11 from the captures alone and ignored the
// subject node, which is why the fixture looked single-faulted here and
// double-faulted in three other engines. They read features.md; this read its
// own code.
func TestASubjectNodeUsesTheCapabilityThatFreezesIt(t *testing.T) {
	bundle := decodeBundle(t, circularSubjectFixture(t))
	// Well founded, so check 15 has nothing to say and only the undeclared
	// capability is left to refuse.
	soundenSubjectNodes(bundle)
	ids := bundle.GetRequiredFeatureIds()
	bundle.RequiredFeatureIds = nil
	for _, id := range ids {
		if id != 11 {
			bundle.RequiredFeatureIds = append(bundle.RequiredFeatureIds, id)
		}
	}

	_, err := LoadRuleset(encodeBundle(t, bundle))
	if err == nil {
		t.Fatal("a subject node without capability 11 declared must be refused by check 25")
	}
	if !strings.Contains(err.Error(), "CAPTURES_AND_CALLS_V1") {
		t.Fatalf("the refusal must name the undeclared capability, got %v", err)
	}
}

func circularSubjectFixture(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bundles", "subject_node_circular.binpb"))
	if err != nil {
		t.Fatalf("reading the fixture: %v", err)
	}
	return raw
}

func decodeBundle(t *testing.T, raw []byte) *irv1.RuleBundle {
	t.Helper()
	var bundle irv1.RuleBundle
	if err := proto.Unmarshal(raw, &bundle); err != nil {
		t.Fatalf("decoding the fixture: %v", err)
	}
	return &bundle
}

func encodeBundle(t *testing.T, bundle *irv1.RuleBundle) []byte {
	t.Helper()
	raw, err := proto.Marshal(bundle)
	if err != nil {
		t.Fatalf("re-encoding the fixture: %v", err)
	}
	return raw
}

// soundenSubjectNodes rebuilds every subject node on the country code, which no
// subject can be defined in terms of, and reports how many it touched.
func soundenSubjectNodes(bundle *irv1.RuleBundle) int {
	repaired := 0
	for _, p := range bundle.GetPrograms() {
		if p.SubjectNode == nil {
			continue
		}
		node := p.GetNodes()[p.GetSubjectNode()]
		node.InputNodes = nil
		node.Operation = &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
			Kind: irv1.StringOpKind_STRING_OP_KIND_COUNTRY_CODE,
		}}
		repaired++
	}
	return repaired
}
