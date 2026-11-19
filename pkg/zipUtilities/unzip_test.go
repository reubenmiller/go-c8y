package zipUtilities

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func writeTestZip(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.zip")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, contents := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestUnzipFileOnlyExtractsNamedFiles(t *testing.T) {
	src := writeTestZip(t, map[string]string{
		"cumulocity.json": `{"name":"app"}`,
		"index.html":      "<html></html>",
		"assets/app.js":   "console.log(1)",
	})
	dest := t.TempDir()

	// Names match case-insensitively
	files, err := UnzipFile(src, dest, []string{"CUMULOCITY.json"})
	if err != nil {
		t.Fatalf("UnzipFile: %v", err)
	}
	if len(files) != 1 || files[0] != filepath.Join(dest, "cumulocity.json") {
		t.Errorf("files = %q, want only cumulocity.json", files)
	}
	if !exists(filepath.Join(dest, "cumulocity.json")) {
		t.Error("cumulocity.json was not extracted")
	}
	for _, name := range []string{"index.html", filepath.Join("assets", "app.js")} {
		if exists(filepath.Join(dest, name)) {
			t.Errorf("%s was extracted although it was not named", name)
		}
	}
}

func TestUnzipFileWithoutNamesExtractsEverything(t *testing.T) {
	src := writeTestZip(t, map[string]string{
		"index.html":    "<html></html>",
		"assets/app.js": "console.log(1)",
	})
	dest := t.TempDir()

	files, err := UnzipFile(src, dest, nil)
	if err != nil {
		t.Fatalf("UnzipFile: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("files = %q, want 2 files", files)
	}
	for _, name := range []string{"index.html", filepath.Join("assets", "app.js")} {
		if !exists(filepath.Join(dest, name)) {
			t.Errorf("%s was not extracted", name)
		}
	}
}

func TestUnzipFileRejectsZipSlip(t *testing.T) {
	src := writeTestZip(t, map[string]string{"../evil.txt": "x"})
	dest := filepath.Join(t.TempDir(), "out")

	if _, err := UnzipFile(src, dest, nil); err == nil {
		t.Fatal("expected an error for an entry outside the destination")
	}
	if exists(filepath.Join(filepath.Dir(dest), "evil.txt")) {
		t.Error("entry was written outside the destination")
	}
}
