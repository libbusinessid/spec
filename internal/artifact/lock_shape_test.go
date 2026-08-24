package artifact

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The two writers of rules.lock must agree on its fields.
//
// tools/sync_engines.sh writes the lock a developer syncs with, and
// .github/workflows/downstream.yml writes the one a release delivers. They were
// two independent definitions and nothing compared them: the lock gained
// conformance_jsonl_sha256 in one and not the other, so the first release built
// a lock of seven digests where every engine verifies eight, and all four
// refused it.
//
// attestation_identity is the one legitimate difference. Only an attested
// release has an identity to record; the developer sync says so in a comment
// where the field would be.
func TestBothWritersOfTheLockAgree(t *testing.T) {
	fields := func(path string, pattern *regexp.Regexp) []string {
		raw, err := os.ReadFile(filepath.Join("..", "..", path))
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		matches := pattern.FindAllStringSubmatch(string(raw), -1)
		out := make([]string, 0, len(matches))
		for _, m := range matches {
			out = append(out, m[1])
		}
		if len(out) == 0 {
			t.Fatalf("%s writes no lock field; the pattern no longer matches", path)
		}
		return out
	}

	sync := fields("tools/sync_engines.sh", regexp.MustCompile(`(?m)^([a-z0-9_]*sha256|rules_version|format_version|stability|source_commit) = `))
	release := fields(".github/workflows/downstream.yml", regexp.MustCompile(`printf '([a-z0-9_]+) = `))

	// The release adds the attested identity and nothing else.
	trimmed := make([]string, 0, len(release))
	for _, f := range release {
		if f != "attestation_identity" {
			trimmed = append(trimmed, f)
		}
	}
	if strings.Join(sync, " ") != strings.Join(trimmed, " ") {
		t.Errorf("the two writers of rules.lock disagree.\n  sync:    %s\n  release: %s\n"+
			"An engine verifies every field the lock declares, so a field one writer "+
			"carries and the other does not is a release its engines refuse.",
			strings.Join(sync, " "), strings.Join(trimmed, " "))
	}
}
