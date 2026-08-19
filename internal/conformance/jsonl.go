// Package conformance reads the reviewed JSONL corpus, validates it strictly,
// compiles it into the shared Protobuf suite and runs it against the internal
// reference interpreter.
//
// The compiler never derives an expectation from the rule under test: every
// status, reason code, canonical value and message key is written by a human
// reviewer from an independent source.
package conformance

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/libbusinessid/spec/internal/diagnostics"
	"github.com/libbusinessid/spec/internal/limits"
)

// Diagnostic codes emitted by the conformance reader.
const (
	CodeJSON           = "CONF001"
	CodeMissingField   = "CONF002"
	CodeForbiddenField = "CONF003"
	CodeBadValue       = "CONF004"
	CodeDuplicateID    = "CONF005"
	CodeDataPolicy     = "CONF006"
	CodeFixture        = "CONF007"
	CodeLimit          = "CONF008"
)

// Operation names accepted in the JSONL corpus.
const (
	OpCanonicalize     = "canonicalize"
	OpValidateFormat   = "validate_format"
	OpValidateChecksum = "validate_checksum"
	OpValidate         = "validate"
	OpLoadRuleset      = "load_ruleset"
)

// Case is one line of the JSONL corpus.
type Case struct {
	ID                  string    `json:"id"`
	Description         string    `json:"description,omitempty"`
	Kind                *string   `json:"kind,omitempty"`
	CountryCode         *string   `json:"countryCode,omitempty"`
	Input               *string   `json:"input,omitempty"`
	Profile             *string   `json:"profile,omitempty"`
	Operation           string    `json:"operation"`
	Expected            *Expected `json:"expected,omitempty"`
	Tags                []string  `json:"tags"`
	SourceIDs           []string  `json:"sourceIds,omitempty"`
	Fixture             *string   `json:"fixture,omitempty"`
	ExpectedEngineError *string   `json:"expectedEngineError,omitempty"`
	Generated           bool      `json:"generated,omitempty"`
	DataClassification  string    `json:"dataClassification"`
	RedistributionBasis string    `json:"redistributionBasis"`

	// Line is the one based position of the case in its file.
	Line int `json:"-"`
	// File is the POSIX relative path of the source file.
	File string `json:"-"`
}

// Expected is the reviewed expectation of a case.
type Expected struct {
	Kind           *string `json:"kind,omitempty"`
	CanonicalValue *string `json:"canonicalValue,omitempty"`
	CountryCode    *string `json:"countryCode,omitempty"`
	Status         *string `json:"status,omitempty"`
	ReasonCode     *string `json:"reasonCode,omitempty"`
	MessageKey     *string `json:"messageKey,omitempty"`
	Format         *Step   `json:"format,omitempty"`
	Checksum       *Step   `json:"checksum,omitempty"`
}

// Step is the reviewed expectation of one validation step.
type Step struct {
	Status     string  `json:"status"`
	ReasonCode string  `json:"reasonCode"`
	MessageKey *string `json:"messageKey,omitempty"`
}

// Read parses one JSONL file and returns its cases.
func Read(path string, src []byte) ([]*Case, *diagnostics.Bag) {
	bag := diagnostics.New()
	if !utf8.Valid(src) {
		bag.Errorf(diagnostics.Position{File: path}, CodeJSON, "the corpus is not valid UTF-8")
		return nil, bag
	}
	var out []*Case
	scanner := bufio.NewScanner(bytes.NewReader(src))
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		raw := scanner.Bytes()
		if len(bytes.TrimSpace(raw)) == 0 {
			continue
		}
		if bytes.HasPrefix(bytes.TrimSpace(raw), []byte("//")) ||
			bytes.HasPrefix(bytes.TrimSpace(raw), []byte("#")) {
			bag.Errorf(diagnostics.Position{File: path, Line: line}, CodeJSON,
				"the corpus holds one self contained JSON object per line and no comment")
			continue
		}
		c := &Case{File: path, Line: line}
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(c); err != nil {
			bag.Errorf(diagnostics.Position{File: path, Line: line}, CodeJSON, "invalid case: %v", err)
			continue
		}
		if err := dec.Decode(new(json.RawMessage)); err != io.EOF {
			bag.Errorf(diagnostics.Position{File: path, Line: line}, CodeJSON,
				"a line must hold exactly one JSON object")
			continue
		}
		out = append(out, c)
	}
	if err := scanner.Err(); err != nil {
		bag.Errorf(diagnostics.Position{File: path}, CodeJSON, "cannot read the corpus: %v", err)
	}
	return out, bag
}

// forbiddenPhrases are rejected everywhere in the corpus: no production data
// can ever enter it.
var forbiddenPhrases = []string{"production case", "cas de production", "production data"}

// Validate applies every structural and data policy rule to a set of cases.
func Validate(cases []*Case) *diagnostics.Bag {
	bag := diagnostics.New()
	if len(cases) > limits.MaxConformanceCases {
		bag.Errorf(diagnostics.Position{}, CodeLimit,
			"the corpus holds %d cases, the limit is %d", len(cases), limits.MaxConformanceCases)
	}
	seen := map[string]*Case{}
	for _, c := range cases {
		pos := diagnostics.Position{File: c.File, Line: c.Line}
		if c.ID == "" {
			bag.Errorf(pos, CodeMissingField, "a case must declare a non empty id")
			continue
		}
		if prev, dup := seen[c.ID]; dup {
			bag.Errorf(pos, CodeDuplicateID, "duplicate case id %q, already declared at %s",
				c.ID, diagnostics.Position{File: prev.File, Line: prev.Line}.String())
			continue
		}
		seen[c.ID] = c
		if len(c.Description) > limits.MaxDescriptionBytes {
			bag.Errorf(pos, CodeLimit, "the description exceeds %d bytes", limits.MaxDescriptionBytes)
		}
		validateTags(bag, pos, c)
		validateDataPolicy(bag, pos, c)
		switch c.Operation {
		case OpCanonicalize, OpValidateFormat, OpValidateChecksum, OpValidate:
			validateBusinessCase(bag, pos, c)
		case OpLoadRuleset:
			validateLoadCase(bag, pos, c)
		case "":
			bag.Errorf(pos, CodeMissingField, "a case must declare an operation")
		default:
			bag.Suggestf(pos, CodeBadValue,
				"accepted operations are canonicalize, validate_format, validate_checksum, validate and load_ruleset",
				"unknown operation %q", c.Operation)
		}
	}
	return bag
}

func validateTags(bag *diagnostics.Bag, pos diagnostics.Position, c *Case) {
	if len(c.Tags) == 0 {
		bag.Errorf(pos, CodeMissingField, "a case must declare at least one tag")
		return
	}
	for i, tag := range c.Tags {
		if tag == "" {
			bag.Errorf(pos, CodeBadValue, "a tag must not be empty")
			return
		}
		if i > 0 && c.Tags[i-1] >= tag {
			bag.Suggestf(pos, CodeBadValue, "sort the tags and remove the duplicates",
				"tags must be sorted and unique, %q follows %q", tag, c.Tags[i-1])
			return
		}
	}
}

func validateDataPolicy(bag *diagnostics.Bag, pos diagnostics.Position, c *Case) {
	switch c.DataClassification {
	case "official_public_example", "synthetic", "public_business_identifier":
	case "":
		bag.Errorf(pos, CodeMissingField, "a case must declare dataClassification")
	default:
		bag.Suggestf(pos, CodeDataPolicy,
			"accepted classifications are official_public_example, synthetic and public_business_identifier",
			"unknown dataClassification %q", c.DataClassification)
	}
	if strings.TrimSpace(c.RedistributionBasis) == "" {
		bag.Errorf(pos, CodeMissingField, "a case must declare a non empty redistributionBasis")
	}
	haystack := strings.ToLower(c.ID + " " + c.Description + " " + c.RedistributionBasis + " " +
		strings.Join(c.Tags, " "))
	for _, phrase := range forbiddenPhrases {
		if strings.Contains(haystack, phrase) {
			bag.Errorf(pos, CodeDataPolicy,
				"the corpus must never mention %q: no production data can enter it", phrase)
		}
	}
	if (c.DataClassification == "official_public_example" ||
		c.DataClassification == "public_business_identifier") && len(c.SourceIDs) == 0 {
		bag.Errorf(pos, CodeDataPolicy,
			"an official example must reference at least one source id")
	}
	for i, id := range c.SourceIDs {
		if id == "" {
			bag.Errorf(pos, CodeBadValue, "a source id must not be empty")
		}
		if i > 0 && c.SourceIDs[i-1] >= id {
			bag.Errorf(pos, CodeBadValue, "source ids must be sorted and unique")
			return
		}
	}
}

//nolint:gocyclo // one exhaustive validation of the business case shape.
func validateBusinessCase(bag *diagnostics.Bag, pos diagnostics.Position, c *Case) {
	if c.Fixture != nil || c.ExpectedEngineError != nil {
		bag.Errorf(pos, CodeForbiddenField,
			"fixture and expectedEngineError belong to load_ruleset cases only")
	}
	if c.Kind == nil {
		bag.Errorf(pos, CodeMissingField, "operation %s requires kind", c.Operation)
	}
	if c.Input == nil {
		bag.Errorf(pos, CodeMissingField, "operation %s requires input", c.Operation)
	}
	if c.Profile == nil {
		bag.Errorf(pos, CodeMissingField, "operation %s requires profile", c.Operation)
	} else if *c.Profile != "compatible" && *c.Profile != "strict_current" {
		bag.Errorf(pos, CodeBadValue, "unknown profile %q", *c.Profile)
	}
	if c.Expected == nil {
		bag.Errorf(pos, CodeMissingField, "operation %s requires expected", c.Operation)
		return
	}
	e := c.Expected
	if e.CanonicalValue == nil {
		bag.Errorf(pos, CodeMissingField, "expected.canonicalValue is always written by the author")
	}
	if c.Operation == OpCanonicalize {
		if e.Format != nil || e.Checksum != nil {
			bag.Errorf(pos, CodeForbiddenField,
				"a canonicalize case expects a status and a reason code, not format or checksum steps")
		}
		if e.Status == nil || e.ReasonCode == nil {
			bag.Errorf(pos, CodeMissingField, "a canonicalize case requires expected.status and expected.reasonCode")
			return
		}
		validateStatusReason(bag, pos, *e.Status, *e.ReasonCode, e.MessageKey)
		return
	}
	if e.Status != nil || e.ReasonCode != nil || e.MessageKey != nil {
		bag.Errorf(pos, CodeForbiddenField,
			"a validation case expects format and checksum steps, not a top level status")
	}
	if e.Format == nil || e.Checksum == nil {
		bag.Errorf(pos, CodeMissingField, "a validation case requires expected.format and expected.checksum")
		return
	}
	validateStatusReason(bag, pos, e.Format.Status, e.Format.ReasonCode, e.Format.MessageKey)
	validateStatusReason(bag, pos, e.Checksum.Status, e.Checksum.ReasonCode, e.Checksum.MessageKey)
	if c.Operation == OpValidateFormat && e.Format.Status == "valid" &&
		(e.Checksum.Status != "not_run" || e.Checksum.ReasonCode != "not_requested") {
		bag.Errorf(pos, CodeBadValue,
			"after a valid format, validate_format requires checksum not_run/not_requested")
	}
}

func validateLoadCase(bag *diagnostics.Bag, pos diagnostics.Position, c *Case) {
	if c.Kind != nil || c.Input != nil || c.Profile != nil || c.Expected != nil {
		bag.Errorf(pos, CodeForbiddenField,
			"a load_ruleset case forbids kind, input, profile and expected")
	}
	if c.CountryCode != nil {
		bag.Errorf(pos, CodeForbiddenField, "a load_ruleset case forbids countryCode")
	}
	if c.Fixture == nil || *c.Fixture == "" {
		bag.Errorf(pos, CodeMissingField, "a load_ruleset case requires a fixture path")
	}
	if c.ExpectedEngineError == nil || *c.ExpectedEngineError == "" {
		bag.Errorf(pos, CodeMissingField, "a load_ruleset case requires expectedEngineError")
		return
	}
	switch *c.ExpectedEngineError {
	case "invalid_ruleset", "incompatible_ruleset":
	default:
		bag.Suggestf(pos, CodeBadValue,
			"accepted engine errors are invalid_ruleset and incompatible_ruleset",
			"unknown expectedEngineError %q", *c.ExpectedEngineError)
	}
}

// statusReasons is the normative pairing of statuses and reason codes.
var statusReasons = map[string]map[string]bool{
	"valid": {"ok": true},
	"not_run": {
		"not_requested": true, "not_run_format_invalid": true, "not_run_format_unsupported": true,
	},
	"invalid": {
		"empty": true, "invalid_length": true, "invalid_characters": true,
		"invalid_format": true, "invalid_checksum": true, "country_mismatch": true,
	},
	"unsupported": {
		"unsupported_kind": true, "unsupported_country": true, "unsupported_format": true,
		"unsupported_checksum": true, "checksum_not_published": true,
		"missing_country_code": true, "registry_not_configured": true,
		"incompatible_ruleset": true, "invalid_ruleset": true, "input_too_long": true,
	},
}

func validateStatusReason(bag *diagnostics.Bag, pos diagnostics.Position, status, reason string, key *string) {
	allowed, ok := statusReasons[status]
	if !ok {
		bag.Suggestf(pos, CodeBadValue, "accepted statuses are valid, invalid, unsupported and not_run",
			"unknown status %q", status)
		return
	}
	if !allowed[reason] {
		bag.Suggestf(pos, CodeBadValue, "see the reason code registry of docs/ir.md",
			"the status %q cannot carry the reason code %q", status, reason)
	}
	if key != nil && *key == "" {
		bag.Errorf(pos, CodeBadValue, "an explicit message key must not be empty")
	}
}

// SortCases orders the corpus by case id.
func SortCases(cases []*Case) {
	sort.SliceStable(cases, func(i, j int) bool { return cases[i].ID < cases[j].ID })
}
