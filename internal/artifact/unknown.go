package artifact

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// findUnknownFields walks the whole message tree and returns the path of the
// first message carrying an unknown Protobuf field.
//
// Relying only on required_feature_ids would be unsafe: a forged bundle could
// carry an unknown field and deliberately omit the matching capability.
func findUnknownFields(m proto.Message) (string, bool) {
	return walkUnknown(m.ProtoReflect(), string(m.ProtoReflect().Descriptor().FullName()))
}

func walkUnknown(m protoreflect.Message, path string) (string, bool) {
	if len(m.GetUnknown()) > 0 {
		return path, true
	}
	found := ""
	ok := false
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList() && fd.Kind() == protoreflect.MessageKind:
			list := v.List()
			for i := 0; i < list.Len(); i++ {
				if p, bad := walkUnknown(list.Get(i).Message(), fmt.Sprintf("%s.%s[%d]", path, fd.Name(), i)); bad {
					found, ok = p, true
					return false
				}
			}
		case fd.IsMap():
			// The IR never uses maps; a map field would already be unknown.
			return true
		case fd.Kind() == protoreflect.MessageKind || fd.Kind() == protoreflect.GroupKind:
			if p, bad := walkUnknown(v.Message(), fmt.Sprintf("%s.%s", path, fd.Name())); bad {
				found, ok = p, true
				return false
			}
		}
		return true
	})
	return found, ok
}
