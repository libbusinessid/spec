package artifact

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// Manifest is the external release manifest. It is the only artifact carrying a
// timestamp, always derived from SOURCE_DATE_EPOCH in a reproducible build.
type Manifest struct {
	RulesVersion              string            `json:"rulesVersion"`
	FormatVersion             uint32            `json:"formatVersion"`
	RequiredFeatures          []Feature         `json:"requiredFeatures"`
	SourceDigest              string            `json:"sourceDigest"`
	ConformanceSourceDigest   string            `json:"conformanceSourceDigest"`
	ArtifactSha256            string            `json:"artifactSha256"`
	ConformanceSha256         string            `json:"conformanceSha256"`
	ConformanceJsonlGzSha256  string            `json:"conformanceJsonlGzSha256"`
	CompilerVersion           string            `json:"compilerVersion"`
	SourceCommit              string            `json:"sourceCommit"`
	GeneratedAt               string            `json:"generatedAt"`
	IdentifierCount           int               `json:"identifierCount"`
	CountryCount              int               `json:"countryCount"`
	CoverageByKind            []KindCoverage    `json:"coverageByKind"`
	MinimumEngineCapabilities []uint32          `json:"minimumEngineCapabilities"`
	RulesProtoSha256          string            `json:"rulesProtoSha256"`
	ConformanceProtoSha256    string            `json:"conformanceProtoSha256"`
	IrDocSha256               string            `json:"irDocSha256"`
	FeaturesDocSha256         string            `json:"featuresDocSha256"`
	Reproducible              bool              `json:"reproducible"`
	ConformanceCaseCount      int               `json:"conformanceCaseCount"`
	ConformanceTagCounts      map[string]int    `json:"conformanceTagCounts"`
	Extra                     map[string]string `json:"extra,omitempty"`
}

// Feature is one declared capability of the bundle.
type Feature struct {
	ID   uint32 `json:"id"`
	Name string `json:"name"`
}

// KindCoverage summarizes the coverage of one identifier kind.
type KindCoverage struct {
	Kind             string   `json:"kind"`
	Countries        []string `json:"countries"`
	WithChecksum     int      `json:"withChecksum"`
	WithoutChecksum  int      `json:"withoutChecksum"`
	ConformanceCases int      `json:"conformanceCases"`
}

// HexDigest renders a 32 byte digest as 64 lower case hexadecimal characters.
func HexDigest(d [32]byte) string { return hex.EncodeToString(d[:]) }

// FormatEpoch renders a SOURCE_DATE_EPOCH value as an RFC 3339 UTC timestamp.
func FormatEpoch(epoch int64) string {
	return time.Unix(epoch, 0).UTC().Format(time.RFC3339)
}

// Encode renders the manifest as stable, indented JSON terminated by a newline.
func (m *Manifest) Encode() ([]byte, error) {
	sort.Slice(m.RequiredFeatures, func(i, j int) bool { return m.RequiredFeatures[i].ID < m.RequiredFeatures[j].ID })
	sort.Slice(m.CoverageByKind, func(i, j int) bool { return m.CoverageByKind[i].Kind < m.CoverageByKind[j].Kind })
	sort.Slice(m.MinimumEngineCapabilities, func(i, j int) bool {
		return m.MinimumEngineCapabilities[i] < m.MinimumEngineCapabilities[j]
	})
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("cannot render the manifest: %w", err)
	}
	return append(out, '\n'), nil
}
