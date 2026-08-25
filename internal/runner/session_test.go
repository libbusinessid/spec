package runner

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	testeev1 "github.com/entid-org/spec/gen/go/entid/testee/v1"
)

// scriptedTestee answers each request with what answer returns, so a session
// can be driven without spawning a process.
type scriptedTestee struct {
	in     bytes.Buffer
	out    bytes.Buffer
	answer func(*testeev1.TesteeRequest) *testeev1.TesteeResponse
	t      *testing.T
}

func (s *scriptedTestee) Write(p []byte) (int, error) {
	n, err := s.in.Write(p)
	if err != nil {
		return n, err
	}
	for {
		fr := newFrameReader(bytes.NewReader(s.in.Bytes()), defaultMaxFrame)
		payload, ferr := fr.next()
		if ferr != nil {
			// Not an error: the runner has not finished writing this frame yet.
			return n, nil //nolint:nilerr // waiting for more bytes is not a failure
		}
		s.in.Next(4 + len(payload))
		var req testeev1.TesteeRequest
		if err := proto.Unmarshal(payload, &req); err != nil {
			s.t.Fatalf("the runner emitted an undecodable request: %v", err)
		}
		resp := s.answer(&req)
		if resp == nil {
			continue
		}
		raw, err := proto.Marshal(resp)
		if err != nil {
			s.t.Fatalf("marshal: %v", err)
		}
		if err := writeFrame(&s.out, raw); err != nil {
			s.t.Fatalf("writeFrame: %v", err)
		}
	}
}

func (s *scriptedTestee) Read(p []byte) (int, error) {
	if s.out.Len() == 0 {
		return 0, io.EOF
	}
	return s.out.Read(p)
}

func twoCases() []*conformancev1.ConformanceCase {
	a := validationCase(proto.String("FR"))
	b := validationCase(proto.String("FR"))
	b.Id = "c2"
	return []*conformancev1.ConformanceCase{a, b}
}

func echoAnswer(req *testeev1.TesteeRequest) *testeev1.TesteeResponse {
	r := validationResponse("552100554")
	r.CaseId = req.GetCaseId()
	return r
}

func TestSessionReportsNoDiffWhenTheTesteeIsCorrect(t *testing.T) {
	st := &scriptedTestee{answer: echoAnswer, t: t}
	diffs, _, err := runSession(st, st, slices.Values(twoCases()), false)
	if err != nil {
		t.Fatalf("runSession: %v", err)
	}
	if len(diffs) != 0 {
		t.Fatalf("a correct testee must produce no diff, got %v", diffs)
	}
}

func TestSessionReportsEveryWrongCase(t *testing.T) {
	st := &scriptedTestee{t: t, answer: func(req *testeev1.TesteeRequest) *testeev1.TesteeResponse {
		r := validationResponse("wrong")
		r.CaseId = req.GetCaseId()
		return r
	}}
	diffs, _, err := runSession(st, st, slices.Values(twoCases()), false)
	if err != nil {
		t.Fatalf("runSession: %v", err)
	}
	if len(diffs) != 2 {
		t.Fatalf("both cases must be reported, got %d", len(diffs))
	}
}

// A testee that echoes the wrong identifier has desynchronized; scoring the
// answer against the wrong case would silently corrupt the verdict.
func TestSessionRefusesADesynchronizedExchange(t *testing.T) {
	st := &scriptedTestee{t: t, answer: func(*testeev1.TesteeRequest) *testeev1.TesteeResponse {
		r := validationResponse("552100554")
		r.CaseId = "not-the-case-we-sent"
		return r
	}}
	_, _, err := runSession(st, st, slices.Values(twoCases()), false)
	if err == nil || !strings.Contains(err.Error(), "answered") {
		t.Fatalf("a mismatched case identifier must abort the run, got %v", err)
	}
}

func TestSessionRefusesATesteeThatStopsAnswering(t *testing.T) {
	n := 0
	st := &scriptedTestee{t: t, answer: func(req *testeev1.TesteeRequest) *testeev1.TesteeResponse {
		n++
		if n > 1 {
			return nil
		}
		return echoAnswer(req)
	}}
	_, _, err := runSession(st, st, slices.Values(twoCases()), false)
	if err == nil {
		t.Fatal("a testee that stops answering must be reported, not counted as passing")
	}
}
