package main

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, name, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestCoverageFloors(t *testing.T) {
	profile := writeFile(t, "coverage.out", `mode: set
example.com/m/a/a.go:1.1,2.2 3 1
example.com/m/a/a.go:3.1,4.2 1 0
example.com/m/a/b.go:1.1,2.2 6 1
example.com/m/b/b.go:1.1,2.2 2 0
example.com/m/b/b.go:1.1,2.2 2 3
`)
	cov, err := readProfile(profile)
	if err != nil {
		t.Fatal(err)
	}
	if got := percent(cov["example.com/m/a"]); got != 90 {
		t.Errorf("a = %v%%, want 90 (9 of 10 statements)", got)
	}
	// A block which appears twice is covered when one entry ran.
	if got := percent(cov["example.com/m/b"]); got != 100 {
		t.Errorf("b = %v%%, want 100", got)
	}

	floors, err := readFloors(writeFile(t, "floors.txt", "# comment\nexample.com/m/a 90\nexample.com/m/b 100\n"))
	if err != nil {
		t.Fatal(err)
	}
	if failed := check(floors, cov, os.Stderr); failed != 0 {
		t.Errorf("check failed %d packages at the exact floor", failed)
	}
	floors["example.com/m/a"] = 91
	floors["example.com/m/missing"] = 10
	if failed := check(floors, cov, os.Stderr); failed != 2 {
		t.Errorf("check failed %d packages, want 2 (below the floor, and missing)", failed)
	}
}

func TestWriteFloorsKeepsAMargin(t *testing.T) {
	cov := map[string]counts{
		"example.com/m/api/x": {covered: 2, total: 3},
		"example.com/m/other": {covered: 1, total: 1},
	}
	out := filepath.Join(t.TempDir(), "floors.txt")
	if err := writeFloors(out, cov, "example.com/m/api/"); err != nil {
		t.Fatal(err)
	}
	floors, err := readFloors(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(floors) != 1 || floors["example.com/m/api/x"] != 64 {
		t.Errorf("floors = %v, want only api/x at 64", floors)
	}
}
