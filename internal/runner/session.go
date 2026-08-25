package runner

import (
	"errors"
	"fmt"
	"io"
	"iter"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	testeev1 "github.com/entid-org/spec/gen/go/entid/testee/v1"
)

// runSession drives one testee through the whole corpus and returns every
// difference it produced.
//
// The exchange is strictly synchronous: one request is written, one response is
// read, and only then does the next request go out. A testee therefore needs
// nothing more than a read-answer loop, and neither side can block on a full
// pipe.
//
// An error here means the run itself is void — a broken exchange, not a wrong
// answer. It is never a verdict of conformance.
func runSession(
	w io.Writer, r io.Reader,
	cases iter.Seq[*conformancev1.ConformanceCase],
	refusalOnly bool,
) ([]Diff, int, error) {
	reader := newFrameReader(r, defaultMaxFrame)
	var diffs []Diff
	sent := 0
	for c := range cases {
		sent++
		raw, err := proto.Marshal(requestFor(c))
		if err != nil {
			return nil, sent, fmt.Errorf("case %s: cannot encode the request: %w", c.GetId(), err)
		}
		if err := writeFrame(w, raw); err != nil {
			return nil, sent, fmt.Errorf("case %s: cannot send the request: %w", c.GetId(), err)
		}
		payload, err := reader.next()
		if errors.Is(err, errEOF) {
			// The stream cannot say how many cases are left without draining
			// it, and draining it to write a nicer message would defeat the
			// reason it is a stream. How far it got is the useful part.
			return nil, sent, fmt.Errorf("the testee stopped answering after %s, on case %d", c.GetId(), sent)
		}
		if err != nil {
			return nil, sent, fmt.Errorf("case %s: %w", c.GetId(), err)
		}
		var resp testeev1.TesteeResponse
		if err := proto.Unmarshal(payload, &resp); err != nil {
			return nil, sent, fmt.Errorf("case %s: the testee sent an undecodable response: %w", c.GetId(), err)
		}
		if resp.GetCaseId() != c.GetId() {
			return nil, sent, fmt.Errorf(
				"the exchange is desynchronized: %s was sent, the testee answered %q",
				c.GetId(), resp.GetCaseId())
		}
		diffs = append(diffs, compare(c, &resp, refusalOnly)...)
	}
	return diffs, sent, nil
}

// requestFor projects a case onto the wire, deliberately omitting the expected
// outcome: a testee never sees what is expected of it.
func requestFor(c *conformancev1.ConformanceCase) *testeev1.TesteeRequest {
	req := &testeev1.TesteeRequest{
		CaseId:      c.GetId(),
		Operation:   c.GetOperation(),
		Input:       c.GetInput(),
		CountryCode: c.CountryCode,
	}
	if k := c.GetKind(); k != "" {
		req.Kind = proto.String(k)
	}
	// Absence is meaningful and is sent as absence, never as an empty string.
	if pr := c.GetProfile(); pr != "" {
		req.Profile = proto.String(pr)
	}
	if p := c.GetRulesPayload(); len(p) > 0 {
		req.RulesPayload = p
	}
	return req
}
