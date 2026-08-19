package artifact_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	irv1 "github.com/libbusinessid/spec/gen/go/libbusinessid/ir/v1"
	"github.com/libbusinessid/spec/internal/artifact"
	"github.com/libbusinessid/spec/internal/features"
)

func loadMinimal(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRootFor(t), "testdata", "bundles", "minimal_valid.binpb"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestRulesetAccessors(t *testing.T) {
	rules, err := artifact.LoadRuleset(loadMinimal(t))
	if err != nil {
		t.Fatal(err)
	}
	if rules.RulesVersion() != "2026.08.0" || rules.FormatVersion() != artifact.SupportedFormatVersion {
		t.Fatalf("unexpected versions: %s/%d", rules.RulesVersion(), rules.FormatVersion())
	}
	got := rules.RequiredFeatures()
	if len(got) == 0 || !features.Known(got[0]) {
		t.Fatalf("unexpected capabilities %v", got)
	}
	got[0] = 9999
	if rules.RequiredFeatures()[0] == 9999 {
		t.Fatal("RequiredFeatures must return a copy")
	}
	if _, ok := rules.DispatcherByKind["demo"]; !ok {
		t.Fatal("the dispatcher index is missing")
	}
}

func TestEveryDecoderFixtureIsClassified(t *testing.T) {
	dir := filepath.Join(repoRootFor(t), "testdata", "bundles")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, e := range entries {
		name := e.Name()
		if name == "minimal_valid.binpb" || name == "minimal_conformance.binpb" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		_, err = artifact.LoadRuleset(data)
		if err == nil {
			t.Errorf("%s must be refused", name)
			continue
		}
		typed, ok := err.(*artifact.Error) //nolint:errorlint // the loader returns this exact type
		if !ok {
			t.Errorf("%s: the loader must return a typed error", name)
			continue
		}
		if typed.Kind != artifact.ErrInvalid && typed.Kind != artifact.ErrIncompatible {
			t.Errorf("%s: unexpected kind %q", name, typed.Kind)
		}
		if !strings.Contains(typed.Error(), string(typed.Kind)) {
			t.Errorf("%s: the message must carry the kind", name)
		}
		seen++
	}
	if seen < 20 {
		t.Fatalf("only %d decoder fixtures were exercised", seen)
	}
}

func TestBundleSizeLimit(t *testing.T) {
	huge := make([]byte, 17*1024*1024)
	_, err := artifact.LoadRuleset(huge)
	if err == nil || !strings.Contains(err.Error(), "exceeds the limit") {
		t.Fatalf("an oversized bundle must be refused, got %v", err)
	}
}

func TestFeatureIDsMustBeAscending(t *testing.T) {
	bundle := &irv1.RuleBundle{
		FormatVersion:      artifact.SupportedFormatVersion,
		RulesVersion:       "2026.08.0",
		RequiredFeatureIds: []uint32{features.CoreGraphV1, features.CoreGraphV1},
		SourceDigest:       make([]byte, 32),
	}
	raw, err := artifact.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifact.LoadRuleset(raw); err == nil ||
		!strings.Contains(err.Error(), "strictly ascending") {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestMarshalIsDeterministic(t *testing.T) {
	data := loadMinimal(t)
	rules, err := artifact.LoadRuleset(data)
	if err != nil {
		t.Fatal(err)
	}
	first, err := artifact.Marshal(rules.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	second, err := artifact.Marshal(rules.Bundle)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("two serializations of the same message must produce the same bytes")
	}
	if artifact.SHA256Hex(first) != artifact.SHA256Hex(second) {
		t.Fatal("the digests must match")
	}
}

func TestGzipIsReproducible(t *testing.T) {
	payload := []byte(strings.Repeat("libbusinessid\n", 100))
	first, err := artifact.Gzip(payload, 1755475200)
	if err != nil {
		t.Fatal(err)
	}
	second, err := artifact.Gzip(payload, 1755475200)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("gzip must be reproducible")
	}
	other, err := artifact.Gzip(payload, 0)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(first, other) {
		t.Fatal("the embedded mtime must follow SOURCE_DATE_EPOCH")
	}
	if first[9] != 255 {
		t.Fatalf("the OS byte must be 255, got %d", first[9])
	}
	reader, err := gzip.NewReader(bytes.NewReader(first))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded, payload) {
		t.Fatal("the payload must round-trip")
	}
	if reader.Name != "" || reader.Comment != "" {
		t.Fatal("the header must carry no name and no comment")
	}
}

func TestWriteFileAtomic(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "nested", "artifact.bin")
	if err := artifact.WriteFileAtomic(target, []byte("first")); err != nil {
		t.Fatal(err)
	}
	if err := artifact.WriteFileAtomic(target, []byte("second")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "second" {
		t.Fatalf("unexpected content %q", got)
	}
	entries, err := os.ReadDir(filepath.Dir(target))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("the temporary file must be removed, found %d entries", len(entries))
	}
	if err := artifact.WriteFileAtomic(filepath.Join(target, "child"), nil); err == nil {
		t.Fatal("writing below a regular file must fail")
	}
}

func TestManifestEncodeSortsAndTerminates(t *testing.T) {
	m := &artifact.Manifest{
		RulesVersion: "2026.08.0",
		RequiredFeatures: []artifact.Feature{
			{ID: 30, Name: "CHECKSUM_TRISTATE_V1"},
			{ID: 1, Name: "CORE_GRAPH_V1"},
		},
		CoverageByKind: []artifact.KindCoverage{
			{Kind: "vat"}, {Kind: "euid"},
		},
		MinimumEngineCapabilities: []uint32{30, 1},
	}
	encoded, err := m.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if encoded[len(encoded)-1] != '\n' {
		t.Fatal("the manifest must end with a newline")
	}
	var decoded artifact.Manifest
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.RequiredFeatures[0].ID != 1 || decoded.CoverageByKind[0].Kind != "euid" ||
		decoded.MinimumEngineCapabilities[0] != 1 {
		t.Fatalf("the manifest is not sorted: %+v", decoded)
	}
}

func TestFormatEpochAndHexDigest(t *testing.T) {
	if got := artifact.FormatEpoch(0); got != "1970-01-01T00:00:00Z" {
		t.Fatalf("unexpected timestamp %q", got)
	}
	var digest [32]byte
	digest[31] = 0xff
	if got := artifact.HexDigest(digest); len(got) != 64 || !strings.HasSuffix(got, "ff") {
		t.Fatalf("unexpected digest %q", got)
	}
}

func TestParseGoModAndSBOM(t *testing.T) {
	goMod := `module github.com/libbusinessid/spec

go 1.24.0

require (
	github.com/hashicorp/hcl/v2 v2.24.0 // indirect
	google.golang.org/protobuf v1.36.10
)

require github.com/zclconf/go-cty v1.16.3 // indirect
`
	path, modules, err := artifact.ParseGoMod(goMod)
	if err != nil {
		t.Fatal(err)
	}
	if path != "github.com/libbusinessid/spec" || len(modules) != 3 {
		t.Fatalf("unexpected parse: %q %+v", path, modules)
	}
	if modules[0].Path != "github.com/hashicorp/hcl/v2" {
		t.Fatalf("modules must be sorted: %+v", modules)
	}
	if _, _, err := artifact.ParseGoMod("go 1.24.0\n"); err == nil {
		t.Fatal("a go.mod without module path must be refused")
	}
	sbom, err := artifact.RenderSBOM(path, modules, "2026.08.0", "1.0.0", "2026-08-18T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	var doc artifact.SPDXDocument
	if err := json.Unmarshal(sbom, &doc); err != nil {
		t.Fatal(err)
	}
	if doc.SPDXVersion != "SPDX-2.3" || len(doc.Packages) != 4 || len(doc.Relationships) != 4 {
		t.Fatalf("unexpected SBOM: %+v", doc)
	}
	if doc.Packages[0].SPDXID != "SPDXRef-Package-root" {
		t.Fatalf("the root package must come first: %+v", doc.Packages[0])
	}
}

func TestRenderedDocumentsAreStable(t *testing.T) {
	ir := artifact.RenderIRDoc()
	if !bytes.Equal(ir, artifact.RenderIRDoc()) {
		t.Fatal("ir.md must be deterministic")
	}
	featuresDoc := artifact.RenderFeaturesDoc()
	if !bytes.Equal(featuresDoc, artifact.RenderFeaturesDoc()) {
		t.Fatal("features.md must be deterministic")
	}
	for _, op := range features.Ops() {
		if !bytes.Contains(ir, []byte("`"+op.Symbol+"`")) {
			t.Fatalf("ir.md does not document %s", op.Symbol)
		}
	}
	for _, c := range features.All() {
		if !bytes.Contains(featuresDoc, []byte("`"+c.Name+"`")) {
			t.Fatalf("features.md does not document %s", c.Name)
		}
	}
	for _, name := range []string{"whitespace_v1", "CHAR_MAPPING_ALNUM_BASE36", "invalid_ruleset"} {
		if !bytes.Contains(ir, []byte(name)) {
			t.Fatalf("ir.md does not mention %s", name)
		}
	}
}

func TestStabilityLevels(t *testing.T) {
	for _, ok := range []string{artifact.StabilityAlpha, artifact.StabilityBeta, artifact.StabilityStable} {
		if !artifact.ValidStability(ok) {
			t.Fatalf("%q must be a valid level", ok)
		}
	}
	for _, bad := range []string{"", "ALPHA", "rc", "1.0"} {
		if artifact.ValidStability(bad) {
			t.Fatalf("%q must be refused", bad)
		}
	}
	// Only a stable release leaves the pre-release state: the level describes
	// the maturity of the contract, never the extent of the rule coverage.
	if !artifact.IsPreRelease(artifact.StabilityAlpha) || !artifact.IsPreRelease(artifact.StabilityBeta) {
		t.Fatal("alpha and beta are pre-releases")
	}
	if artifact.IsPreRelease(artifact.StabilityStable) {
		t.Fatal("a stable release is not a pre-release")
	}
}
