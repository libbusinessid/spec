// Package optimize implements the optional, documented structural
// deduplication of identical IR sub-graphs.
//
// Deduplication is opt-out: `entidc compile --optimize=false` disables it
// and an equivalence test proves that both modes execute identically. The
// structural key never merges two nodes with a different assertion order,
// reason code or message key, because those values are part of the key.
package optimize

import (
	"google.golang.org/protobuf/proto"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
)

// unencodableKey is used for the nodes the runtime cannot encode. It is unique
// per call, so such a node is never deduplicated with any other.
const unencodablePrefix = "\x00unencodable\x00"

// StructuralKey returns the documented deduplication key of a node.
//
// The key is the deterministic Protobuf encoding of the node itself, which
// already contains the operation, every parameter and the indices of the
// operands. Two nodes therefore share a key if and only if they are
// observationally identical, operands included.
func StructuralKey(node *irv1.Node) string {
	encoded, err := proto.MarshalOptions{Deterministic: true}.Marshal(node)
	if err != nil {
		return unencodablePrefix + err.Error()
	}
	return "\x01" + string(encoded)
}
