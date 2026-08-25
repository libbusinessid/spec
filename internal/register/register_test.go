package register_test

import (
	"strings"
	"testing"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	"github.com/entid-org/spec/internal/register"
)

// A register sweep asks one question of every identifier an issuer has handed
// out: is it accepted. The answer must be yes for all of them, because the
// register is the authority on what it issues, and section 1.2 makes refusing a
// valid identifier the worst defect this project recognises.
func TestSweepReadsEveryIdentifier(t *testing.T) {
	csv := "CompanyName, CompanyNumber,CompanyStatus\n" +
		"\"ACME, LTD\", 08209948,Active\n" +
		"BETA LTD,SC606050,Active\n" +
		"GAMMA LTD,IP10067R,Liquidation\n"

	def := register.Definition{
		ID: "gb-companies-house", Kind: "company_number",
		Country: "GB", Column: "CompanyNumber",
	}
	var got []string
	for c, err := range register.Sweep(strings.NewReader(csv), def) {
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got = append(got, c.GetInput())
		report := c.GetExpected().GetValidationReport()
		if report.GetFormat().GetStatus() != conformancev1.StepStatus_STEP_STATUS_VALID {
			t.Fatalf("%s: a register entry must be expected valid", c.GetInput())
		}
		if c.GetKind() != "company_number" || c.GetCountryCode() != "GB" {
			t.Fatalf("%s: wrong routing", c.GetInput())
		}
		// The question a sweep asks is whether the identifier is refused, and a
		// rule refuses just as firmly through the checksum as through the
		// format. Asking only for the format would never run the checksum at
		// all, and the sweep would report a clean run over rules it never
		// exercised.
		if c.GetOperation() != conformancev1.Operation_OPERATION_VALIDATE {
			t.Fatalf("%s: a sweep must ask for the whole validation, got %s",
				c.GetInput(), c.GetOperation())
		}
		// A register establishes existence and nothing else, so a sweep case
		// must not claim a canonical value or a checksum it cannot know.
		if report.GetCanonicalValue() != "" || report.GetChecksum() != nil {
			t.Fatalf("%s: a sweep case asserts more than the register supports", c.GetInput())
		}
	}
	want := []string{"08209948", "SC606050", "IP10067R"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// Case identifiers have to be unique and stable, because a difference is
// reported by identifier and a sweep reports thousands of them.
func TestSweepIdentifiersAreUniqueAndStable(t *testing.T) {
	csv := "CompanyNumber\n00000001\n00000002\n"
	def := register.Definition{ID: "gb", Kind: "k", Country: "GB", Column: "CompanyNumber"}
	seen := map[string]bool{}
	for c, err := range register.Sweep(strings.NewReader(csv), def) {
		if err != nil {
			t.Fatal(err)
		}
		if seen[c.GetId()] {
			t.Fatalf("duplicate case id %q", c.GetId())
		}
		seen[c.GetId()] = true
		if !strings.Contains(c.GetId(), "gb") {
			t.Fatalf("case id %q does not name its register", c.GetId())
		}
	}
	if len(seen) != 2 {
		t.Fatalf("got %d cases", len(seen))
	}
}

// A sweep reports a refusal by case identifier and nothing else, so the
// identifier has to carry the value. "row 2 was refused" sends the operator
// back to a five million line file; "ZZ999999 was refused" is the answer.
func TestSweepCaseIdentifierCarriesTheValue(t *testing.T) {
	def := register.Definition{ID: "gb", Kind: "k", Country: "GB", Column: "N"}
	for c, err := range register.Sweep(strings.NewReader("N\nZZ999999\n"), def) {
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(c.GetId(), "ZZ999999") {
			t.Fatalf("case id %q does not carry the value it is about", c.GetId())
		}
	}
}

// A dump whose column is missing is not an empty sweep: it is a broken input,
// and reporting it as "nothing refused" would be the worst possible answer.
func TestSweepRefusesADumpWithoutTheColumn(t *testing.T) {
	def := register.Definition{ID: "gb", Kind: "k", Country: "GB", Column: "CompanyNumber"}
	var err error
	for _, e := range register.Sweep(strings.NewReader("Name,Status\nA,Active\n"), def) {
		err = e
		break
	}
	if err == nil {
		t.Fatal("a dump without the declared column must fail the sweep")
	}
	if !strings.Contains(err.Error(), "CompanyNumber") {
		t.Fatalf("the error must name the missing column, got %v", err)
	}
}

// Blank cells exist in real dumps and are not identifiers.
func TestSweepSkipsEmptyCells(t *testing.T) {
	def := register.Definition{ID: "gb", Kind: "k", Country: "GB", Column: "N"}
	n := 0
	for c, err := range register.Sweep(strings.NewReader("N\n\n 08209948 \n\n"), def) {
		if err != nil {
			t.Fatal(err)
		}
		if c.GetInput() != "08209948" {
			t.Fatalf("the cell must be trimmed, got %q", c.GetInput())
		}
		n++
	}
	if n != 1 {
		t.Fatalf("got %d cases, want 1", n)
	}
}

// Bulk exports contain short rows. Skipping them is right; crashing on them, or
// reading a neighbouring column as if it were the identifier, is not.
func TestSweepSkipsRowsShorterThanTheColumn(t *testing.T) {
	def := register.Definition{ID: "gb", Kind: "k", Country: "GB", Column: "B"}
	var got []string
	for c, err := range register.Sweep(strings.NewReader("A,B\nonly-one-cell\nx,08209948\n"), def) {
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, c.GetInput())
	}
	if len(got) != 1 || got[0] != "08209948" {
		t.Fatalf("got %v", got)
	}
}

// A caller that stops early - a runner whose testee died - must not leave the
// sweep pushing rows into a closed pipe.
func TestSweepStopsWhenTheCallerStops(t *testing.T) {
	def := register.Definition{ID: "gb", Kind: "k", Country: "GB", Column: "N"}
	n := 0
	for range register.Sweep(strings.NewReader("N\n1\n2\n3\n4\n"), def) {
		n++
		break
	}
	if n != 1 {
		t.Fatalf("the sweep produced %d cases after the caller stopped", n)
	}
}

// An empty input has no header at all, which is a broken dump rather than a
// register that issued nothing.
func TestSweepRefusesAnInputWithoutAHeader(t *testing.T) {
	def := register.Definition{ID: "gb", Kind: "k", Country: "GB", Column: "N"}
	var err error
	for _, e := range register.Sweep(strings.NewReader(""), def) {
		err = e
	}
	if err == nil || !strings.Contains(err.Error(), "header") {
		t.Fatalf("got %v", err)
	}
}
