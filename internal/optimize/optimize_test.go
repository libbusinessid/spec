package optimize_test

import (
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/optimize"
)

func stringNode(text string, inputs ...uint32) *irv1.Node {
	return &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		InputNodes: inputs,
		Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
			Kind: irv1.StringOpKind_STRING_OP_KIND_CONSTANT,
			Text: &text,
		}},
	}
}

func TestStructuralKeyIdentifiesObservationallyIdenticalNodes(t *testing.T) {
	a := stringNode("FR")
	b := stringNode("FR")
	if optimize.StructuralKey(a) != optimize.StructuralKey(b) {
		t.Fatal("two identical nodes must share their key")
	}
}

func TestStructuralKeySeparatesDifferentParameters(t *testing.T) {
	if optimize.StructuralKey(stringNode("FR")) == optimize.StructuralKey(stringNode("BE")) {
		t.Fatal("a different constant must produce a different key")
	}
	if optimize.StructuralKey(stringNode("FR")) == optimize.StructuralKey(stringNode("FR", 1)) {
		t.Fatal("different operands must produce a different key")
	}
}

func TestStructuralKeySeparatesMessageKeysAndReasonCodes(t *testing.T) {
	build := func(reason irv1.ReasonCode, key string) *irv1.Node {
		return &irv1.Node{
			OutputType: irv1.ValueType_VALUE_TYPE_ASSERTION,
			InputNodes: []uint32{0},
			Operation: &irv1.Node_AssertionOperation{AssertionOperation: &irv1.AssertionOperation{
				Kind:       irv1.AssertionOpKind_ASSERTION_OP_KIND_REQUIRE,
				ReasonCode: &reason,
				MessageKey: &key,
			}},
		}
	}
	a := build(irv1.ReasonCode_REASON_CODE_INVALID_LENGTH, "a.length")
	b := build(irv1.ReasonCode_REASON_CODE_INVALID_LENGTH, "b.length")
	c := build(irv1.ReasonCode_REASON_CODE_INVALID_CHARACTERS, "a.length")
	if optimize.StructuralKey(a) == optimize.StructuralKey(b) {
		t.Fatal("deduplication must never merge two different message keys")
	}
	if optimize.StructuralKey(a) == optimize.StructuralKey(c) {
		t.Fatal("deduplication must never merge two different reason codes")
	}
}

func TestStructuralKeyOfAnEmptyNode(t *testing.T) {
	first := &irv1.Node{}
	second := &irv1.Node{}
	if optimize.StructuralKey(first) != optimize.StructuralKey(second) {
		t.Fatal("two empty nodes must share their key")
	}
	if optimize.StructuralKey(first) == optimize.StructuralKey(stringNode("FR")) {
		t.Fatal("an empty node must not share the key of a real node")
	}
}

func TestStructuralKeyOfAnUnencodableNode(t *testing.T) {
	// A proto3 string field must hold valid UTF-8: an invalid constant cannot
	// be encoded, and the node must then never be deduplicated with any other.
	invalid := string([]byte{0xff, 0xfe})
	node := &irv1.Node{
		OutputType: irv1.ValueType_VALUE_TYPE_STRING,
		Operation: &irv1.Node_StringOperation{StringOperation: &irv1.StringOperation{
			Kind: irv1.StringOpKind_STRING_OP_KIND_CONSTANT,
			Text: &invalid,
		}},
	}
	key := optimize.StructuralKey(node)
	if key == optimize.StructuralKey(stringNode("FR")) {
		t.Fatal("an unencodable node must not share a key with a valid one")
	}
	if !strings.HasPrefix(key, "\x00unencodable\x00") {
		t.Fatalf("unexpected key %q", key)
	}
}
