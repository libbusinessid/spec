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
	// engine.md itself, which is where the contradiction actually was: its
	// minimal API list still offered engineFromRules and registryLookup, three
	// sections after forbidding both. Auditing only the per-language contracts
	// missed the document they all defer to.
	//
	// And spec.md, which none of these guards read for four audits. It still
	// said an engine MAY embed the bundle and interpret it, in the very section
	// that documents rules.lock, which is the sentence a TypeScript engine was
	// built on before section 1.2 existed.
	contracts = append(contracts,
		filepath.Join("..", "..", "docs", "spec", "engine.md"),
		filepath.Join("..", "..", "docs", "spec", "spec.md"))

	// A declaration lives in a code block; the sentences explaining why there is
	// none live in prose, and must stay. The audit first matched fromRules and
	// RegistryProvider only, and missed engineFromRules and registryLookup in
	// the minimal API list - the same commitments under other spellings.
	declaration := regexp.MustCompile(
		`(?im)\b(init\(rules|from_?rules|fromBytes|engineFromRules|registryLookup|RegistryProvider)\b`)

	for _, path := range contracts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			text := readDoc(t, path)
			if m := declaration.FindString(codeBlocksOf(text)); m != "" {
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
				// The bundle as a resource of the shipped package is the same
				// mistake in layout form: it makes every caller carry data the
				// generated code makes pointless, and implies a decoder to read
				// it. Three contracts had it in their recommended tree.
				"Resources/businessid-rules",
				"resources/businessid-rules",
				"assets/businessid-rules",
				// The same permission stated as prose rather than as a
				// signature or a path. engine.md still said the engine MAY be
				// built from bytes and listed loading a bundle from memory,
				// three sections after 1.2 forbade it, and the Swift contract
				// still made the bundle an SPM resource one line after saying
				// it is not one.
				"Bundle.module",
				"construction depuis des bytes",
				"bundle fourni en mémoire",
				"charger la ressource",
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
			if !strings.Contains(text, "section 1.2") {
				t.Errorf("%s never points at engine.md section 1.2.\n"+
					"An implementer reading only this document would not learn that "+
					"the bundle is read by a generator rather than interpreted.",
					filepath.Base(path))
			}
		})
	}
}

// The hand written documents must not spell out how many load checks there are.
//
// A written count is a fact that goes stale the day a check is added, and adding
// check 14 did exactly that: four documents kept saying twenty four while ir.md
// enumerated twenty five, including the sentence that delegates authority to
// ir.md - which named a number that document no longer had. ir.md computes its
// own count; the others delegate.
func TestHandWrittenDocumentsDoNotCountTheLoadChecks(t *testing.T) {
	for _, name := range []string{"engine.md", "spec.md", "engine-go.md",
		"engine-swift.md", "engine-kotlin.md", "engine-typescript.md"} {
		path := filepath.Join("..", "..", "docs", "spec", name)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		t.Run(name, func(t *testing.T) {
			text := readDoc(t, path)
			want := loadCheckCount(t)
			for spelled, n := range map[string]int{
				"vingt-trois": 23, "vingt-quatre": 24, "vingt-cinq": 25, "vingt-six": 26,
			} {
				if !strings.Contains(text, spelled+" contrôles") {
					continue
				}
				if n != want {
					t.Errorf("%s says %q contrôles where ir.md enumerates %d.\n"+
						"Either fix the number or delegate without counting.",
						name, spelled, want)
				}
			}
		})
	}
}

// loadCheckCount reads the enumeration out of the generated document.
func loadCheckCount(t *testing.T) int {
	t.Helper()
	ir := readDoc(t, filepath.Join("..", "..", "docs", "ir.md"))
	start := strings.Index(ir, "1. binary size")
	end := strings.Index(ir, "A size, structural")
	if start < 0 || end < 0 {
		t.Fatal("ir.md no longer carries the load check enumeration")
	}
	return len(regexp.MustCompile(`(?m)^\s*\d+\. `).FindAllString(ir[start:end], -1))
}

// codeBlocksOf returns only what sits inside fenced code blocks, which is where
// a document declares an API rather than discusses one.
func codeBlocksOf(text string) string {
	var out strings.Builder
	inside := false
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inside = !inside
			continue
		}
		if inside {
			out.WriteString(line)
			out.WriteString("\n")
		}
	}
	return out.String()
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

// No contract may ask an engine to deliver a registry, or to ship the bundle as
// a packaged resource.
//
// The runtime loader guard above reads signatures and prose paragraphs, and both
// were clean. The residues had retreated into the checklists nobody re-reads: a
// deliverables line still listing "interface registre", a test list still asking
// for "resource loading depuis JAR assemblé", a README list still promising a
// "moteur personnalisé". A checklist is what an implementer works from, so a
// stale line there costs a week of somebody's work - which is exactly what it
// cost, twice.
func TestNoEngineContractAsksForARegistryOrAPackagedBundle(t *testing.T) {
	contracts, err := filepath.Glob(filepath.Join("..", "..", "docs", "spec", "engine-*.md"))
	if err != nil || len(contracts) == 0 {
		t.Fatalf("no engine contract found: %v", err)
	}
	contracts = append(contracts, filepath.Join("..", "..", "docs", "spec", "engine.md"))

	forbidden := map[string]string{
		// engine.md section 10 defers the registry and reserves its shape. A
		// contract that lists one among the deliverables asks for a public type,
		// and a public type is a commitment SemVer freezes.
		"interface registre":        "engine.md section 10 defers the registry; no engine delivers a type for it",
		"provider registre injecté": "engine.md section 10 defers the registry; no engine delivers a type for it",
		"provider registre concret": "engine.md section 10 defers the registry; no engine delivers a type for it", //nolint:misspell // French, not a typo of concert
		// The bundle is an input to the generator. Every phrasing below asks the
		// published package to carry it, which implies a decoder to read it.
		"ressource SPM":             "the bundle is an input to the generator, not a resource of the package",
		"resource loading":          "the bundle is an input to the generator, not a resource of the package",
		"packaging de la ressource": "the bundle is an input to the generator, not a resource of the package",
		"bundle embarqué":           "the bundle is an input to the generator, not a resource of the package",
		// A custom ruleset goes through the generator, at build time. Offering a
		// "custom engine" in a README is the byte factory under another name.
		"moteur personnalisé": "a custom ruleset goes through the generator, at build time",
		"depuis BINPB":        "the corpus drives the emitted code; only the generator decodes",
	}

	for _, path := range contracts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			text := readDoc(t, path)
			for phrase, why := range forbidden {
				if strings.Contains(text, phrase) {
					t.Errorf("%s asks for %q.\n%s.", filepath.Base(path), phrase, why)
				}
			}
		})
	}
}

// Every engine contract must point at the shared runner, and none may ask for a
// comparator of its own.
//
// spec.md section 8.7 has always said the comparison logic exists exactly once,
// and gave the reason: an engine that reads the expected results itself can
// declare conformance by comparing too weakly. But none of the five contracts -
// the documents an implementer actually works from - ever said where the runner
// comes from, and no release existed to download one. Two engines wrote their
// own. The root cause is the one the checklists had: a rule stated in a document
// nobody reads while implementing.
func TestEveryEngineContractPointsAtTheSharedRunner(t *testing.T) {
	contracts, err := filepath.Glob(filepath.Join("..", "..", "docs", "spec", "engine-*.md"))
	if err != nil || len(contracts) == 0 {
		t.Fatalf("no engine contract found: %v", err)
	}
	contracts = append(contracts, filepath.Join("..", "..", "docs", "spec", "engine.md"))

	for _, path := range contracts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			text := readDoc(t, path)
			if !strings.Contains(text, "conformance-runner@") {
				t.Errorf("%s never says where the conformance runner comes from.\n"+
					"An implementer reading only this document writes one, and then the "+
					"engine and its judge are the same code.", filepath.Base(path))
			}
			// The step that no case can carry. A contract that stays silent about
			// it leaves a public reason code with no coverage anywhere.
			if !strings.Contains(text, "invalid_encoding") {
				t.Errorf("%s never requires a native test for invalid_encoding.\n"+
					"No conformance case can carry an ill formed input, so the branch is "+
					"covered by that test or by nothing.", filepath.Base(path))
			}
		})
	}
}

// The tooling may not put the bundle where the doctrine forbids it either.
//
// open_downstream_pr.sh copied it into Sources/BusinessID/Resources and
// src/main/resources on every release, carrying the exact phrases the contract
// guard forbids -- while that guard read the documents and never the scripts
// that act on them. A rule stated in prose and contradicted by the automation
// is the automation's reading that wins, because the automation is what runs.
func TestTheToolingDoesNotShipTheBundleAsAResource(t *testing.T) {
	scripts, err := filepath.Glob(filepath.Join("..", "..", "tools", "*.sh"))
	if err != nil || len(scripts) == 0 {
		t.Fatalf("no tooling found: %v", err)
	}
	for _, path := range scripts {
		t.Run(filepath.Base(path), func(t *testing.T) {
			text := readDoc(t, path)
			for _, phrase := range []string{
				"Resources/businessid-rules",
				"resources/businessid-rules",
				"assets/businessid-rules",
				"internal/rules/businessid-rules",
				"src/rules/businessid-rules",
			} {
				if strings.Contains(text, phrase) {
					t.Errorf("%s writes the bundle to %q.\n"+
						"That is the bundle as a resource of the published package, which "+
						"engine.md section 1.2 forbids: it belongs under spec/, as an input "+
						"to the engine's generator.", filepath.Base(path), phrase)
				}
			}
		})
	}
}
