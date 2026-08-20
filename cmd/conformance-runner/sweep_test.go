package main

import (
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testManifest = `{"registers":[
  {"id":"gb-companies-house","kind":"company_number","country":"GB",
   "column":"CompanyNumber","authority":"Companies House","terms":"OGL v3.0"}
]}`

func writeManifest(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "registers.json")
	if err := os.WriteFile(p, []byte(testManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestParseSweepPairsTheFileWithItsRegister(t *testing.T) {
	m := writeManifest(t)
	def, path, err := parseSweep("gb-companies-house=/tmp/dump.csv", m)
	if err != nil {
		t.Fatal(err)
	}
	if def.Kind != "company_number" || def.Country != "GB" || def.Column != "CompanyNumber" {
		t.Fatalf("wrong definition: %+v", def)
	}
	if path != "/tmp/dump.csv" {
		t.Fatalf("got path %q", path)
	}
}

// An unknown register must say what it does know, because the mistake is
// almost always a typo rather than a missing entry.
func TestParseSweepNamesTheRegistersItKnows(t *testing.T) {
	_, _, err := parseSweep("fr-inpi=/tmp/dump.csv", writeManifest(t))
	if err == nil {
		t.Fatal("an unknown register must be refused")
	}
	if !strings.Contains(err.Error(), "gb-companies-house") {
		t.Fatalf("the error must list the known registers, got %v", err)
	}
}

func TestParseSweepRefusesAMalformedArgument(t *testing.T) {
	m := writeManifest(t)
	for _, arg := range []string{"gb-companies-house", "=file.csv", "gb=", ""} {
		if _, _, err := parseSweep(arg, m); err == nil {
			t.Fatalf("%q must be refused", arg)
		}
	}
}

func TestParseSweepReportsAMissingManifest(t *testing.T) {
	_, _, err := parseSweep("gb=x.csv", filepath.Join(t.TempDir(), "absent.json"))
	if err == nil || !strings.Contains(err.Error(), "absent.json") {
		t.Fatalf("the error must name the file it could not read, got %v", err)
	}
}

// Issuers publish plain and gzipped dumps alike. Guessing wrong would sweep an
// empty file, and a sweep that read nothing looks exactly like a sweep that
// refused nothing.
func TestOpenDumpReadsPlainAndGzip(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "d.csv")
	if err := os.WriteFile(plain, []byte("CompanyNumber\n08209948\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	zipped := filepath.Join(dir, "d.csv.gz")
	f, err := os.Create(zipped)
	if err != nil {
		t.Fatal(err)
	}
	zw := gzip.NewWriter(f)
	if _, err := zw.Write([]byte("CompanyNumber\n08209948\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{plain, zipped} {
		r, err := openDump(p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		buf := make([]byte, 13)
		if _, err := r.Read(buf); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		if string(buf) != "CompanyNumber" {
			t.Fatalf("%s: read %q", p, buf)
		}
		if err := r.Close(); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
}

func TestOpenDumpRefusesAFileThatIsNotGzip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lying.csv.gz")
	if err := os.WriteFile(p, []byte("not gzip at all"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := openDump(p); err == nil {
		t.Fatal("a file that only claims to be gzip must be refused")
	}
}

// A sweep asks one question and is not a conformance verdict, so combining it
// with the diagnosis flag would produce a result that is neither.
func TestSweepAndOperationCannotBeCombined(t *testing.T) {
	_, err := run("", testeeCommand(t), 0, "validate", "gb-companies-house=x.csv")
	if err == nil || !strings.Contains(err.Error(), "cannot be combined") {
		t.Fatalf("got %v", err)
	}
}

const sirenManifest = `{"registers":[
  {"id":"fr-test","kind":"siren","country":"FR","column":"siren",
   "authority":"Test","terms":"n/a"}
]}`

func sweepFixture(t *testing.T, rows string) (string, string) {
	t.Helper()
	dir := t.TempDir()
	m := filepath.Join(dir, "registers.json")
	if err := os.WriteFile(m, []byte(sirenManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	d := filepath.Join(dir, "dump.csv")
	if err := os.WriteFile(d, []byte(rows), 0o600); err != nil {
		t.Fatal(err)
	}
	return m, d
}

// A register holds only identifiers that exist, so a rule that refuses none of
// them is the whole claim a sweep makes.
func TestRunSweepAcceptsEveryIdentifierOfTheRegister(t *testing.T) {
	m, d := sweepFixture(t, "siren\n012345674\n123456782\n000000000\n")
	ok, err := runSweep("fr-test="+d, testeeCommand(t), 0, m)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("no identifier of the register should have been refused")
	}
}

// The failure that matters: an identifier the issuer handed out that the rule
// turns away. Section 1.2 calls it the worst defect this project can commit.
func TestRunSweepReportsARefusedIdentifier(t *testing.T) {
	m, d := sweepFixture(t, "siren\n012345674\n000000000000000\n")
	ok, err := runSweep("fr-test="+d, testeeCommand(t), 0, m)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("a refused identifier must fail the sweep")
	}
}

// An empty dump refuses nothing, which reads exactly like a clean sweep. That
// confusion is the one worth ruling out.
func TestRunSweepRefusesAnEmptyDump(t *testing.T) {
	m, d := sweepFixture(t, "siren\n")
	_, err := runSweep("fr-test="+d, testeeCommand(t), 0, m)
	if err == nil || !strings.Contains(err.Error(), "no identifier") {
		t.Fatalf("an empty dump must void the sweep, got %v", err)
	}
}

func TestRunSweepReportsAnUnreadableDump(t *testing.T) {
	m, _ := sweepFixture(t, "siren\n")
	_, err := runSweep("fr-test="+filepath.Join(t.TempDir(), "absent.csv"), testeeCommand(t), 0, m)
	if err == nil || !strings.Contains(err.Error(), "cannot open the dump") {
		t.Fatalf("got %v", err)
	}
}

// A dump missing its declared column has to void the run rather than end it
// quietly: a truncated sweep that refused nothing looks like a complete one.
func TestRunSweepVoidsOnAMalformedDump(t *testing.T) {
	m, d := sweepFixture(t, "something_else\n012345674\n")
	_, err := runSweep("fr-test="+d, testeeCommand(t), 0, m)
	if err == nil || !strings.Contains(err.Error(), "siren") {
		t.Fatalf("the error must name the missing column, got %v", err)
	}
}

func TestParseSweepReportsAMalformedManifest(t *testing.T) {
	p := filepath.Join(t.TempDir(), "registers.json")
	if err := os.WriteFile(p, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, _, err := parseSweep("gb=x.csv", p)
	if err == nil || !strings.Contains(err.Error(), "valid JSON") {
		t.Fatalf("got %v", err)
	}
}

// A sweep over a real register can refuse thousands of values, and printing all
// of them buries the run. The listing stops at twenty and says how many are
// left.
func TestRunSweepTruncatesALongListOfRefusals(t *testing.T) {
	rows := "siren\n"
	for i := 0; i < 25; i++ {
		rows += "not-a-siren\n"
	}
	m, d := sweepFixture(t, rows)

	stdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	ok, runErr := runSweep("fr-test="+d, testeeCommand(t), 0, m)
	os.Stdout = stdout
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	printed, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if runErr != nil {
		t.Fatal(runErr)
	}
	if ok {
		t.Fatal("twenty five refusals must fail the sweep")
	}
	if n := strings.Count(string(printed), "REFUSED"); n != 20 {
		t.Fatalf("printed %d refusals, want 20", n)
	}
	if !strings.Contains(string(printed), "and 5 more") {
		t.Fatalf("the listing must say how many were left out:\n%s", printed)
	}
}
