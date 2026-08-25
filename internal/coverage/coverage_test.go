package coverage_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/entid-org/spec/internal/coverage"
)

func TestParseLine(t *testing.T) {
	name, span, block, ok := coverage.ParseLine("pkg/file.go:12.3,15.4 2 7")
	if !ok || name != "pkg/file.go" || span != "12.3,15.4" || block.Statements != 2 || block.Count != 7 {
		t.Fatalf("unexpected parse: %q %q %+v %t", name, span, block, ok)
	}
	for _, bad := range []string{"", "no-colon", "pkg/file.go:12.3,15.4 2", "pkg/file.go:12.3,15.4 x 1", "pkg/file.go:12.3,15.4 1 y"} {
		if _, _, _, ok := coverage.ParseLine(bad); ok {
			t.Fatalf("%q must be rejected", bad)
		}
	}
}

func TestParseMergesDuplicateBlocks(t *testing.T) {
	profile := strings.Join([]string{
		"mode: atomic",
		"pkg/a.go:1.1,2.2 3 0",
		"pkg/a.go:3.1,4.2 1 5",
		// The same blocks reported by a second test binary.
		"pkg/a.go:1.1,2.2 3 4",
		"pkg/a.go:3.1,4.2 1 0",
		"",
	}, "\n")
	summary, err := coverage.Parse(strings.NewReader(profile))
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalStatements != 4 || summary.CoveredStatements != 4 {
		t.Fatalf("duplicate blocks must be merged: %+v", summary)
	}
	if summary.LineCoverage != 100 || summary.BlockCoverage != 100 {
		t.Fatalf("unexpected coverage: %+v", summary)
	}
}

func TestParsePartialCoverage(t *testing.T) {
	profile := "mode: set\npkg/a.go:1.1,2.2 4 1\npkg/b.go:1.1,2.2 4 0\n"
	summary, err := coverage.Parse(strings.NewReader(profile))
	if err != nil {
		t.Fatal(err)
	}
	if summary.LineCoverage != 50 || summary.BlockCoverage != 50 {
		t.Fatalf("unexpected coverage: %+v", summary)
	}
	if len(summary.WorstFiles) != 2 || summary.WorstFiles[0].File != "pkg/b.go" {
		t.Fatalf("unexpected ranking: %+v", summary.WorstFiles)
	}
}

func TestParseRejectsAMalformedProfile(t *testing.T) {
	if _, err := coverage.Parse(strings.NewReader("mode: set\nbroken line\n")); err == nil {
		t.Fatal("a malformed profile must be rejected")
	}
}

func TestReportThresholds(t *testing.T) {
	summary, err := coverage.Parse(strings.NewReader("mode: set\npkg/a.go:1.1,2.2 4 1\npkg/b.go:1.1,2.2 4 0\n"))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	failures := summary.Report(&buf, 95, 90)
	if len(failures) != 2 {
		t.Fatalf("expected two threshold failures, got %v", failures)
	}
	if !strings.Contains(buf.String(), "line coverage") {
		t.Fatalf("unexpected report: %q", buf.String())
	}
	if failures := summary.Report(&buf, 10, 10); len(failures) != 0 {
		t.Fatalf("expected no failure, got %v", failures)
	}
}

func TestParseEmptyProfile(t *testing.T) {
	summary, err := coverage.Parse(strings.NewReader(""))
	if err != nil {
		t.Fatal(err)
	}
	if summary.LineCoverage != 0 || summary.TotalBlocks != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
}

func TestWorstFilesIsCapped(t *testing.T) {
	var b strings.Builder
	b.WriteString("mode: set\n")
	for i := 0; i < 20; i++ {
		b.WriteString("pkg/f")
		b.WriteString(string(rune('a' + i)))
		b.WriteString(".go:1.1,2.2 1 0\n")
	}
	summary, err := coverage.Parse(strings.NewReader(b.String()))
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.WorstFiles) != 10 {
		t.Fatalf("expected ten worst files, got %d", len(summary.WorstFiles))
	}
}
