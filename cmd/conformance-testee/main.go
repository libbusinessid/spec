// Command conformance-testee answers the conformance runner using the
// reference interpreter.
//
// It exists for two reasons: it exercises the runner itself, and it is a
// worked example of the protocol for anyone implementing an engine. It is
// deliberately small — a testee reads a request, calls its engine, and reports
// what it observed. It never reads the corpus and never sees an expectation.
package main

import (
	"bufio"
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	testeev1 "github.com/entid-org/spec/gen/go/entid/testee/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/reference"
)

func main() {
	bundlePath := flag.String("bundle", "", "path to the rules bundle to answer with")
	flag.Parse()
	if err := run(*bundlePath, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(bundlePath string, stdin io.Reader, stdout io.Writer) error {
	var engine *reference.Engine
	if bundlePath != "" {
		raw, err := os.ReadFile(filepath.Clean(bundlePath))
		if err != nil {
			return fmt.Errorf("cannot read the bundle: %w", err)
		}
		engine, err = reference.NewEngine(raw)
		if err != nil {
			return fmt.Errorf("cannot load the bundle: %w", err)
		}
	}

	in := bufio.NewReader(stdin)
	out := bufio.NewWriter(stdout)
	defer func() { _ = out.Flush() }()

	for {
		payload, err := readFrame(in)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		var req testeev1.TesteeRequest
		if err := proto.Unmarshal(payload, &req); err != nil {
			return fmt.Errorf("undecodable request: %w", err)
		}
		raw, err := proto.Marshal(answer(engine, &req))
		if err != nil {
			return fmt.Errorf("cannot encode the response: %w", err)
		}
		if err := writeFrame(out, raw); err != nil {
			return err
		}
		if err := out.Flush(); err != nil {
			return err
		}
	}
}

func answer(engine *reference.Engine, req *testeev1.TesteeRequest) *testeev1.TesteeResponse {
	resp := &testeev1.TesteeResponse{CaseId: req.GetCaseId()}

	if req.GetOperation() == conformancev1.Operation_OPERATION_LOAD_RULESET {
		resp.Result = &testeev1.TesteeResponse_Load{Load: loadOutcome(req.GetRulesPayload())}
		return resp
	}
	if engine == nil {
		return failure(resp, testeev1.FailureKind_FAILURE_KIND_INTERNAL_ERROR, "no bundle was supplied")
	}

	input := reference.Input{Kind: req.GetKind(), Value: req.GetInput(), CountryCode: req.CountryCode}
	opts := reference.Options{Profile: reference.Profile(req.GetProfile())}

	switch req.GetOperation() {
	case conformancev1.Operation_OPERATION_CANONICALIZE:
		got, err := engine.Canonicalize(input, opts)
		if err != nil {
			return failure(resp, testeev1.FailureKind_FAILURE_KIND_INTERNAL_ERROR, err.Error())
		}
		resp.Result = &testeev1.TesteeResponse_Canonicalization{Canonicalization: &testeev1.ObservedCanonicalization{
			Kind:           got.Kind,
			CanonicalValue: got.CanonicalValue,
			CountryCode:    got.CountryCode,
			Status:         statusOf(got.Status),
			ReasonCode:     got.ReasonCode,
		}}
	case conformancev1.Operation_OPERATION_VALIDATE,
		conformancev1.Operation_OPERATION_VALIDATE_FORMAT,
		conformancev1.Operation_OPERATION_VALIDATE_CHECKSUM:
		var got reference.ValidationReport
		var err error
		switch req.GetOperation() {
		case conformancev1.Operation_OPERATION_VALIDATE:
			got, err = engine.Validate(input, opts)
		case conformancev1.Operation_OPERATION_VALIDATE_FORMAT:
			got, err = engine.ValidateFormat(input, opts)
		default:
			got, err = engine.ValidateChecksum(input, opts)
		}
		if err != nil {
			return failure(resp, testeev1.FailureKind_FAILURE_KIND_INTERNAL_ERROR, err.Error())
		}
		resp.Result = &testeev1.TesteeResponse_ValidationReport{ValidationReport: &testeev1.ObservedValidationReport{
			Kind:           got.Kind,
			CanonicalValue: got.CanonicalValue,
			CountryCode:    got.CountryCode,
			Format:         step(got.Format),
			Checksum:       step(got.Checksum),
		}}
	default:
		return failure(resp, testeev1.FailureKind_FAILURE_KIND_UNSUPPORTED_OPERATION, req.GetOperation().String())
	}
	return resp
}

func loadOutcome(payload []byte) *testeev1.ObservedLoad {
	if _, err := reference.NewEngine(payload); err != nil {
		var typed *artifact.Error
		if errors.As(err, &typed) {
			return &testeev1.ObservedLoad{Accepted: false, EngineError: string(typed.Kind)}
		}
		// An untyped rejection is still a rejection, but the contract is the
		// typed kind, so report what was seen rather than guessing one.
		return &testeev1.ObservedLoad{Accepted: false, EngineError: err.Error()}
	}
	return &testeev1.ObservedLoad{Accepted: true}
}

func step(s reference.StepResult) *testeev1.ObservedStep {
	out := &testeev1.ObservedStep{
		Status:     statusOf(s.Status),
		ReasonCode: s.ReasonCode,
	}
	// Absent when the result came from before any rule assertion, which is what
	// engine.md section 11.2 requires: dispatch, input_too_long, not_requested
	// and the not_run reasons carry no key.
	if s.MessageKey != nil {
		out.MessageKey = s.MessageKey
	}
	return out
}

// statusOf maps the engine's status onto the wire enum. An unknown status is
// reported as UNSPECIFIED rather than guessed, so the runner sees a difference
// instead of a plausible wrong answer.
func statusOf(s reference.StepStatus) conformancev1.StepStatus {
	switch s {
	case reference.StatusValid:
		return conformancev1.StepStatus_STEP_STATUS_VALID
	case reference.StatusInvalid:
		return conformancev1.StepStatus_STEP_STATUS_INVALID
	case reference.StatusUnsupported:
		return conformancev1.StepStatus_STEP_STATUS_UNSUPPORTED
	case reference.StatusNotRun:
		return conformancev1.StepStatus_STEP_STATUS_NOT_RUN
	default:
		return conformancev1.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}

func failure(resp *testeev1.TesteeResponse, kind testeev1.FailureKind, detail string) *testeev1.TesteeResponse {
	resp.Result = &testeev1.TesteeResponse_Failure{Failure: &testeev1.TesteeFailure{Kind: kind, Detail: detail}}
	return resp
}

func readFrame(r io.Reader) ([]byte, error) {
	var head [4]byte
	if _, err := io.ReadFull(r, head[:]); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, io.EOF
		}
		return nil, fmt.Errorf("cannot read the frame header: %w", err)
	}
	size := binary.LittleEndian.Uint32(head[:])
	if size == 0 {
		return []byte{}, nil
	}
	payload := make([]byte, size)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, fmt.Errorf("cannot read a frame of %d bytes: %w", size, err)
	}
	return payload, nil
}

func writeFrame(w io.Writer, payload []byte) error {
	var head [4]byte
	binary.LittleEndian.PutUint32(head[:], uint32(len(payload)))
	if _, err := w.Write(head[:]); err != nil {
		return err
	}
	_, err := w.Write(payload)
	return err
}
