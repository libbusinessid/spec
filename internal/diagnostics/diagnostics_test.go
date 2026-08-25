package diagnostics_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/entid-org/spec/internal/diagnostics"
)

func TestPositionString(t *testing.T) {
	tests := []struct {
		name string
		pos  diagnostics.Position
		want string
	}{
		{"empty", diagnostics.Position{}, ""},
		{"file only", diagnostics.Position{File: "a.hcl"}, "a.hcl"},
		{"file and line", diagnostics.Position{File: "a.hcl", Line: 3}, "a.hcl:3"},
		{"full", diagnostics.Position{File: "a.hcl", Line: 3, Column: 7}, "a.hcl:3:7"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.pos.String(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBagSortsDeterministically(t *testing.T) {
	b := diagnostics.New()
	b.Errorf(diagnostics.Position{File: "b.hcl", Line: 1, Column: 1}, "E2", "second")
	b.Errorf(diagnostics.Position{File: "a.hcl", Line: 9, Column: 1}, "E1", "later line")
	b.Errorf(diagnostics.Position{File: "a.hcl", Line: 2, Column: 5}, "E1", "earlier line")
	b.Warnf(diagnostics.Position{File: "a.hcl", Line: 2, Column: 1}, "W1", "warning")

	got := b.Sorted()
	want := []string{"warning", "earlier line", "later line", "second"}
	if len(got) != len(want) {
		t.Fatalf("got %d diagnostics, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Message != want[i] {
			t.Fatalf("position %d: got %q, want %q", i, got[i].Message, want[i])
		}
	}
	if !b.HasErrors() || b.Len() != 4 {
		t.Fatal("bag state is wrong")
	}
}

func TestBagSortsByCodeThenMessage(t *testing.T) {
	b := diagnostics.New()
	b.Errorf(diagnostics.Position{}, "E2", "z")
	b.Errorf(diagnostics.Position{}, "E1", "b")
	b.Errorf(diagnostics.Position{}, "E1", "a")
	got := b.Sorted()
	if got[0].Message != "a" || got[1].Message != "b" || got[2].Code != "E2" {
		t.Fatalf("unexpected order: %+v", got)
	}
}

func TestWarningsAreNotErrors(t *testing.T) {
	b := diagnostics.New()
	b.Warnf(diagnostics.Position{File: "x"}, "W", "careful")
	if b.HasErrors() {
		t.Fatal("a warning must not be an error")
	}
	if b.Err() != nil {
		t.Fatal("Err must be nil without an error diagnostic")
	}
}

func TestSuggestionRendering(t *testing.T) {
	b := diagnostics.New()
	b.Suggestf(diagnostics.Position{File: "a.hcl", Line: 1, Column: 2}, "E9", "use format.fr.siren", "unknown symbol %q", "format.fr.sirene")
	var buf bytes.Buffer
	if err := b.WriteText(&buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "a.hcl:1:2: error [E9] unknown symbol \"format.fr.sirene\"") {
		t.Fatalf("missing diagnostic line: %s", out)
	}
	if !strings.Contains(out, "suggestion: use format.fr.siren") {
		t.Fatalf("missing suggestion: %s", out)
	}
}

func TestWriteJSON(t *testing.T) {
	b := diagnostics.New()
	b.Errorf(diagnostics.Position{File: "a.hcl", Line: 4, Column: 1}, "E1", "boom")
	var buf bytes.Buffer
	if err := b.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Diagnostics []diagnostics.Diagnostic `json:"diagnostics"`
	}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Diagnostics) != 1 || doc.Diagnostics[0].Code != "E1" || doc.Diagnostics[0].Position.Line != 4 {
		t.Fatalf("unexpected json: %s", buf.String())
	}
}

func TestWriteJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := diagnostics.New().WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), `"diagnostics": []`) {
		t.Fatalf("empty bag must render an empty array: %s", buf.String())
	}
}

func TestErrorAggregation(t *testing.T) {
	b := diagnostics.New()
	b.Errorf(diagnostics.Position{File: "a"}, "E1", "one")
	err := b.Err()
	if err == nil || !strings.Contains(err.Error(), "one") {
		t.Fatalf("unexpected error %v", err)
	}
	b.Errorf(diagnostics.Position{File: "b"}, "E2", "two")
	err = b.Err()
	if err == nil || !strings.HasPrefix(err.Error(), "2 errors, first: ") {
		t.Fatalf("unexpected error %v", err)
	}
	var empty diagnostics.Error
	if empty.Error() != "no diagnostic" {
		t.Fatalf("unexpected error %q", empty.Error())
	}
}

func TestExtendAndDefaultSeverity(t *testing.T) {
	a := diagnostics.New()
	a.Add(diagnostics.Diagnostic{Code: "E", Message: "m"})
	if !a.HasErrors() {
		t.Fatal("severity must default to error")
	}
	b := diagnostics.New()
	b.Extend(a)
	b.Extend(nil)
	if b.Len() != 1 {
		t.Fatalf("got %d", b.Len())
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) { return 0, errBoom }

var errBoom = &boomError{}

type boomError struct{}

func (*boomError) Error() string { return "boom" }

func TestWriteTextPropagatesWriterError(t *testing.T) {
	b := diagnostics.New()
	b.Errorf(diagnostics.Position{}, "E", "m")
	if err := b.WriteText(failingWriter{}); err == nil {
		t.Fatal("expected the writer error to be propagated")
	}
}
