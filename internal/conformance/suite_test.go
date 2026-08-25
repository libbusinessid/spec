package conformance_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/conformance"
	"github.com/entid-org/spec/internal/reference"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("cannot locate the repository root")
	return ""
}

// TestReferenceInterpreterPassesTheWholeSuite is the normative check of the
// repository: the reference interpreter must satisfy every reviewed case.
func TestReferenceInterpreterPassesTheWholeSuite(t *testing.T) {
	root := repoRoot(t)
	rules, bag := artifact.CompileRules(filepath.Join(root, "rules"), artifact.CompileOptions{
		RulesVersion: "2026.08.0", Optimize: true,
	})
	if bag.HasErrors() {
		var sb strings.Builder
		for _, d := range bag.Sorted() {
			sb.WriteString(d.String() + "\n")
		}
		t.Fatalf("the rules do not compile:\n%s", sb.String())
	}
	suite, cbag := conformance.Compile(filepath.Join(root, "conformance"), conformance.CompileOptions{
		RulesVersion:  rules.Bundle.GetRulesVersion(),
		FormatVersion: rules.Bundle.GetFormatVersion(),
		FixtureRoot:   filepath.Join(root, "testdata"),
	})
	if cbag.HasErrors() {
		var sb strings.Builder
		for _, d := range cbag.Sorted() {
			sb.WriteString(d.String() + "\n")
		}
		t.Fatalf("the conformance corpus does not compile:\n%s", sb.String())
	}
	engine := reference.NewEngineFromRuleset(rules.Ruleset)
	report := conformance.Run(engine, suite.Bundle)
	if len(report.Failures) > 0 {
		for _, f := range report.Failures {
			t.Errorf("%s", f.String())
		}
		t.Fatalf("%d/%d conformance cases passed", report.Passed, report.Total)
	}
	if report.Total < 100 {
		t.Fatalf("the corpus is suspiciously small: %d cases", report.Total)
	}
	t.Logf("%d conformance cases passed", report.Total)
}

// TestReferenceBundlePassesItsSuite proves that the minimal reference bundle
// published with every release, and the suite that accompanies it, are
// consistent. An engine can start from those two files alone.
func TestReferenceBundlePassesItsSuite(t *testing.T) {
	root := repoRoot(t)
	rulesBytes, err := os.ReadFile(filepath.Join(root, "testdata", "bundles", "minimal_valid.binpb"))
	if err != nil {
		t.Fatal(err)
	}
	suiteBytes, err := os.ReadFile(filepath.Join(root, "testdata", "bundles", "minimal_conformance.binpb"))
	if err != nil {
		t.Fatal(err)
	}
	engine, err := reference.NewEngine(rulesBytes)
	if err != nil {
		t.Fatalf("the reference bundle must load: %v", err)
	}
	suite := &conformancev1.ConformanceBundle{}
	if err := proto.Unmarshal(suiteBytes, suite); err != nil {
		t.Fatal(err)
	}
	if suite.GetRulesVersion() != engine.Ruleset().RulesVersion() {
		t.Fatalf("the reference suite targets %q, the bundle is %q",
			suite.GetRulesVersion(), engine.Ruleset().RulesVersion())
	}
	report := conformance.Run(engine, suite)
	for _, f := range report.Failures {
		t.Errorf("%s", f.String())
	}
	if report.Passed != report.Total || report.Total == 0 {
		t.Fatalf("%d/%d reference cases passed", report.Passed, report.Total)
	}
}
