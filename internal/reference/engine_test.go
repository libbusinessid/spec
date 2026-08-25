package reference_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	irv1 "github.com/entid-org/spec/gen/go/entid/ir/v1"
	"github.com/entid-org/spec/internal/reference"
)

func TestNewEngineFromBytes(t *testing.T) {
	root := repoRoot(t)
	raw, err := os.ReadFile(filepath.Join(root, "testdata", "bundles", "minimal_valid.binpb"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := reference.NewEngine(raw)
	if err != nil {
		t.Fatal(err)
	}
	report, err := e.Validate(reference.Input{Kind: "demo", Value: " 1230 "}, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Format.Status != reference.StatusValid || report.Checksum.Status != reference.StatusValid {
		t.Fatalf("unexpected report %+v", report)
	}
	if _, err := reference.NewEngine([]byte{0xff}); err == nil {
		t.Fatal("a broken bundle must be refused")
	}
}

// stubProvider is the only registry provider of the repository: it exists to
// prove the interface is usable, and it never performs any I/O.
type stubProvider struct {
	supports bool
	fail     bool
}

func (p stubProvider) Supports(string, *string) bool { return p.supports }

func (p stubProvider) Lookup(canonical string, _ reference.Input) (reference.RegistryResult, error) {
	if p.fail {
		return reference.RegistryResult{}, errors.New("transport failure")
	}
	return reference.RegistryResult{
		Status:         reference.RegistryNotFound,
		ProviderID:     "stub",
		CheckedAt:      "2026-08-18T00:00:00Z",
		CanonicalValue: canonical,
		ReasonCode:     irv1.ReasonCode_REASON_CODE_OK,
		Metadata:       map[string]string{"provider": "stub"},
	}, nil
}

func TestRegistryProviderInterface(t *testing.T) {
	e := engine(t)
	in := reference.Input{Kind: "siren", Value: "012345674"}

	got, err := e.RegistryLookup(in, stubProvider{supports: false}, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != reference.RegistryUnsupported ||
		got.ReasonCode != irv1.ReasonCode_REASON_CODE_REGISTRY_NOT_CONFIGURED {
		t.Fatalf("an unsupported provider must report registry_not_configured: %+v", got)
	}

	got, err = e.RegistryLookup(in, stubProvider{supports: true}, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != reference.RegistryNotFound || got.CanonicalValue != "012345674" {
		t.Fatalf("unexpected registry result %+v", got)
	}

	if _, err := e.RegistryLookup(in, stubProvider{supports: true, fail: true}, reference.Options{}); err == nil {
		t.Fatal("a technical failure must stay an error, never a business result")
	}
}

func TestEngineExposesItsRuleset(t *testing.T) {
	e := engine(t)
	if e.Ruleset() == nil || len(e.Ruleset().RequiredFeatures()) == 0 {
		t.Fatal("the ruleset must be inspectable")
	}
}

func TestValidateChecksumOnAnAbsentAlgorithm(t *testing.T) {
	e := engine(t)
	report, err := e.ValidateChecksum(reference.Input{Kind: "vat", Value: "DE123456789"}, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Checksum.Status != reference.StatusUnsupported ||
		report.Checksum.ReasonCode != irv1.ReasonCode_REASON_CODE_UNSUPPORTED_CHECKSUM {
		t.Fatalf("unexpected checksum %+v", report.Checksum)
	}
	if report.Checksum.MessageKey != nil {
		t.Fatal("an engine produced result carries no message key")
	}
}

func TestProfileDefaultsToTheDefinition(t *testing.T) {
	e := engine(t)
	report, err := e.Validate(reference.Input{Kind: "siren", Value: "012345674"}, reference.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if report.Profile != reference.ProfileCompatible {
		t.Fatalf("unexpected profile %q", report.Profile)
	}
	canon, err := e.Canonicalize(reference.Input{Kind: "siren", Value: "012345674"},
		reference.Options{Profile: reference.ProfileStrictCurrent})
	if err != nil {
		t.Fatal(err)
	}
	if canon.Profile != reference.ProfileStrictCurrent {
		t.Fatalf("unexpected profile %q", canon.Profile)
	}
}
