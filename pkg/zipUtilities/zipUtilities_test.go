package zipUtilities

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestZipDirectoryContentsUsesForwardSlashes(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "assets", "img"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", filepath.Join("assets", "img", "logo.svg")} {
		if err := os.WriteFile(filepath.Join(src, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	target := filepath.Join(t.TempDir(), "app.zip")
	if err := ZipDirectoryContents(src+string(os.PathSeparator), target); err != nil {
		t.Fatalf("ZipDirectoryContents: %v", err)
	}

	zr, err := zip.OpenReader(target)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	names := []string{}
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)

	want := []string{"assets/", "assets/img/", "assets/img/logo.svg", "index.html"}
	if len(names) != len(want) {
		t.Fatalf("entries = %q, want %q", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("entries = %q, want %q", names, want)
		}
	}
}

func TestZipDirectoryContentsMissingSource(t *testing.T) {
	target := filepath.Join(t.TempDir(), "app.zip")
	if err := ZipDirectoryContents(filepath.Join(t.TempDir(), "missing"), target); err == nil {
		t.Fatal("expected an error for a missing source folder")
	}
}
