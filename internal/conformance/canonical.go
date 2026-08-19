package conformance

import (
	"bytes"
	"encoding/json"
	"sort"
)

// canonicalCase mirrors the JSONL schema. `encoding/json` emits struct fields
// in declaration order, so the declaration order below *is* the canonical key
// order of the published corpus.
type canonicalCase struct {
	ID                  string             `json:"id"`
	Description         string             `json:"description,omitempty"`
	Kind                *string            `json:"kind,omitempty"`
	CountryCode         *string            `json:"countryCode,omitempty"`
	Input               *string            `json:"input,omitempty"`
	Profile             *string            `json:"profile,omitempty"`
	Operation           string             `json:"operation"`
	Expected            *canonicalExpected `json:"expected,omitempty"`
	Tags                []string           `json:"tags"`
	SourceIDs           []string           `json:"sourceIds,omitempty"`
	Fixture             *string            `json:"fixture,omitempty"`
	ExpectedEngineError *string            `json:"expectedEngineError,omitempty"`
	Generated           bool               `json:"generated,omitempty"`
	DataClassification  string             `json:"dataClassification"`
	RedistributionBasis string             `json:"redistributionBasis"`
}

type canonicalExpected struct {
	Kind           *string        `json:"kind,omitempty"`
	CanonicalValue *string        `json:"canonicalValue,omitempty"`
	CountryCode    *string        `json:"countryCode,omitempty"`
	Status         *string        `json:"status,omitempty"`
	ReasonCode     *string        `json:"reasonCode,omitempty"`
	MessageKey     *string        `json:"messageKey,omitempty"`
	Format         *canonicalStep `json:"format,omitempty"`
	Checksum       *canonicalStep `json:"checksum,omitempty"`
}

type canonicalStep struct {
	Status     string  `json:"status"`
	ReasonCode string  `json:"reasonCode"`
	MessageKey *string `json:"messageKey,omitempty"`
}

// WriteCanonicalJSONL renders the reviewed corpus in its canonical release
// form: cases sorted by id, JSON keys in schema order, UTF-8, one LF terminated
// line per case and no insignificant whitespace.
func WriteCanonicalJSONL(cases []*Case) ([]byte, error) {
	ordered := append([]*Case(nil), cases...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	var buf bytes.Buffer
	for _, c := range ordered {
		line, err := CanonicalLine(c)
		if err != nil {
			return nil, err
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

// CanonicalLine renders one case in the canonical form, without any trailing
// newline. HTML escaping is disabled so that the bytes only depend on the
// reviewed content, never on the defaults of a JSON library.
func CanonicalLine(c *Case) ([]byte, error) {
	out := canonicalCase{
		ID:                  c.ID,
		Description:         c.Description,
		Kind:                c.Kind,
		CountryCode:         c.CountryCode,
		Input:               c.Input,
		Profile:             c.Profile,
		Operation:           c.Operation,
		Tags:                c.Tags,
		SourceIDs:           c.SourceIDs,
		Fixture:             c.Fixture,
		ExpectedEngineError: c.ExpectedEngineError,
		Generated:           c.Generated,
		DataClassification:  c.DataClassification,
		RedistributionBasis: c.RedistributionBasis,
	}
	if c.Expected != nil {
		out.Expected = &canonicalExpected{
			Kind:           c.Expected.Kind,
			CanonicalValue: c.Expected.CanonicalValue,
			CountryCode:    c.Expected.CountryCode,
			Status:         c.Expected.Status,
			ReasonCode:     c.Expected.ReasonCode,
			MessageKey:     c.Expected.MessageKey,
			Format:         toCanonicalStep(c.Expected.Format),
			Checksum:       toCanonicalStep(c.Expected.Checksum),
		}
	}
	if out.Tags == nil {
		out.Tags = []string{}
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(out); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

func toCanonicalStep(s *Step) *canonicalStep {
	if s == nil {
		return nil
	}
	return &canonicalStep{Status: s.Status, ReasonCode: s.ReasonCode, MessageKey: s.MessageKey}
}
