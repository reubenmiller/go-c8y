package zipUtilities

import (
	"archive/zip"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeTestApp(t *testing.T) string {
	t.Helper()
	src := filepath.Join(t.TempDir(), "build")
	if err := os.MkdirAll(filepath.Join(src, "assets", "img"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"index.html", filepath.Join("assets", "img", "logo.svg")} {
		if err := os.WriteFile(filepath.Join(src, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return src
}

func zipNames(t *testing.T, path string) []string {
	t.Helper()
	zr, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()
	names := []string{}
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	return names
}

func assertNames(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("entries = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("entries = %q, want %q", got, want)
		}
	}
}

// Entry names use '/' and are relative, whether or not the source folder has
// a trailing separator.
func TestZipDirectoryContentsUsesForwardSlashes(t *testing.T) {
	src := writeTestApp(t)
	want := []string{"assets/", "assets/img/", "assets/img/logo.svg", "index.html"}

	for _, source := range []string{src, src + string(os.PathSeparator)} {
		target := filepath.Join(t.TempDir(), "app.zip")
		if err := ZipDirectoryContents(source, target); err != nil {
			t.Fatalf("ZipDirectoryContents(%q): %v", source, err)
		}
		assertNames(t, zipNames(t, target), want)
	}
}

func TestZipDirectoryIncludesRoot(t *testing.T) {
	src := writeTestApp(t)
	target := filepath.Join(t.TempDir(), "app.zip")
	if err := ZipDirectory(src, target); err != nil {
		t.Fatalf("ZipDirectory: %v", err)
	}
	assertNames(t, zipNames(t, target), []string{"build/", "build/assets/", "build/assets/img/", "build/assets/img/logo.svg", "build/index.html"})
}

func TestUnzipFileRoundTrip(t *testing.T) {
	src := writeTestApp(t)
	target := filepath.Join(t.TempDir(), "app.zip")
	if err := ZipDirectoryContents(src, target); err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	if _, err := UnzipFile(target, dest, nil); err != nil {
		t.Fatalf("UnzipFile: %v", err)
	}
	for _, name := range []string{"index.html", filepath.Join("assets", "img", "logo.svg")} {
		if _, err := os.Stat(filepath.Join(dest, name)); err != nil {
			t.Errorf("missing %s after unzip: %v", name, err)
		}
	}
}

func TestZipDirectoryContentsMissingSource(t *testing.T) {
	target := filepath.Join(t.TempDir(), "app.zip")
	if err := ZipDirectoryContents(filepath.Join(t.TempDir(), "missing"), target); err == nil {
		t.Fatal("expected an error for a missing source folder")
	}
}
