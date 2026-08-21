package artifact_test

import (
	"os"
	"path/filepath"
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
