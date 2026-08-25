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

// ChangeClass is the normative classification of a bundle change.
type ChangeClass string

// The change classes reported by `entidc diff`.
const (
	// ClassWidening accepts more inputs than before: it fixes false negatives.
	ClassWidening ChangeClass = "widening"
	// ClassRestriction accepts fewer inputs than before: high risk change.
	ClassRestriction ChangeClass = "restriction"
	// ClassMetadata changes provenance, versions or documentation only.
	ClassMetadata ChangeClass = "metadata"
	// ClassIRFeature changes the capabilities an engine must implement.
	ClassIRFeature ChangeClass = "ir_feature"
	// ClassPotentialIncompatibility may break an engine or a consumer.
	ClassPotentialIncompatibility ChangeClass = "potential_incompatibility"
)

// Change is one classified difference between two bundles.
type Change struct {
	Class   ChangeClass `json:"class"`
	Subject string      `json:"subject"`
	Detail  string      `json:"detail"`
}

// DiffBundles classifies every difference between two bundles.
func DiffBundles(oldBundle, newBundle *irv1.RuleBundle) []Change {
	var changes []Change
	add := func(class ChangeClass, subject, format string, args ...any) {
		changes = append(changes, Change{Class: class, Subject: subject, Detail: fmt.Sprintf(format, args...)})
	}
	if oldBundle.GetFormatVersion() != newBundle.GetFormatVersion() {
		add(ClassPotentialIncompatibility, "format_version", "%d becomes %d",
			oldBundle.GetFormatVersion(), newBundle.GetFormatVersion())
	}
	if oldBundle.GetRulesVersion() != newBundle.GetRulesVersion() {
		add(ClassMetadata, "rules_version", "%q becomes %q",
			oldBundle.GetRulesVersion(), newBundle.GetRulesVersion())
	}
	diffCapabilities(oldBundle, newBundle, add)
	diffDefinitions(oldBundle, newBundle, add)
	diffRouting(oldBundle, newBundle, add)
	sort.SliceStable(changes, func(i, j int) bool {
		if changes[i].Class != changes[j].Class {
			return changes[i].Class < changes[j].Class
		}
		if changes[i].Subject != changes[j].Subject {
			return changes[i].Subject < changes[j].Subject
		}
		return changes[i].Detail < changes[j].Detail
	})
	return changes
}

// addChange records one classified difference.
type addChange func(class ChangeClass, subject, format string, args ...any)

func diffCapabilities(oldBundle, newBundle *irv1.RuleBundle, add addChange) {
	oldFeatures := featureSet(oldBundle)
	newFeatures := featureSet(newBundle)
	for _, id := range sortedIDs(newFeatures) {
		if !oldFeatures[id] {
			add(ClassIRFeature, capabilityName(id), "capability %d is now required", id)
		}
	}
	for _, id := range sortedIDs(oldFeatures) {
		if !newFeatures[id] {
			add(ClassIRFeature, capabilityName(id), "capability %d is no longer required", id)
		}
	}
}

func diffDefinitions(oldBundle, newBundle *irv1.RuleBundle, add addChange) {
	oldDefs := definitionKeys(oldBundle)
	newDefs := definitionKeys(newBundle)
	for _, key := range sortedKeys(newDefs) {
		if _, ok := oldDefs[key]; !ok {
			add(ClassWidening, key, "the definition is new")
		}
	}
	for _, key := range sortedKeys(oldDefs) {
		if _, ok := newDefs[key]; !ok {
			add(ClassRestriction, key, "the definition disappears")
		}
	}
	for _, key := range sortedKeys(newDefs) {
		before, ok := oldDefs[key]
		if !ok {
			continue
		}
		after := newDefs[key]
		if before.GetDefaultProfile() != after.GetDefaultProfile() {
			add(ClassPotentialIncompatibility, key, "the default profile becomes %q", after.GetDefaultProfile())
		}
		switch {
		case before.ChecksumProgram == nil && after.ChecksumProgram != nil:
			add(ClassRestriction, key, "a checksum is now applied")
		case before.ChecksumProgram != nil && after.ChecksumProgram == nil:
			add(ClassWidening, key, "the checksum is no longer applied")
		}
		if sourceIDs(before) != sourceIDs(after) {
			add(ClassMetadata, key, "the sources become [%s]", sourceIDs(after))
		}
		if programFingerprint(oldBundle, before.GetFormatProgram()) !=
			programFingerprint(newBundle, after.GetFormatProgram()) {
			add(ClassPotentialIncompatibility, key, "the format program changed")
		}
		if programFingerprint(oldBundle, before.GetCanonicalizationProgram()) !=
			programFingerprint(newBundle, after.GetCanonicalizationProgram()) {
			add(ClassPotentialIncompatibility, key, "the canonicalization program changed")
		}
		if before.ChecksumProgram != nil && after.ChecksumProgram != nil &&
			programFingerprint(oldBundle, before.GetChecksumProgram()) !=
				programFingerprint(newBundle, after.GetChecksumProgram()) {
			add(ClassPotentialIncompatibility, key, "the checksum program changed")
		}
	}
}

func diffRouting(oldBundle, newBundle *irv1.RuleBundle, add addChange) {
	oldDispatch := dispatchKeys(oldBundle)
	newDispatch := dispatchKeys(newBundle)
	for _, key := range sortedStrings(newDispatch) {
		if _, ok := oldDispatch[key]; !ok {
			add(ClassWidening, "dispatch:"+key, "the routing entry is new")
		}
	}
	for _, key := range sortedStrings(oldDispatch) {
		if _, ok := newDispatch[key]; !ok {
			add(ClassRestriction, "dispatch:"+key, "the routing entry disappears")
		}
	}
}

func capabilityName(id uint32) string {
	if c, ok := features.Lookup(id); ok {
		return c.Name
	}
	return fmt.Sprintf("capability-%d", id)
}

func featureSet(b *irv1.RuleBundle) map[uint32]bool {
	out := map[uint32]bool{}
	for _, id := range b.GetRequiredFeatureIds() {
		out[id] = true
	}
	return out
}

func sortedIDs(m map[uint32]bool) []uint32 {
	out := make([]uint32, 0, len(m))
	for id := range m {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func definitionKeys(b *irv1.RuleBundle) map[string]*irv1.IdentifierDefinition {
	out := map[string]*irv1.IdentifierDefinition{}
	for _, d := range b.GetIdentifiers() {
		country := globalCountry
		if d.CountryCode != nil {
			country = d.GetCountryCode()
		}
		out[d.GetKind()+"/"+country] = d
	}
	return out
}

func sortedKeys(m map[string]*irv1.IdentifierDefinition) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sortedStrings(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func sourceIDs(d *irv1.IdentifierDefinition) string {
	ids := make([]string, 0, len(d.GetSources()))
	for _, s := range d.GetSources() {
		ids = append(ids, s.GetId())
	}
	return strings.Join(ids, ",")
}

func dispatchKeys(b *irv1.RuleBundle) map[string]bool {
	out := map[string]bool{}
	for _, d := range b.GetDispatchers() {
		out["kind:"+d.GetKind()] = true
		for _, alias := range d.GetKindAliases() {
			out["kind-alias:"+alias] = true
		}
		for _, ca := range d.GetCountryAliases() {
			out["country-alias:"+d.GetKind()+":"+ca.GetAlias()+"="+ca.GetCountryCode()] = true
		}
		for _, t := range d.GetTargets() {
			country := globalCountry
			if t.CountryCode != nil {
				country = t.GetCountryCode()
			}
			out["target:"+d.GetKind()+":"+country] = true
			for _, p := range t.GetAcceptedPrefixes() {
				out["prefix:"+d.GetKind()+":"+p] = true
			}
		}
	}
	return out
}

// programFingerprint renders a program in a stable textual form so that two
// bundles can be compared without depending on node numbering alone.
func programFingerprint(b *irv1.RuleBundle, id uint32) string {
	for _, p := range b.GetPrograms() {
		if p.GetId() != id {
			continue
		}
		var sb strings.Builder
		_, _ = fmt.Fprintf(&sb, "kind=%v root=%d", p.GetKind(), p.GetRootNode())
		for i, n := range p.GetNodes() {
			_, _ = fmt.Fprintf(&sb, "|%d:%s:%v", i, operationName(n), n.GetInputNodes())
		}
		return sb.String()
	}
	return "missing"
}

func runDiff(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("diff", flag.ContinueOnError)
	fs.SetOutput(stderr)
	asJSON := fs.Bool("json", false, "render the changes as JSON")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 2 {
		_, _ = fmt.Fprintln(stderr, "usage: entidc diff [--json] <old.binpb> <new.binpb>")
		return exitUsage
	}
	oldRaw, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return fail(stderr, "cannot read %s: %v", fs.Arg(0), err)
	}
	newRaw, err := os.ReadFile(fs.Arg(1))
	if err != nil {
		return fail(stderr, "cannot read %s: %v", fs.Arg(1), err)
	}
	oldRules, err := artifact.LoadRuleset(oldRaw)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s is refused: %v\n", fs.Arg(0), err)
		return exitRejected
	}
	newRules, err := artifact.LoadRuleset(newRaw)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s is refused: %v\n", fs.Arg(1), err)
		return exitRejected
	}
	changes := DiffBundles(oldRules.Bundle, newRules.Bundle)
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(struct {
			Changes []Change `json:"changes"`
		}{changes}); err != nil {
			return fail(stderr, "cannot render the changes: %v", err)
		}
		return exitOK
	}
	if len(changes) == 0 {
		_, _ = fmt.Fprintln(stdout, "the two bundles are semantically identical")
		return exitOK
	}
	for _, c := range changes {
		_, _ = fmt.Fprintf(stdout, "%-26s %-24s %s\n", c.Class, c.Subject, c.Detail)
	}
	return exitOK
}
