package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/artifact"
	"github.com/entid-org/spec/internal/features"
)

// inspection is the machine readable description of a bundle.
type inspection struct {
	FormatVersion    uint32                `json:"formatVersion"`
	RulesVersion     string                `json:"rulesVersion"`
	Sha256           string                `json:"sha256"`
	SourceDigest     string                `json:"sourceDigest"`
	RequiredFeatures []inspectedFeature    `json:"requiredFeatures"`
	Identifiers      []inspectedIdentifier `json:"identifiers"`
	Dispatchers      []inspectedDispatcher `json:"dispatchers"`
	ProgramCount     int                   `json:"programCount"`
	NodeCount        int                   `json:"nodeCount"`
	Operations       map[string]int        `json:"operations"`
}

type inspectedFeature struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}

type inspectedIdentifier struct {
	ID          uint32   `json:"id"`
	Kind        string   `json:"kind"`
	CountryCode string   `json:"countryCode"`
	Profile     string   `json:"defaultProfile"`
	HasChecksum bool     `json:"hasChecksum"`
	Sources     []string `json:"sources"`
}

type inspectedDispatcher struct {
	Kind           string   `json:"kind"`
	KindAliases    []string `json:"kindAliases,omitempty"`
	CountryAliases []string `json:"countryAliases,omitempty"`
	Targets        []string `json:"targets"`
}

func inspectBundle(raw []byte) (*inspection, error) {
	rules, err := artifact.LoadRuleset(raw)
	if err != nil {
		return nil, err
	}
	bundle := rules.Bundle
	out := &inspection{
		FormatVersion: bundle.GetFormatVersion(),
		RulesVersion:  bundle.GetRulesVersion(),
		Sha256:        artifact.SHA256Hex(raw),
		SourceDigest:  fmt.Sprintf("%x", bundle.GetSourceDigest()),
		ProgramCount:  len(bundle.GetPrograms()),
		Operations:    map[string]int{},
	}
	for _, id := range bundle.GetRequiredFeatureIds() {
		c, ok := features.Lookup(id)
		if !ok {
			continue
		}
		out.RequiredFeatures = append(out.RequiredFeatures, inspectedFeature{ID: c.ID, Name: c.Name})
	}
	for _, d := range bundle.GetIdentifiers() {
		country := globalCountry
		if d.CountryCode != nil {
			country = d.GetCountryCode()
		}
		var sources []string
		for _, s := range d.GetSources() {
			sources = append(sources, s.GetId())
		}
		out.Identifiers = append(out.Identifiers, inspectedIdentifier{
			ID: d.GetId(), Kind: d.GetKind(), CountryCode: country,
			Profile: d.GetDefaultProfile(), HasChecksum: d.ChecksumProgram != nil, Sources: sources,
		})
	}
	for _, d := range bundle.GetDispatchers() {
		entry := inspectedDispatcher{Kind: d.GetKind(), KindAliases: d.GetKindAliases()}
		for _, ca := range d.GetCountryAliases() {
			entry.CountryAliases = append(entry.CountryAliases, ca.GetAlias()+"="+ca.GetCountryCode())
		}
		for _, t := range d.GetTargets() {
			country := globalCountry
			if t.CountryCode != nil {
				country = t.GetCountryCode()
			}
			entry.Targets = append(entry.Targets, fmt.Sprintf("%s->%d", country, t.GetIdentifierDefinitionId()))
		}
		out.Dispatchers = append(out.Dispatchers, entry)
	}
	for _, p := range bundle.GetPrograms() {
		out.NodeCount += len(p.GetNodes())
		for _, n := range p.GetNodes() {
			out.Operations[operationName(n)]++
		}
	}
	return out, nil
}

func operationName(n *irv1.Node) string {
	switch op := n.GetOperation().(type) {
	case *irv1.Node_StringOperation:
		return op.StringOperation.GetKind().String()
	case *irv1.Node_IntegerOperation:
		return op.IntegerOperation.GetKind().String()
	case *irv1.Node_PredicateOperation:
		return op.PredicateOperation.GetKind().String()
	case *irv1.Node_CanonicalizationOperation:
		return op.CanonicalizationOperation.GetKind().String()
	case *irv1.Node_AssertionOperation:
		return op.AssertionOperation.GetKind().String()
	case *irv1.Node_ChecksumOperation:
		return op.ChecksumOperation.GetKind().String()
	case *irv1.Node_CallOperation:
		return op.CallOperation.GetKind().String()
	default:
		return "UNKNOWN"
	}
}

func runInspect(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("inspect", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "render the inspection as JSON")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "usage: entidc inspect [--json] <bundle.binpb>")
		return exitUsage
	}
	raw, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return fail(stderr, "cannot read the bundle: %v", err)
	}
	out, err := inspectBundle(raw)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "the bundle is refused: %v\n", err)
		return exitRejected
	}
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			return fail(stderr, "cannot render the inspection: %v", err)
		}
		return exitOK
	}
	_, _ = fmt.Fprintf(stdout, "rules version   %s\n", out.RulesVersion)
	_, _ = fmt.Fprintf(stdout, "format version  %d\n", out.FormatVersion)
	_, _ = fmt.Fprintf(stdout, "sha256          %s\n", out.Sha256)
	_, _ = fmt.Fprintf(stdout, "source digest   %s\n", out.SourceDigest)
	_, _ = fmt.Fprintf(stdout, "programs        %d (%d nodes)\n", out.ProgramCount, out.NodeCount)
	_, _ = fmt.Fprintf(stdout, "identifiers     %d\n", len(out.Identifiers))
	_, _ = fmt.Fprintln(stdout, "capabilities")
	for _, f := range out.RequiredFeatures {
		_, _ = fmt.Fprintf(stdout, "  %3d %s\n", f.ID, f.Name)
	}
	_, _ = fmt.Fprintln(stdout, "definitions")
	for _, d := range out.Identifiers {
		checksum := "no checksum"
		if d.HasChecksum {
			checksum = "checksum"
		}
		_, _ = fmt.Fprintf(stdout, "  %-10s %-8s %-14s %s (sources: %s)\n",
			d.Kind, d.CountryCode, d.Profile, checksum, strings.Join(d.Sources, ", "))
	}
	_, _ = fmt.Fprintln(stdout, "dispatchers")
	for _, d := range out.Dispatchers {
		_, _ = fmt.Fprintf(stdout, "  %-10s aliases=[%s] countries=[%s] targets=[%s]\n",
			d.Kind, strings.Join(d.KindAliases, ","), strings.Join(d.CountryAliases, ","),
			strings.Join(d.Targets, ","))
	}
	_, _ = fmt.Fprintln(stdout, "operations")
	names := make([]string, 0, len(out.Operations))
	for name := range out.Operations {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		_, _ = fmt.Fprintf(stdout, "  %-52s %d\n", name, out.Operations[name])
	}
	return exitOK
}
