package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	testeev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/testee/v1"
)

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errors.New("pipe is closed") }

func TestWriteFrameReportsAFailingWriter(t *testing.T) {
	if err := writeFrame(failingWriter{}, []byte("x")); err == nil {
		t.Fatal("a failing writer must be reported")
	}
}

func TestRunNeedsACommand(t *testing.T) {
	_, err := Run(context.Background(), nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "no testee command") {
		t.Fatalf("got %v", err)
	}
}

func TestWithStderrAttachesWhatTheTesteePrinted(t *testing.T) {
	base := errors.New("boom")
	if got := withStderr(base, "   "); !errors.Is(got, base) || got.Error() != base.Error() {
		t.Fatal("blank stderr must not be attached")
	}
	got := withStderr(base, "panic: nil map")
	if !strings.Contains(got.Error(), "nil map") || !errors.Is(got, base) {
		t.Fatalf("got %v", got)
	}
}

func TestShapeOfNamesEveryShape(t *testing.T) {
	for want, resp := range map[string]*testeev1.TesteeResponse{
		"a canonicalization": {Result: &testeev1.TesteeResponse_Canonicalization{
			Canonicalization: &testeev1.ObservedCanonicalization{}}},
		"a validation report": {Result: &testeev1.TesteeResponse_ValidationReport{
			ValidationReport: &testeev1.ObservedValidationReport{}}},
		"a load outcome": {Result: &testeev1.TesteeResponse_Load{Load: &testeev1.ObservedLoad{}}},
		"nothing":        {},
	} {
		if got := shapeOf(resp); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestBothCountryCodesAbsentIsAMatch(t *testing.T) {
	var diffs []Diff
	add := func(f, w, g string) { diffs = append(diffs, Diff{Field: f, Want: w, Got: g}) }
	cmpOptional(add, "countryCode", nil, nil)
	if len(diffs) != 0 {
		t.Fatalf("two absent values match, got %v", diffs)
	}
}

// A testee that emits garbage instead of a response must void the run.
func TestAnUndecodableResponseVoidsTheRun(t *testing.T) {
	st := &scriptedTestee{t: t, answer: func(*testeev1.TesteeRequest) *testeev1.TesteeResponse { return nil }}
	// answer returns nil, so nothing is written: emulate garbage directly.
	if err := writeFrame(&st.out, []byte{0xff, 0xff, 0xff, 0xff}); err != nil {
		t.Fatal(err)
	}
	_, _, err := runSession(st, st, slices.Values(twoCases()), false)
	if err == nil || !strings.Contains(err.Error(), "undecodable") {
		t.Fatalf("got %v", err)
	}
}

// A run that outlives its budget must say so. The session sees the pipe close
// and reports that the testee stopped answering, which reads as a crashed
// engine and sends the reader looking for a bug that is not there. Register
// sweeps made this routine: the SIRET sweep runs for about ten minutes, which
// is exactly the default budget.
func TestATimeoutIsReportedAsATimeout(t *testing.T) {
	slow := filepath.Join(t.TempDir(), "slow.sh")
	script := "#!/bin/sh\nsleep 30\n"
	if err := os.WriteFile(slow, []byte(script), 0o700); err != nil { //nolint:gosec // a script this test runs
		t.Fatal(err)
	}
	_, err := Run(context.Background(), twoCases(), Options{
		Command: []string{slow},
		Timeout: 100 * time.Millisecond,
	})
	if err == nil {
		t.Fatal("a run that outlives its budget must fail")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "budget") {
		t.Fatalf("the error must name the timeout, got: %v", err)
	}
}
