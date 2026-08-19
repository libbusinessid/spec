package runner

import (
	"errors"
	"fmt"
	"io"

	"google.golang.org/protobuf/proto"

	conformancev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/conformance/v1"
	testeev1 "github.com/libbusinessid/spec/gen/go/libbusinessid/testee/v1"
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
func runSession(w io.Writer, r io.Reader, cases []*conformancev1.ConformanceCase) ([]Diff, error) {
	reader := newFrameReader(r, defaultMaxFrame)
	var diffs []Diff
	for _, c := range cases {
		raw, err := proto.Marshal(requestFor(c))
		if err != nil {
			return nil, fmt.Errorf("case %s: cannot encode the request: %w", c.GetId(), err)
		}
		if err := writeFrame(w, raw); err != nil {
			return nil, fmt.Errorf("case %s: cannot send the request: %w", c.GetId(), err)
		}
		payload, err := reader.next()
		if errors.Is(err, errEOF) {
			left := len(cases) - indexOf(cases, c)
			return nil, fmt.Errorf("the testee stopped answering after %s; %d cases were left unanswered",
				c.GetId(), left)
		}
		if err != nil {
			return nil, fmt.Errorf("case %s: %w", c.GetId(), err)
		}
		var resp testeev1.TesteeResponse
		if err := proto.Unmarshal(payload, &resp); err != nil {
			return nil, fmt.Errorf("case %s: the testee sent an undecodable response: %w", c.GetId(), err)
		}
		if resp.GetCaseId() != c.GetId() {
			return nil, fmt.Errorf("the exchange is desynchronized: %s was sent, the testee answered %q", c.GetId(), resp.GetCaseId())
		}
		diffs = append(diffs, compare(c, &resp)...)
	}
	return diffs, nil
}

// requestFor projects a case onto the wire, deliberately omitting the expected
// outcome: a testee never sees what is expected of it.
func requestFor(c *conformancev1.ConformanceCase) *testeev1.TesteeRequest {
	req := &testeev1.TesteeRequest{
		CaseId:      c.GetId(),
		Operation:   c.GetOperation(),
		Input:       c.GetInput(),
		Profile:     c.GetProfile(),
		CountryCode: c.CountryCode,
	}
	if k := c.GetKind(); k != "" {
		req.Kind = proto.String(k)
	}
	if p := c.GetRulesPayload(); len(p) > 0 {
		req.RulesPayload = p
	}
	return req
}

func indexOf(cases []*conformancev1.ConformanceCase, target *conformancev1.ConformanceCase) int {
	for i, c := range cases {
		if c == target {
			return i
		}
	}
	return 0
}
