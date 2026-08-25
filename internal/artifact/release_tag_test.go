package artifact

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

// The release must refuse a tag that does not name the rules version.
//
// The README states the rule -- "One release is one rulesVersion and one
// immutable Git tag v<rulesVersion>" -- and nothing enforced it. Practice
// drifted immediately: v0.1.0 carried rules 2026.08.32 and v0.1.1 carried
// 2026.08.33. Tag and version were two independent axes, so nothing stopped two
// tags from delivering one rulesVersion, which is the one collision rules.lock
// cannot represent: it identifies a bundle by that version and its digest.
//
// The guard lives in the workflow because that is where a tag first meets the
// version. This test exists so it cannot be deleted quietly.
func TestTheReleaseRefusesATagThatDoesNotNameTheRulesVersion(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("reading release.yml: %v", err)
	}
	workflow := string(raw)

	for _, want := range []struct {
		what    string
		pattern *regexp.Regexp
	}{
		{"it derives the expected tag from RULES_VERSION",
			regexp.MustCompile(`expected="v\$\(cat RULES_VERSION\)"`)},
		{"it compares the pushed tag against it",
			regexp.MustCompile(`\[ "\$\{GITHUB_REF_NAME\}" != "\$\{expected\}" \]`)},
		{"it fails rather than warning",
			regexp.MustCompile(`(?s)GITHUB_REF_NAME\}" != "\$\{expected\}".*?exit 1`)},
	} {
		if !want.pattern.MatchString(workflow) {
			t.Errorf("release.yml no longer proves that %s.\n"+
				"Without it a tag may deliver a rulesVersion it does not name, and two "+
				"tags may deliver one version -- a collision rules.lock cannot express.",
				want.what)
		}
	}
}
