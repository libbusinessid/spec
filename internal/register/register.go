// Package register turns an issuer's own bulk register into conformance cases.
//
// The corpus under conformance/ is chosen by hand: one case per prefix, one per
// shape, one per boundary. It proves that a rule handles the forms someone
// thought of. A register sweep proves something the corpus cannot, which is
// that the rule refuses nothing the issuer has actually handed out - all
// 5695465 companies of the Companies House register rather than the sixty a
// person selected from it.
//
// The expectation is not a judgement. An identifier that appears in its own
// issuer's register is valid by definition, so every case a sweep produces
// expects valid. That is also why a sweep is never authored from the reference
// interpreter: the authority is the register, and the interpreter is what is
// being questioned.
//
// Dumps are large - half a gigabyte compressed for one country - renewed
// monthly, and published under the issuer's own terms. None of that belongs in
// version control, so a sweep reads a file the caller supplies and this package
// holds only the description of where that file comes from.
package register

import (
	"encoding/csv"
	"fmt"
	"io"
	"iter"
	"strconv"
	"strings"

	conformancev1 "github.com/entid-org/spec/gen/go/entid/conformance/v1"
	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
)

// Definition describes one issuer's bulk register: what to download, how to
// read it, and which rule it exercises.
type Definition struct {
	// ID names the register in the manifest and in every case it produces.
	ID string `json:"id"`
	// Kind and Country route the identifiers to a rule.
	Kind    string `json:"kind"`
	Country string `json:"country"`
	// URL is where the issuer publishes the dump, and Archive names the entry
	// to read when the download is a zip.
	URL     string `json:"url"`
	Archive string `json:"archive,omitempty"`
	// Column is the CSV header holding the identifier.
	Column string `json:"column"`
	// Authority, Terms and Notes carry provenance, on the same footing as the
	// source block of a rule.
	Authority string `json:"authority"`
	Terms     string `json:"terms"`
	Notes     string `json:"notes,omitempty"`
}

// Sweep reads a CSV dump and yields one conformance case per identifier.
//
// It streams: a caller consuming the sequence never holds more than one case,
// which is what makes a five million row register affordable. An error ends the
// sequence, because a dump that cannot be read must not be reported as a sweep
// that refused nothing.
func Sweep(r io.Reader, def Definition) iter.Seq2[*conformancev1.ConformanceCase, error] {
	return func(yield func(*conformancev1.ConformanceCase, error) bool) {
		cr := csv.NewReader(r)
		cr.FieldsPerRecord = -1
		cr.ReuseRecord = true
		// Real dumps carry quoting a strict reader rejects; refusing the file
		// over one malformed quote would lose the other five million rows.
		cr.LazyQuotes = true

		header, err := cr.Read()
		if err != nil {
			yield(nil, fmt.Errorf("register %s: cannot read the header: %w", def.ID, err))
			return
		}
		column := -1
		for i, h := range header {
			if normalizeHeader(h) == def.Column {
				column = i
				break
			}
		}
		if column < 0 {
			yield(nil, fmt.Errorf("register %s: the dump carries no column %q, only %s",
				def.ID, def.Column, strings.Join(header, ", ")))
			return
		}

		for row := 1; ; row++ {
			record, err := cr.Read()
			if err == io.EOF {
				return
			}
			if err != nil {
				yield(nil, fmt.Errorf("register %s: row %d: %w", def.ID, row, err))
				return
			}
			if column >= len(record) {
				continue
			}
			value := strings.TrimSpace(record[column])
			if value == "" {
				continue
			}
			if !yield(caseFor(def, value, row), nil) {
				return
			}
		}
	}
}

// caseFor builds the case for one register entry.
func caseFor(def Definition, value string, row int) *conformancev1.ConformanceCase {
	country := def.Country
	return &conformancev1.ConformanceCase{
		// The identifier carries the value as well as the row, because a sweep
		// reports a refusal by identifier and nothing else. "row 4213877 was
		// refused" sends the reader back into a five million line file; the
		// value is the answer they were going to look up.
		Id:          def.ID + "-" + strconv.Itoa(row) + "-" + value,
		Kind:        def.Kind,
		CountryCode: &country,
		Input:       value,
		Profile:     "compatible",
		// The whole validation, not just the format: a rule refuses just as
		// firmly through the checksum, and asking only for the format would
		// report a clean sweep over a checksum it never ran.
		Operation: conformancev1.Operation_OPERATION_VALIDATE,
		Expected: &conformancev1.ExpectedOutcome{
			Value: &conformancev1.ExpectedOutcome_ValidationReport{
				ValidationReport: &conformancev1.ExpectedValidationReport{
					Kind:        def.Kind,
					InputValue:  value,
					CountryCode: &country,
					Profile:     "compatible",
					Format: &conformancev1.ExpectedStep{
						Status:     conformancev1.StepStatus_STEP_STATUS_VALID,
						ReasonCode: irv1.ReasonCode_REASON_CODE_OK,
					},
				},
			},
		},
	}
}

// bom is the byte order mark bulk exports leave on the first header name.
const bom = "\ufeff"

// normalizeHeader strips the byte order mark and the padding that bulk exports
// put around header names.
func normalizeHeader(h string) string {
	return strings.TrimSpace(strings.TrimPrefix(h, bom))
}
