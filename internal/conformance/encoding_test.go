package conformance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// No conformance case may claim invalid_encoding.
//
// ir.md section 5 step 1 refuses an input that is not valid UTF-8. The corpus
// cannot carry such an input: a proto3 string is valid UTF-8 by definition, on
// the wire between the runner and a testee and in the compiled corpus alike.
// Nor is there one portable malformed value to carry - an invalid byte where
// strings are bytes, an unpaired surrogate where they are UTF-16 code units,
// and nothing at all in Swift, whose String is always well formed.
//
// A case added anyway would not carry what its author meant: the malformed
// bytes are replaced on the way in, and the case would then assert the reason
// against a value that is perfectly well formed - green, and proving the
// opposite of what it claims. Three fixtures of this repository have already
// failed to test what they said they tested; this one is refused before it can
// join them.
//
// The step is pinned instead by a native test in each engine, which ir.md now
// requires, and here by internal/reference/encoding_test.go.
func TestNoConformanceCaseClaimsInvalidEncoding(t *testing.T) {
	root := repoRoot(t)
	corpus := filepath.Join(root, "conformance")

	err := filepath.WalkDir(corpus, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || filepath.Ext(path) != ".jsonl" {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for i, line := range strings.Split(string(raw), "\n") {
			if !strings.Contains(line, "invalid_encoding") {
				continue
			}
			rel, _ := filepath.Rel(root, path)
			t.Errorf("%s:%d claims invalid_encoding.\n"+
				"The corpus cannot carry an input that is not valid UTF-8, so the case "+
				"would assert that reason against a well formed value and pass for the "+
				"wrong reason. Pin the step with a native test instead, as ir.md "+
				"section 5 step 1 requires.", rel, i+1)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the corpus: %v", err)
	}
}
