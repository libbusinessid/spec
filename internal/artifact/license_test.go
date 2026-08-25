package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// The SBOM declares Apache-2.0 for this repository, so the LICENSE file must be
// Apache-2.0 -- verbatim, not a paraphrase of it.
//
// It was a paraphrase. The file carried all nine sections and the END OF TERMS
// heading, but the clauses were rewritten and shortened: 5157 bytes against the
// canonical 11358. Section 9 dropped "or Derivative Works thereof" and the whole
// sentence bounding a Contributor's exposure. GitHub reported the repository as
// "Other" rather than Apache-2.0, which is how it was noticed; the licence being
// materially different from the one the SBOM asserts is the actual defect.
//
// The pinned digest is the file published at https://www.apache.org/licenses/
// LICENSE-2.0.txt, cross-checked byte for byte against an independent copy in
// the Go module cache and against the three engine repositories that already
// carried it correctly.
func TestTheLicenseIsVerbatimApache2(t *testing.T) {
	const want = "cfc7749b96f63bd31c3c42b5c471bf756814053e847c10f3eb003417bc523d30"
	raw, err := os.ReadFile(filepath.Join("..", "..", "LICENSE"))
	if err != nil {
		t.Fatalf("reading LICENSE: %v", err)
	}
	sum := sha256.Sum256(raw)
	if got := hex.EncodeToString(sum[:]); got != want {
		t.Errorf("LICENSE is not the verbatim Apache 2.0 text.\n got  %s (%d bytes)\n want %s (11358 bytes)\n"+
			"The SBOM declares Apache-2.0 and consumers rely on that declaration, so "+
			"the file may not be edited -- not even to insert a copyright line.",
			got, len(raw), want)
	}
}
