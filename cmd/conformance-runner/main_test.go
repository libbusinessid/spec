package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/libbusinessid/spec/internal/runner"
)

const fixtures = "../../internal/runner/testdata"

func testeeCommand(t *testing.T) []string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "testee")
	out, err := exec.Command("go", "build", "-o", bin,
		"github.com/libbusinessid/spec/cmd/conformance-testee").CombinedOutput()
	if err != nil {
		t.Fatalf("cannot build the testee: %v\n%s", err, out)
	}
	return []string{bin, "--bundle", filepath.Join(fixtures, "rules.binpb")}
}

func TestRunRejectsIncompleteInvocations(t *testing.T) {
	for name, tc := range map[string]struct {
		corpus  string
		command []string
		want    string
	}{
		"no corpus":   {"", []string{"x"}, "--corpus is required"},
		"no command":  {filepath.Join(fixtures, "conformance.binpb"), nil, "after --"},
		"bad corpus":  {filepath.Join(fixtures, "nope.binpb"), []string{"x"}, "cannot read the corpus"},
		"not aBundle": {filepath.Join(fixtures, "rules.binpb"), []string{"x"}, ""},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := run(tc.corpus, tc.command, 0, "")
			if err == nil {
				t.Fatal("an incomplete invocation must fail")
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("got %v, want it to mention %q", err, tc.want)
			}
		})
	}
}

func TestOperationByName(t *testing.T) {
	if _, ok := operationByName("validate"); !ok {
		t.Fatal("validate must resolve")
	}
	if _, ok := operationByName("VALIDATE"); !ok {
		t.Fatal("the name must be case insensitive")
	}
	if _, ok := operationByName("unspecified"); ok {
		t.Fatal("the zero value must not be selectable, it would run nothing")
	}
	if _, ok := operationByName("teleport"); ok {
		t.Fatal("an unknown operation must be refused")
	}
}

func TestUnknownOperationIsRefused(t *testing.T) {
	_, err := run(filepath.Join(fixtures, "conformance.binpb"), testeeCommand(t), 0, "teleport")
	if err == nil || !strings.Contains(err.Error(), "unknown operation") {
		t.Fatalf("got %v", err)
	}
}

// Restricting to an operation that no case uses must fail rather than report a
// vacuous success on zero cases.
// A restricted run answers correctly yet must never claim conformance.
func TestARestrictedRunIsNeverAVerdict(t *testing.T) {
	ok, err := run(filepath.Join(fixtures, "conformance.binpb"), testeeCommand(t), 0, "load_ruleset")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if ok {
		t.Fatal("a run restricted to one operation must not report conformance")
	}
}

func TestAFullRunOfTheFixtureIsConformant(t *testing.T) {
	ok, err := run(filepath.Join(fixtures, "conformance.binpb"), testeeCommand(t), 0, "")
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !ok {
		t.Fatal("the reference testee must be conformant on the fixture")
	}
}

func TestReportNamesTheOutcome(*testing.T) {
	report(runner.Result{Cases: 3}, "2026.08.0", false)
	report(runner.Result{Cases: 3, Diffs: []runner.Diff{{CaseID: "c1", Field: "kind", Want: "a", Got: "b"}}}, "2026.08.0", false)
	report(runner.Result{Cases: 1}, "2026.08.0", true)
}
