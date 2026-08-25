package artifact

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The writer of rules.lock must agree with the specification that every engine
// verifies against.
//
// This guard began by comparing two writers: tools/sync_engines.sh and
// .github/workflows/downstream.yml. They were two independent definitions and
// nothing compared them, so the lock gained conformance_jsonl_sha256 in one and
// not the other, the first release built a lock of seven digests where every
// engine verifies eight, and all four refused it.
//
// Since section 11.4 the release no longer pushes: each engine fetches the
// release and writes its own lock, so the second writer now lives in four
// repositories this test cannot reach. Comparing scripts is therefore replaced
// by comparing the one writer here against the normative list in engine.md
// section 16 -- which is the list the four engines implement. That is a
// stronger check than the one it replaces: it has one authority instead of two
// peers, and it fails here rather than in an engine's pull request.
//
// attestation_identity is the one legitimate difference. Only an attested
// release has an identity to record; the developer sync says so in a comment
// where the field would be.
func TestTheLockWriterAgreesWithTheSpecification(t *testing.T) {
	read := func(path string) string {
		raw, err := os.ReadFile(filepath.Join("..", "..", path))
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		return string(raw)
	}

	fields := func(path, text string, pattern *regexp.Regexp) []string {
		matches := pattern.FindAllStringSubmatch(text, -1)
		out := make([]string, 0, len(matches))
		for _, m := range matches {
			out = append(out, m[1])
		}
		if len(out) == 0 {
			t.Fatalf("%s declares no lock field; the pattern no longer matches", path)
		}
		return out
	}

	const contract = "docs/spec/engine.md"
	block := regexp.MustCompile("(?s)```lock-fields\n(.*?)```").FindStringSubmatch(read(contract))
	if block == nil {
		t.Fatalf("%s carries no lock-fields block; the normative list moved or was deleted", contract)
	}
	specified := fields(contract, block[1], regexp.MustCompile(`(?m)^([a-z0-9_]+)$`))

	const writer = "tools/sync_engines.sh"
	written := fields(writer, read(writer),
		regexp.MustCompile(`(?m)^([a-z0-9_]*sha256|rules_version|format_version|stability|source_commit) = `))

	if strings.Join(specified, " ") != strings.Join(written, " ") {
		t.Errorf("the lock writer and the specification disagree.\n  %s: %s\n  %s: %s\n"+
			"An engine verifies every field the lock declares, so a field one side "+
			"carries and the other does not is a release its engines refuse.",
			contract, strings.Join(specified, " "), writer, strings.Join(written, " "))
	}

	// The comment standing in for attestation_identity must still be there: an
	// absent field that says nothing is indistinguishable from one forgotten.
	if !strings.Contains(read(writer), "attestation_identity") {
		t.Error(writer + " no longer mentions attestation_identity, so a local sync " +
			"now produces a lock whose missing identity is silent rather than stated")
	}
}
