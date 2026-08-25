package runner_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	"github.com/entid-org/spec/internal/runner"
)

// The fixture is a representative subset, not the whole corpus: it proves the
// mechanism end to end and stays stable. The full corpus is run by
// `make conformance`, which is what states conformance.
func loadCorpus(t *testing.T) []*conformancev1.ConformanceCase {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "conformance.binpb"))
	if err != nil {
		t.Fatalf("cannot read the fixture: %v", err)
	}
	var bundle conformancev1.ConformanceBundle
	if err := proto.Unmarshal(raw, &bundle); err != nil {
		t.Fatalf("corpus: %v", err)
	}
	return bundle.GetCases()
}

func referenceTestee(t *testing.T) []string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "testee")
	out, err := exec.Command("go", "build", "-o", bin,
		"github.com/entid-org/spec/cmd/conformance-testee").CombinedOutput()
	if err != nil {
		t.Fatalf("cannot build the reference testee: %v\n%s", err, out)
	}
	return []string{bin, "--bundle", filepath.Join("testdata", "rules.binpb")}
}

// End to end proof that framing, protocol and comparison agree.
func TestReferenceTesteeIsConformant(t *testing.T) {
	cases := loadCorpus(t)
	res, err := runner.Run(context.Background(), cases, runner.Options{Command: referenceTestee(t)})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !res.Conformant() {
		t.Fatalf("the reference testee must be conformant, %d differences: %v", len(res.Diffs), res.Diffs)
	}
	if res.Cases != len(cases) {
		t.Fatalf("every case must be answered: %d of %d", res.Cases, len(cases))
	}
}

// A runner that always reported conformance would pass the test above. Altering
// an expectation makes the very same testee wrong, so this fails unless
// differences are genuinely detected.
func TestAnAlteredExpectationIsCaught(t *testing.T) {
	cases := loadCorpus(t)
	altered := false
	for _, c := range cases {
		if r := c.GetExpected().GetValidationReport(); r != nil {
			r.CanonicalValue += "0"
			altered = true
			break
		}
	}
	if !altered {
		t.Fatal("the fixture must carry at least one validation case")
	}
	res, err := runner.Run(context.Background(), cases, runner.Options{Command: referenceTestee(t)})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.Conformant() {
		t.Fatal("an answer that departs from the corpus must never be reported as conformant")
	}
}

// A testee that answers nothing must void the run rather than shorten it into a
// pass.
func TestATesteeThatAnswersNothingVoidsTheRun(t *testing.T) {
	silent := filepath.Join(t.TempDir(), "silent")
	src := filepath.Join(t.TempDir(), "main.go")
	if err := os.WriteFile(src, []byte("package main\nfunc main() {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("go", "build", "-o", silent, src).CombinedOutput(); err != nil {
		t.Fatalf("cannot build the silent testee: %v\n%s", err, out)
	}
	_, err := runner.Run(context.Background(), loadCorpus(t), runner.Options{Command: []string{silent}})
	if err == nil {
		t.Fatal("a testee that stops answering must produce an error, never a verdict")
	}
	if !strings.Contains(err.Error(), "stopped answering") {
		t.Fatalf("the error should name the cause, got %v", err)
	}
}

func TestAMissingTesteeIsReported(t *testing.T) {
	_, err := runner.Run(context.Background(), loadCorpus(t), runner.Options{
		Command: []string{filepath.Join(t.TempDir(), "does-not-exist")},
	})
	if err == nil {
		t.Fatal("a testee that cannot be started must be reported")
	}
}
