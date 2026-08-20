package main

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/libbusinessid/spec/internal/register"
)

// manifestPath describes where each issuer publishes its complete register.
// The dumps are never committed; this is.
const manifestPath = "conformance/registers.json"

type manifest struct {
	Registers []register.Definition `json:"registers"`
}

// parseSweep reads a --register argument of the form <id>=<path> and pairs the
// file with the manifest entry that says how to read it.
func parseSweep(arg, manifestFile string) (register.Definition, string, error) {
	id, path, found := strings.Cut(arg, "=")
	if !found || id == "" || path == "" {
		return register.Definition{}, "", fmt.Errorf(
			"--register expects <id>=<file>, got %q", arg)
	}
	raw, err := os.ReadFile(filepath.Clean(manifestFile))
	if err != nil {
		return register.Definition{}, "", fmt.Errorf("cannot read %s: %w", manifestFile, err)
	}
	var m manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return register.Definition{}, "", fmt.Errorf("%s is not valid JSON: %w", manifestFile, err)
	}
	known := make([]string, 0, len(m.Registers))
	for _, d := range m.Registers {
		if d.ID == id {
			return d, path, nil
		}
		known = append(known, d.ID)
	}
	return register.Definition{}, "", fmt.Errorf(
		"%s describes no register %q; it knows %s", manifestFile, id, strings.Join(known, ", "))
}

// openDump opens a register dump, transparently decompressing a gzip file.
//
// A plain CSV and a gzipped one are both what an issuer hands out, and asking
// the operator to remember which is which invites the mistake of sweeping an
// empty file and reading the silence as success.
func openDump(path string) (io.ReadCloser, error) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("cannot open the dump: %w", err)
	}
	if !strings.HasSuffix(path, ".gz") {
		return f, nil
	}
	zr, err := gzip.NewReader(f)
	if err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("cannot read %s as gzip: %w", path, err)
	}
	return struct {
		io.Reader
		io.Closer
	}{Reader: zr, Closer: f}, nil
}
