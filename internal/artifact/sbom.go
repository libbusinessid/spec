package artifact

import (
	"bufio"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SPDX constants shared by every package entry of the document.
const (
	spdxRootPackage = "SPDXRef-Package-root"
	spdxNoAssertion = "NOASSERTION"
	spdxPackageRefs = "PACKAGE-MANAGER"
)

// SPDXDocument is the minimal SPDX 2.3 document published with a release.
type SPDXDocument struct {
	SPDXVersion       string             `json:"spdxVersion"`
	DataLicense       string             `json:"dataLicense"`
	SPDXID            string             `json:"SPDXID"`
	Name              string             `json:"name"`
	DocumentNamespace string             `json:"documentNamespace"`
	CreationInfo      SPDXCreationInfo   `json:"creationInfo"`
	Packages          []SPDXPackage      `json:"packages"`
	Relationships     []SPDXRelationship `json:"relationships"`
}

// SPDXCreationInfo records how the document was produced.
type SPDXCreationInfo struct {
	Created  string   `json:"created"`
	Creators []string `json:"creators"`
}

// SPDXPackage is one module of the dependency graph.
type SPDXPackage struct {
	SPDXID           string       `json:"SPDXID"`
	Name             string       `json:"name"`
	VersionInfo      string       `json:"versionInfo"`
	DownloadLocation string       `json:"downloadLocation"`
	FilesAnalyzed    bool         `json:"filesAnalyzed"`
	LicenseConcluded string       `json:"licenseConcluded"`
	LicenseDeclared  string       `json:"licenseDeclared"`
	Supplier         string       `json:"supplier"`
	ExternalRefs     []SPDXExtRef `json:"externalRefs,omitempty"`
}

// SPDXExtRef carries the package URL of a module.
type SPDXExtRef struct {
	ReferenceCategory string `json:"referenceCategory"`
	ReferenceType     string `json:"referenceType"`
	ReferenceLocator  string `json:"referenceLocator"`
}

// SPDXRelationship links two SPDX elements.
type SPDXRelationship struct {
	SPDXElementID      string `json:"spdxElementId"`
	RelationshipType   string `json:"relationshipType"`
	RelatedSPDXElement string `json:"relatedSpdxElement"`
}

// Module is one entry of a parsed `go.mod` file.
type Module struct {
	Path    string
	Version string
}

// ParseGoMod extracts the module path and every required module of a `go.mod`.
// It is a small deterministic parser: the SBOM must not depend on the build
// information of the running binary, which differs between a test binary and a
// released one.
func ParseGoMod(content string) (string, []Module, error) {
	var modulePath string
	var modules []Module
	scanner := bufio.NewScanner(strings.NewReader(content))
	inRequireBlock := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if idx := strings.Index(line, "//"); idx >= 0 {
			line = strings.TrimSpace(line[:idx])
		}
		switch {
		case line == "":
		case strings.HasPrefix(line, "module "):
			modulePath = strings.TrimSpace(strings.TrimPrefix(line, "module "))
		case line == "require (":
			inRequireBlock = true
		case inRequireBlock && line == ")":
			inRequireBlock = false
		case inRequireBlock:
			if m, ok := parseModuleLine(line); ok {
				modules = append(modules, m)
			}
		case strings.HasPrefix(line, "require "):
			if m, ok := parseModuleLine(strings.TrimPrefix(line, "require ")); ok {
				modules = append(modules, m)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", nil, err
	}
	if modulePath == "" {
		return "", nil, fmt.Errorf("go.mod declares no module path")
	}
	sort.Slice(modules, func(i, j int) bool {
		if modules[i].Path != modules[j].Path {
			return modules[i].Path < modules[j].Path
		}
		return modules[i].Version < modules[j].Version
	})
	return modulePath, modules, nil
}

func parseModuleLine(line string) (Module, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return Module{}, false
	}
	return Module{Path: fields[0], Version: fields[1]}, true
}

// RenderSBOM builds the SPDX document of a release.
func RenderSBOM(modulePath string, modules []Module, rulesVersion, compilerVersion, generatedAt string) ([]byte, error) {
	doc := SPDXDocument{
		SPDXVersion:       "SPDX-2.3",
		DataLicense:       "CC0-1.0",
		SPDXID:            "SPDXRef-DOCUMENT",
		Name:              "libbusinessid-rules-" + rulesVersion,
		DocumentNamespace: "https://" + modulePath + "/sbom/" + rulesVersion,
		CreationInfo: SPDXCreationInfo{
			Created:  generatedAt,
			Creators: []string{"Tool: businessidc-" + compilerVersion},
		},
	}
	doc.Packages = append(doc.Packages, SPDXPackage{
		SPDXID:           spdxRootPackage,
		Name:             modulePath,
		VersionInfo:      rulesVersion,
		DownloadLocation: "https://" + modulePath,
		FilesAnalyzed:    false,
		LicenseConcluded: "Apache-2.0",
		LicenseDeclared:  "Apache-2.0",
		Supplier:         "Organization: LibBusinessID",
		ExternalRefs: []SPDXExtRef{{
			ReferenceCategory: spdxPackageRefs,
			ReferenceType:     "purl",
			ReferenceLocator:  "pkg:golang/" + modulePath + "@" + rulesVersion,
		}},
	})
	doc.Relationships = append(doc.Relationships, SPDXRelationship{
		SPDXElementID:      "SPDXRef-DOCUMENT",
		RelationshipType:   "DESCRIBES",
		RelatedSPDXElement: spdxRootPackage,
	})
	for i, m := range modules {
		id := fmt.Sprintf("SPDXRef-Package-%d", i+1)
		doc.Packages = append(doc.Packages, SPDXPackage{
			SPDXID:           id,
			Name:             m.Path,
			VersionInfo:      m.Version,
			DownloadLocation: "https://proxy.golang.org/" + m.Path + "/@v/" + m.Version + ".zip",
			FilesAnalyzed:    false,
			LicenseConcluded: spdxNoAssertion,
			LicenseDeclared:  spdxNoAssertion,
			Supplier:         spdxNoAssertion,
			ExternalRefs: []SPDXExtRef{{
				ReferenceCategory: spdxPackageRefs,
				ReferenceType:     "purl",
				ReferenceLocator:  "pkg:golang/" + m.Path + "@" + m.Version,
			}},
		})
		doc.Relationships = append(doc.Relationships, SPDXRelationship{
			SPDXElementID:      spdxRootPackage,
			RelationshipType:   "DEPENDS_ON",
			RelatedSPDXElement: id,
		})
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}
