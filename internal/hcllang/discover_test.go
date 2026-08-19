package hcllang_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/libbusinessid/spec/internal/hcllang"
)

func TestDiscoverSortsAndSkips(t *testing.T) {
	root := t.TempDir()
	write := func(rel string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("# x\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("z/last.hcl")
	write("a/first.hcl")
	write("a/note.txt")
	write("dist/ignored.hcl")
	write("vendor/ignored.hcl")
	write(".hidden/ignored.hcl")

	files, err := hcllang.Discover(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a/first.hcl", "z/last.hcl"}
	if len(files) != len(want) {
		t.Fatalf("got %d files: %#v", len(files), files)
	}
	for i := range want {
		if files[i].RelPath != want[i] {
			t.Fatalf("position %d: got %q, want %q", i, files[i].RelPath, want[i])
		}
	}
}

func TestDiscoverRefusesSymlinks(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "real.hcl")
	if err := os.WriteFile(target, []byte("# x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(root, "link.hcl")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := hcllang.Discover(root); err == nil {
		t.Fatal("symlinks must be refused")
	}
}

func TestDiscoverRefusesSymlinkedRoot(t *testing.T) {
	base := t.TempDir()
	targetDir := filepath.Join(base, "real")
	if err := os.Mkdir(targetDir, 0o750); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "link")
	if err := os.Symlink(targetDir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := hcllang.Discover(link); err == nil {
		t.Fatal("a symlinked root must be refused")
	}
}

func TestDiscoverRejectsNonDirectoryRoot(t *testing.T) {
	root := t.TempDir()
	f := filepath.Join(root, "file")
	if err := os.WriteFile(f, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := hcllang.Discover(f); err == nil {
		t.Fatal("a file root must be refused")
	}
	if _, err := hcllang.Discover(filepath.Join(root, "missing")); err == nil {
		t.Fatal("a missing root must be refused")
	}
}

func TestDiscoverExt(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "cases.jsonl"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	files, err := hcllang.DiscoverExt(root, ".jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].RelPath != "cases.jsonl" {
		t.Fatalf("unexpected files %#v", files)
	}
}
