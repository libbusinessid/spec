package artifact_test

import (
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/proto"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
)

func fixtureSeeds(t testing.TB) [][]byte {
	t.Helper()
	dir := filepath.Join(repoRootFor(t), "testdata", "bundles")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var out [][]byte
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, data)
	}
	return out
}

func repoRootFor(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("cannot locate the repository root")
	return ""
}

// FuzzLoadRuleset feeds arbitrary bytes to the defensive loader. A rejection is
// always fine; a panic, a hang or an unbounded allocation is not.
func FuzzLoadRuleset(f *testing.F) {
	for _, seed := range fixtureSeeds(f) {
		f.Add(seed)
	}
	f.Add([]byte{})
	f.Add([]byte{0x08, 0x01})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff})
	f.Fuzz(func(t *testing.T, data []byte) {
		rules, err := artifact.LoadRuleset(data)
		if err != nil {
			var typed *artifact.Error
			if !errorsAs(err, &typed) {
				t.Fatalf("the loader must return a typed error, got %T", err)
			}
			if typed.Kind != artifact.ErrInvalid && typed.Kind != artifact.ErrIncompatible {
				t.Fatalf("unexpected error kind %q", typed.Kind)
			}
			return
		}
		if rules == nil || rules.Bundle == nil {
			t.Fatal("an accepted bundle must be indexed")
		}
	})
}

// FuzzMutateBundle mutates a valid bundle at the message level, exercising node
// indices, enum values, arithmetic bounds and dispatch tables.
func FuzzMutateBundle(f *testing.F) {
	base, err := os.ReadFile(filepath.Join(repoRootFor(f), "testdata", "bundles", "minimal_valid.binpb"))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(base, uint32(0), uint32(0), int64(0))
	f.Add(base, uint32(1), uint32(3), int64(-1))
	f.Add(base, uint32(2), uint32(99), int64(1<<62))
	f.Fuzz(func(t *testing.T, data []byte, selector, index uint32, value int64) {
		bundle := &irv1.RuleBundle{}
		if err := proto.Unmarshal(data, bundle); err != nil {
			t.Skip("not a bundle")
		}
		mutateBundle(bundle, selector, index, value)
		mutated, err := proto.Marshal(bundle)
		if err != nil {
			t.Skip("unencodable mutation")
		}
		if _, err := artifact.LoadRuleset(mutated); err != nil {
			var typed *artifact.Error
			if !errorsAs(err, &typed) {
				t.Fatalf("the loader must return a typed error, got %T", err)
			}
		}
	})
}

func mutateBundle(b *irv1.RuleBundle, selector, index uint32, value int64) {
	switch selector % 12 {
	case 0:
		b.FormatVersion = uint32(value)
	case 1:
		b.RequiredFeatureIds = append(b.RequiredFeatureIds, uint32(value))
	case 2:
		b.SourceDigest = make([]byte, index%64)
	case 3:
		if len(b.GetPrograms()) > 0 {
			p := b.GetPrograms()[int(index)%len(b.GetPrograms())]
			p.RootNode = uint32(value)
		}
	case 4:
		if len(b.GetPrograms()) > 0 {
			p := b.GetPrograms()[int(index)%len(b.GetPrograms())]
			if len(p.GetNodes()) > 0 {
				n := p.GetNodes()[int(index)%len(p.GetNodes())]
				n.InputNodes = append(n.InputNodes, uint32(value))
			}
		}
	case 5:
		if len(b.GetPrograms()) > 0 {
			p := b.GetPrograms()[int(index)%len(b.GetPrograms())]
			if len(p.GetNodes()) > 0 {
				p.GetNodes()[int(index)%len(p.GetNodes())].Operation = nil
			}
		}
	case 6:
		if len(b.GetPrograms()) > 0 {
			p := b.GetPrograms()[int(index)%len(b.GetPrograms())]
			p.Kind = irv1.ProgramKind(value % 5)
		}
	case 7:
		if len(b.GetIdentifiers()) > 0 {
			d := b.GetIdentifiers()[int(index)%len(b.GetIdentifiers())]
			d.DefaultProfile = "unknown"
		}
	case 8:
		if len(b.GetDispatchers()) > 0 {
			d := b.GetDispatchers()[int(index)%len(b.GetDispatchers())]
			d.Targets = append(d.Targets, &irv1.DispatchTarget{IdentifierDefinitionId: uint32(value)})
		}
	case 9:
		if len(b.GetPrograms()) > 0 {
			p := b.GetPrograms()[int(index)%len(b.GetPrograms())]
			for _, n := range p.GetNodes() {
				if op := n.GetIntegerOperation(); op != nil {
					op.Modulus = &value
				}
				if op := n.GetPredicateOperation(); op != nil {
					length := uint32(value)
					op.Length = &length
				}
			}
		}
	case 10:
		if len(b.GetPrograms()) > 0 {
			p := b.GetPrograms()[int(index)%len(b.GetPrograms())]
			for _, n := range p.GetNodes() {
				if op := n.GetIntegerOperation(); op != nil {
					op.Weights = append(op.Weights, value)
				}
			}
		}
	default:
		b.RulesVersion = ""
	}
}

func errorsAs(err error, target **artifact.Error) bool {
	typed, ok := err.(*artifact.Error) //nolint:errorlint // the loader returns this exact type
	if ok {
		*target = typed
	}
	return ok
}
