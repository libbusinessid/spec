// Package hcllang parses the LibBusinessID HCL surface language into the typed
// AST of internal/ast. It never evaluates HCL expressions and never resolves a
// symbol: every reference is kept structural for the linker.
package hcllang

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// SourceFile is one discovered source file.
type SourceFile struct {
	// AbsPath is the path used to read the file.
	AbsPath string
	// RelPath is the POSIX relative path inside the discovery root. It is the
	// only path that ever reaches a diagnostic, a digest or an artifact.
	RelPath string
}

// Discover walks root and returns every `*.hcl` file, sorted by the UTF-8 bytes
// of its POSIX relative path. Hidden directories, `dist` and `vendor` are
// skipped. Symbolic links are refused, at the root and at any depth.
func Discover(root string) ([]SourceFile, error) {
	return discover(root, ".hcl")
}

// DiscoverExt walks root and returns every file with the given extension.
func DiscoverExt(root, ext string) ([]SourceFile, error) {
	return discover(root, ext)
}

func discover(root, ext string) ([]SourceFile, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return nil, err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return nil, fmt.Errorf("%s: symbolic links are refused", root)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s: not a directory", root)
	}

	var out []SourceFile
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s: symbolic links are refused", rel)
		}
		if d.IsDir() {
			if path == root {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, ".") || name == "dist" || name == "vendor" {
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return fmt.Errorf("%s: only regular files are accepted", rel)
		}
		if filepath.Ext(path) != ext {
			return nil
		}
		out = append(out, SourceFile{AbsPath: path, RelPath: rel})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelPath < out[j].RelPath })
	for i := 1; i < len(out); i++ {
		if out[i].RelPath == out[i-1].RelPath {
			return nil, fmt.Errorf("%s: duplicate relative path", out[i].RelPath)
		}
	}
	return out, nil
}
