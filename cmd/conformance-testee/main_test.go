package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/conformance/v1"
	testeev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/testee/v1"
	"github.com/libbusinessid/spec/internal/reference"
)

const bundlePath = "../../internal/runner/testdata/rules.binpb"

func loadEngine(t *testing.T) *reference.Engine {
	t.Helper()
	raw, err := os.ReadFile(bundlePath)
	if err != nil {
		t.Fatalf("fixture: %v", err)
	}
	e, err := reference.NewEngine(raw)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	return e
}

func TestAnswerEveryOperation(t *testing.T) {
	e := loadEngine(t)
	for name, op := range map[string]conformancev1.Operation{
		"canonicalize":    conformancev1.Operation_OPERATION_CANONICALIZE,
		"validate":        conformancev1.Operation_OPERATION_VALIDATE,
		"validate format": conformancev1.Operation_OPERATION_VALIDATE_FORMAT,
		"validate cksum":  conformancev1.Operation_OPERATION_VALIDATE_CHECKSUM,
	} {
		t.Run(name, func(t *testing.T) {
			resp := answer(e, &testeev1.TesteeRequest{
				CaseId: "c", Operation: op, Input: "552100554",
				Profile: proto.String("compatible"), Kind: proto.String("siren"),
			})
			if resp.GetCaseId() != "c" {
				t.Fatal("the case identifier must be echoed")
			}
			if resp.GetFailure() != nil {
				t.Fatalf("unexpected failure: %v", resp.GetFailure())
			}
		})
	}
}

func TestAnUnknownOperationIsAFailureNotAGuess(t *testing.T) {
	resp := answer(loadEngine(t), &testeev1.TesteeRequest{
		CaseId: "c", Operation: conformancev1.Operation_OPERATION_UNSPECIFIED,
	})
	if resp.GetFailure().GetKind() != testeev1.FailureKind_FAILURE_KIND_UNSUPPORTED_OPERATION {
		t.Fatalf("got %v", resp.GetResult())
	}
}

func TestWithoutABundleValidationFails(t *testing.T) {
	resp := answer(nil, &testeev1.TesteeRequest{
		CaseId: "c", Operation: conformancev1.Operation_OPERATION_VALIDATE,
	})
	if resp.GetFailure().GetKind() != testeev1.FailureKind_FAILURE_KIND_INTERNAL_ERROR {
		t.Fatalf("got %v", resp.GetResult())
	}
}

func TestLoadOutcomeReportsTheTypedError(t *testing.T) {
	t.Run("valid bundle", func(t *testing.T) {
		raw, err := os.ReadFile(bundlePath)
		if err != nil {
			t.Fatal(err)
		}
		if got := loadOutcome(raw); !got.GetAccepted() {
			t.Fatalf("a valid bundle must be accepted, got %v", got)
		}
	})
	t.Run("hostile bundle", func(t *testing.T) {
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "bundles", "truncated.binpb"))
		if err != nil {
			t.Skipf("fixture missing: %v", err)
		}
		got := loadOutcome(raw)
		if got.GetAccepted() {
			t.Fatal("a truncated bundle must never be accepted")
		}
		if got.GetEngineError() != "invalid_ruleset" && got.GetEngineError() != "incompatible_ruleset" {
			t.Fatalf("the refusal must carry a typed kind, got %q", got.GetEngineError())
		}
	})
}

func TestStatusOfMapsEveryStatus(t *testing.T) {
	for in, want := range map[reference.StepStatus]conformancev1.StepStatus{
		reference.StatusValid:       conformancev1.StepStatus_STEP_STATUS_VALID,
		reference.StatusInvalid:     conformancev1.StepStatus_STEP_STATUS_INVALID,
		reference.StatusUnsupported: conformancev1.StepStatus_STEP_STATUS_UNSUPPORTED,
		reference.StatusNotRun:      conformancev1.StepStatus_STEP_STATUS_NOT_RUN,
		reference.StepStatus("???"): conformancev1.StepStatus_STEP_STATUS_UNSPECIFIED,
	} {
		if got := statusOf(in); got != want {
			t.Fatalf("%q: got %v, want %v", in, got, want)
		}
	}
}

// The loop must answer each framed request and stop cleanly at end of input.
func TestRunAnswersFramedRequests(t *testing.T) {
	var in bytes.Buffer
	raw, err := proto.Marshal(&testeev1.TesteeRequest{
		CaseId: "c1", Operation: conformancev1.Operation_OPERATION_VALIDATE,
		Input: "552100554", Profile: proto.String("compatible"), Kind: proto.String("siren"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeFrame(&in, raw); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run(bundlePath, &in, &out); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.Len() == 0 {
		t.Fatal("a request must produce a response")
	}
}

func TestRunRejectsABadBundlePath(t *testing.T) {
	err := run(filepath.Join(t.TempDir(), "nope.binpb"), bytes.NewReader(nil), &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "cannot read the bundle") {
		t.Fatalf("got %v", err)
	}
}

func TestRunRejectsAnUndecodableRequest(t *testing.T) {
	var in bytes.Buffer
	if err := writeFrame(&in, []byte{0xff, 0xff, 0xff}); err != nil {
		t.Fatal(err)
	}
	err := run(bundlePath, &in, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "undecodable") {
		t.Fatalf("got %v", err)
	}
}

func TestReadFrameHandlesEmptyAndTruncated(t *testing.T) {
	t.Run("empty payload", func(t *testing.T) {
		var buf bytes.Buffer
		if err := writeFrame(&buf, nil); err != nil {
			t.Fatal(err)
		}
		got, err := readFrame(&buf)
		if err != nil || len(got) != 0 {
			t.Fatalf("got %v, %v", got, err)
		}
	})
	t.Run("truncated header", func(t *testing.T) {
		if _, err := readFrame(bytes.NewReader([]byte{1, 2})); err == nil {
			t.Fatal("a truncated header must be reported")
		}
	})
	t.Run("truncated body", func(t *testing.T) {
		if _, err := readFrame(bytes.NewReader([]byte{8, 0, 0, 0, 'a'})); err == nil {
			t.Fatal("a truncated body must be reported")
		}
	})
}

func TestWriteFrameReportsAFailingWriter(t *testing.T) {
	if err := writeFrame(brokenWriter{}, []byte("x")); err == nil {
		t.Fatal("a failing writer must be reported")
	}
}

type brokenWriter struct{}

func (brokenWriter) Write([]byte) (int, error) { return 0, os.ErrClosed }

func TestAnswerReportsAnUnknownKind(t *testing.T) {
	resp := answer(loadEngine(t), &testeev1.TesteeRequest{
		CaseId: "c", Operation: conformancev1.Operation_OPERATION_VALIDATE,
		Input: "x", Profile: proto.String("compatible"), Kind: proto.String("not-a-kind"),
	})
	if resp.GetValidationReport() == nil && resp.GetFailure() == nil {
		t.Fatal("an unknown kind must produce either a report or a typed failure")
	}
}
