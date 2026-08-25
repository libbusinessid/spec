package artifact_test

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// tools/check_lock.sh is the gate an engine runs against a lock it has just
// assembled, and it verified seven of the eight digests. The one it skipped was
// conformance_jsonl_sha256 -- the very field whose absence from one writer made
// the first release ship seven digests where every engine verifies eight. Given
// a lock with that digest deliberately zeroed, it printed seven `ok` lines and
// exited 0.
//
// Two properties are needed, and the second is the one a rewrite forgets:
//
//   - every digest the lock declares is verified, and a digest the script cannot
//     place stops the run instead of being walked past;
//   - the lock must declare every digest the contract requires, or a lock of
//     seven passes by verifying seven. The count comes from the normative
//     lock-fields block in engine.md, never from a number in the script.
//
// This test runs the script rather than reading it, because both defects were
// behavioural and a reading test would have passed throughout.
func TestCheckLockRefusesEveryWayALockCanBeShort(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash is required to run tools/check_lock.sh")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "tools", "check_lock.sh"))
	if err != nil {
		t.Fatal(err)
	}

	const version = "2026.08.38"
	digests := map[string]string{
		"rules_sha256":             "entid-rules-" + version + ".binpb",
		"conformance_sha256":       "entid-conformance-" + version + ".binpb",
		"conformance_jsonl_sha256": "entid-conformance-" + version + ".jsonl.gz",
		"rules_proto_sha256":       "rules.proto",
		"conformance_proto_sha256": "conformance.proto",
		"testee_proto_sha256":      "testee.proto",
		"ir_doc_sha256":            "ir.md",
		"features_doc_sha256":      "features.md",
	}
	// The order the contract fixes; the lock is written in it.
	order := []string{
		"rules_sha256", "conformance_sha256", "conformance_jsonl_sha256",
		"rules_proto_sha256", "conformance_proto_sha256", "testee_proto_sha256",
		"ir_doc_sha256", "features_doc_sha256",
	}

	// build lays out an artifact directory and a lock, then applies one mutation.
	build := func(t *testing.T, mutate func(lock []string, dir string) []string) (string, string) {
		t.Helper()
		dir := t.TempDir()
		for field, name := range digests {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("content of "+field), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		contract := "## 16\n\n```lock-fields\nrules_version\nformat_version\n" +
			strings.Join(order, "\n") + "\nstability\nsource_commit\n```\n"
		if err := os.WriteFile(filepath.Join(dir, "engine.md"), []byte(contract), 0o600); err != nil {
			t.Fatal(err)
		}

		lock := []string{`rules_version = "` + version + `"`, "format_version = 1"}
		for _, field := range order {
			raw, err := os.ReadFile(filepath.Join(dir, digests[field]))
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			lock = append(lock, field+` = "`+hex.EncodeToString(sum[:])+`"`)
		}
		lock = append(lock, `stability = "alpha"`, `source_commit = "0"`,
			`attestation_identity = "entid-org/spec/.github/workflows/release.yml@refs/tags/v`+version+`"`)

		if mutate != nil {
			lock = mutate(lock, dir)
		}
		path := filepath.Join(dir, "rules.lock")
		if err := os.WriteFile(path, []byte(strings.Join(lock, "\n")+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path, dir
	}

	run := func(t *testing.T, lock, dir string) (string, int) {
		t.Helper()
		cmd := exec.Command("bash", script, lock, dir, version)
		out, err := cmd.CombinedOutput()
		code := 0
		var exit *exec.ExitError
		if err != nil {
			if !errors.As(err, &exit) {
				t.Fatalf("running the script: %v", err)
			}
			code = exit.ExitCode()
		}
		return string(out), code
	}

	drop := func(prefix string) func([]string, string) []string {
		return func(lock []string, _ string) []string {
			kept := lock[:0:0]
			for _, line := range lock {
				if !strings.HasPrefix(line, prefix) {
					kept = append(kept, line)
				}
			}
			return kept
		}
	}
	zero := func(prefix string) func([]string, string) []string {
		return func(lock []string, _ string) []string {
			for i, line := range lock {
				if strings.HasPrefix(line, prefix) {
					lock[i] = prefix + ` = "` + strings.Repeat("0", 64) + `"`
				}
			}
			return lock
		}
	}

	t.Run("an intact lock passes and says how many digests it verified", func(t *testing.T) {
		lock, dir := build(t, nil)
		out, code := run(t, lock, dir)
		if code != 0 {
			t.Fatalf("an intact lock must pass, got exit %d:\n%s", code, out)
		}
		if !strings.Contains(out, "ok 8 digests verified") {
			t.Errorf("the script must state the count, so a shorter run does not read as the same success:\n%s", out)
		}
	})

	for _, c := range []struct {
		name   string
		mutate func([]string, string) []string
		says   string
	}{
		{"a corrupted conformance_jsonl digest", zero("conformance_jsonl_sha256"), "conformance_jsonl_sha256 mismatch"},
		{"a corrupted rules digest", zero("rules_sha256"), "rules_sha256 mismatch"},
		{"a missing conformance_jsonl digest", drop("conformance_jsonl_sha256"), "missing digests the contract requires"},
		{"a missing features_doc digest", drop("features_doc_sha256"), "missing digests the contract requires"},
		{"a missing attestation identity", drop("attestation_identity"), "no attestation identity"},
		// Added rather than renamed: renaming a required field makes the lock
		// incomplete too, and the completeness check fires first. This case
		// exists to exercise the other path, so the lock must stay complete.
		{"a digest the script cannot place", func(lock []string, _ string) []string {
			return append(lock, `surprise_sha256 = "`+strings.Repeat("0", 64)+`"`)
		}, "does not know which file it covers"},
		{"a contract with no lock-fields block", func(lock []string, dir string) []string {
			if err := os.WriteFile(filepath.Join(dir, "engine.md"), []byte("## 16\n\nno block here\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			return lock
		}, "carries no lock-fields block"},
		{"a contract that is not there at all", func(lock []string, dir string) []string {
			if err := os.Remove(filepath.Join(dir, "engine.md")); err != nil {
				t.Fatal(err)
			}
			return lock
		}, "cannot be read and the lock cannot be judged complete"},
	} {
		t.Run(c.name+" is refused", func(t *testing.T) {
			lock, dir := build(t, c.mutate)
			out, code := run(t, lock, dir)
			if code == 0 {
				t.Fatalf("%s must be refused, the script exited 0:\n%s", c.name, out)
			}
			if !strings.Contains(out, c.says) {
				t.Errorf("%s was refused, but not for the stated reason.\n want to contain: %s\n got:\n%s",
					c.name, c.says, out)
			}
		})
	}
}
