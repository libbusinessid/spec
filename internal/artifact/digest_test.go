package artifact_test

import (
	"encoding/hex"
	"testing"

	"github.com/entid-org/spec/internal/artifact"
)

func TestSourceDigestIsOrderIndependent(t *testing.T) {
	a := []artifact.SourceEntry{
		{Path: "rules/b.hcl", Content: []byte("b\n")},
		{Path: "rules/a.hcl", Content: []byte("a\n")},
	}
	b := []artifact.SourceEntry{
		{Path: "rules/a.hcl", Content: []byte("a\n")},
		{Path: "rules/b.hcl", Content: []byte("b\n")},
	}
	da, err := artifact.SourceDigest(artifact.RulesDomain, a)
	if err != nil {
		t.Fatal(err)
	}
	db, err := artifact.SourceDigest(artifact.RulesDomain, b)
	if err != nil {
		t.Fatal(err)
	}
	if da != db {
		t.Fatal("the digest must not depend on the input order")
	}
}

func TestSourceDigestNormalizesLineEndings(t *testing.T) {
	unix, err := artifact.SourceDigest(artifact.RulesDomain,
		[]artifact.SourceEntry{{Path: "rules/a.hcl", Content: []byte("a\nb\n")}})
	if err != nil {
		t.Fatal(err)
	}
	windows, err := artifact.SourceDigest(artifact.RulesDomain,
		[]artifact.SourceEntry{{Path: "rules/a.hcl", Content: []byte("a\r\nb\r\n")}})
	if err != nil {
		t.Fatal(err)
	}
	mac, err := artifact.SourceDigest(artifact.RulesDomain,
		[]artifact.SourceEntry{{Path: "rules/a.hcl", Content: []byte("a\rb\r")}})
	if err != nil {
		t.Fatal(err)
	}
	if unix != windows || unix != mac {
		t.Fatal("CRLF and CR must normalize to LF")
	}
}

func TestSourceDigestDomainSeparation(t *testing.T) {
	entries := []artifact.SourceEntry{{Path: "a", Content: []byte("x")}}
	rules, _ := artifact.SourceDigest(artifact.RulesDomain, entries)
	conf, _ := artifact.SourceDigest(artifact.ConformanceDomain, entries)
	if rules == conf {
		t.Fatal("the two domains must produce different digests")
	}
}

func TestSourceDigestLengthPrefixIsUnambiguous(t *testing.T) {
	first, _ := artifact.SourceDigest(artifact.RulesDomain, []artifact.SourceEntry{
		{Path: "ab", Content: []byte("c")},
	})
	second, _ := artifact.SourceDigest(artifact.RulesDomain, []artifact.SourceEntry{
		{Path: "a", Content: []byte("bc")},
	})
	if first == second {
		t.Fatal("length prefixes must remove any ambiguity")
	}
}

func TestSourceDigestRejections(t *testing.T) {
	tests := []struct {
		name  string
		entry artifact.SourceEntry
	}{
		{"empty path", artifact.SourceEntry{Path: "", Content: nil}},
		{"absolute", artifact.SourceEntry{Path: "/etc/passwd", Content: nil}},
		{"parent", artifact.SourceEntry{Path: "rules/../x", Content: nil}},
		{"dot", artifact.SourceEntry{Path: "rules/./x", Content: nil}},
		{"backslash", artifact.SourceEntry{Path: `rules\x`, Content: nil}},
		{"invalid utf8 path", artifact.SourceEntry{Path: string([]byte{0xff}), Content: nil}},
		{"invalid utf8 content", artifact.SourceEntry{Path: "a", Content: []byte{0xff}}},
		{"bom", artifact.SourceEntry{Path: "a", Content: []byte{0xEF, 0xBB, 0xBF, 'x'}}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := artifact.SourceDigest(artifact.RulesDomain, []artifact.SourceEntry{tc.entry}); err == nil {
				t.Fatal("expected a rejection")
			}
		})
	}
}

func TestSourceDigestRejectsDuplicatePaths(t *testing.T) {
	_, err := artifact.SourceDigest(artifact.RulesDomain, []artifact.SourceEntry{
		{Path: "rules/a.hcl", Content: []byte("a")},
		{Path: "rules/a.hcl", Content: []byte("b")},
	})
	if err == nil {
		t.Fatal("duplicate paths must be refused")
	}
}

func TestSourceDigestIsStable(t *testing.T) {
	digest, err := artifact.SourceDigest(artifact.RulesDomain,
		[]artifact.SourceEntry{{Path: "rules/a.hcl", Content: []byte("x\n")}})
	if err != nil {
		t.Fatal(err)
	}
	// Frozen golden value, computed independently from this implementation
	// (see tools/canonical_stream.py). Any change to the canonical stream
	// breaks every published digest and must be an explicit, reviewed decision.
	// Moved once, at the entid rename: the domain tag became ENTID-SOURCE-V1
	// (ND-012). Both implementations were rerun and agree on this value.
	const want = "c5bda7845dadeec8c04077852a178bdfb2f83fd5fb54ca7656b2c412fd28ff0b"
	if got := hex.EncodeToString(digest[:]); got != want {
		t.Fatalf("canonical stream changed:\n got  %s\n want %s", got, want)
	}
}
