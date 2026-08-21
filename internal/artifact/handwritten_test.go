package artifact_test

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
)

// spec.md and engine.md are written by hand while ir.md and features.md are
// generated, so the hand written pair drifts and nothing notices. The
// TypeScript engine found four such drifts at once, one of them material: the
// load order in engine.md still put the unknown field scan before the version
// checks, which turns a version gap into a suspected forgery.
//
// This holds the parts that can be compared mechanically. It cannot judge
// prose, and does not try.
func TestHandWrittenDocumentsAgreeWithTheSchema(t *testing.T) {
	codes := reasonCodesOfSchema()

	for _, doc := range []string{"engine.md", "spec.md"} {
		t.Run(doc, func(t *testing.T) {
			text := readDoc(t, filepath.Join("..", "..", "docs", "spec", doc))
			for _, code := range codes {
				if !strings.Contains(text, code) {
					t.Errorf("%s never mentions the reason code %q.\n"+
						"rules.proto defines it, so an engine written from this "+
						"document alone would not know it exists.", doc, code)
				}
			}
		})
	}
}

// The load order is the one drift that changes an answer rather than a
// document, so it is checked on its own.
func TestEngineDocumentPutsTheVersionChecksFirst(t *testing.T) {
	text := readDoc(t, filepath.Join("..", "..", "docs", "spec", "engine.md"))
	version := strings.Index(text, "`format_version` supportée")
	features := strings.Index(text, "`required_feature_ids` connues")
	unknown := strings.Index(text, "absence de champ inconnu")
	if version < 0 || features < 0 || unknown < 0 {
		t.Fatal("engine.md no longer carries the three checks this test compares")
	}
	if version >= features || features >= unknown {
		t.Fatalf("engine.md orders the load checks as version=%d features=%d unknown=%d.\n"+
			"ir.md section 10 puts the version checks first, and the order is the "+
			"difference between reporting a version gap and reporting a forgery.",
			version, features, unknown)
	}
}

// Every engine contract must point at the rule that the bundle is read by a
// generator, never interpreted at runtime, and none may declare a factory
// taking bundle bytes.
//
// This is the defect that cost a whole engine. spec.md permitted interpretation
// in as many words, engine.md said nothing either way, and two contracts
// required a factory over bytes - which forces the loader and the opcode
// machine into every caller, an interpreter by another name. The TypeScript
// engine followed the normative document and built one.
func TestEveryEngineContractRefusesARuntimeLoader(t *testing.T) {
	contracts, err := filepath.Glob(filepath.Join("..", "..", "docs", "spec", "engine-*.md"))
	if err != nil || len(contracts) == 0 {
		t.Fatalf("no engine contract found: %v", err)
	}
	// A declaration, not the sentence explaining why there is none.
	declaration := regexp.MustCompile(
		`(?m)^\s*(public |static |func )?(init\(rules|fromRules|from_rules|fromBytes)\b`)

	for _, path := range contracts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			text := readDoc(t, path)
			if m := declaration.FindString(text); m != "" {
				t.Errorf("%s declares %q.\n"+
					"A factory taking bundle bytes forces the whole loader and the "+
					"whole opcode machine into every caller, which is an interpreter.\n"+
					"engine.md section 1.2 requires a generator.",
					filepath.Base(path), strings.TrimSpace(m))
			}
			// A signature is not the only way a document asks for an
			// interpreter. The Go contract stated the generator rule in its
			// opening and then, three paragraphs later, required the library to
			// embed the bundle, decode the IR defensively and build an engine
			// from custom bytes. The first version of this test looked for
			// declarations and saw none of it.
			for _, phrase := range []string{
				"embarquer `businessid-rules.binpb`",
				"bytes personnalisés",
				"go:embed",
			} {
				if strings.Contains(text, phrase) {
					t.Errorf("%s asks for %q.\n"+
						"That is an interpreter stated in prose rather than in a "+
						"signature: the bundle is an input to the generator, not "+
						"data in the published package.", filepath.Base(path), phrase)
				}
			}
			// A registry lookup carries an API token, so it must never be
			// reachable from a browser. engine.md section 10 defers the whole
			// surface, and a public type is a commitment SemVer freezes: a
			// contract that declares one now cannot take it back.
			if strings.Contains(text, "RegistryProvider") {
				t.Errorf("%s declares RegistryProvider.\n"+
					"engine.md section 10 defers the registry: a public type is a "+
					"commitment, and this one has not been specified yet.",
					filepath.Base(path))
			}
			if !strings.Contains(text, "section 1.2") {
				t.Errorf("%s never points at engine.md section 1.2.\n"+
					"An implementer reading only this document would not learn that "+
					"the bundle is read by a generator rather than interpreted.",
					filepath.Base(path))
			}
		})
	}
}

func readDoc(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read %s: %v", path, err)
	}
	return string(raw)
}

// reasonCodesOfSchema returns the business reason codes as the schema spells
// them in a document: lower case, without the enum prefix.
func reasonCodesOfSchema() []string {
	var out []string
	for name := range irv1.ReasonCode_name {
		symbol := irv1.ReasonCode(name).String()
		if symbol == "REASON_CODE_UNSPECIFIED" {
			continue
		}
		out = append(out, strings.ToLower(strings.TrimPrefix(symbol, "REASON_CODE_")))
	}
	slices.Sort(out)
	return out
}
