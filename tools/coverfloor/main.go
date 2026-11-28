// Command coverfloor fails when the statement coverage of a package falls below
// its floor.
//
// It reads a Go coverage profile (the coverage.out written by `task test-ci`)
// and a floors file with one "<package path> <percent>" pair per line. Packages
// which are not in the floors file are not checked, so a new package can be
// added with a floor of its own.
//
//	go run ./tools/coverfloor -profile coverage.out -floors tools/coverfloor/floors.txt
//	go run ./tools/coverfloor -profile coverage.out -floors tools/coverfloor/floors.txt -update
//
// With -update the floors file is rewritten from the profile: every package
// under -prefix gets its current coverage, rounded down to a whole percent minus
// a margin of two points, because a few paths (timeouts, retries) run in some
// runs only.
// Floors only ever go up in a normal change; lowering one needs a reason in the
// pull request.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"math"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
)

// floorMargin is the number of points -update keeps below the measured coverage.
const floorMargin = 2

type counts struct{ covered, total int }

func main() {
	profile := flag.String("profile", "coverage.out", "coverage profile")
	floorsFile := flag.String("floors", "tools/coverfloor/floors.txt", "floors file")
	update := flag.Bool("update", false, "rewrite the floors file from the profile")
	prefix := flag.String("prefix", "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/", "package path prefix which -update writes floors for")
	flag.Parse()

	cov, err := readProfile(*profile)
	if err != nil {
		fatal(err)
	}

	if *update {
		if err := writeFloors(*floorsFile, cov, *prefix); err != nil {
			fatal(err)
		}
		return
	}

	floors, err := readFloors(*floorsFile)
	if err != nil {
		fatal(err)
	}
	failed := check(floors, cov, os.Stdout)
	if failed > 0 {
		fmt.Printf("%d package(s) are below their coverage floor\n", failed)
		os.Exit(1)
	}
	fmt.Printf("coverage floors ok (%d packages)\n", len(floors))
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "coverfloor:", err)
	os.Exit(2)
}

// check compares the coverage of each package with its floor and reports the
// packages below it. It returns the number of failures.
func check(floors map[string]float64, cov map[string]counts, out *os.File) int {
	pkgs := make([]string, 0, len(floors))
	for pkg := range floors {
		pkgs = append(pkgs, pkg)
	}
	sort.Strings(pkgs)
	failed := 0
	for _, pkg := range pkgs {
		c, ok := cov[pkg]
		got := percent(c)
		switch {
		case !ok:
			fmt.Fprintf(out, "FAIL %s: not in the profile (floor %.0f%%)\n", pkg, floors[pkg])
			failed++
		case got+1e-9 < floors[pkg]:
			fmt.Fprintf(out, "FAIL %s: %.1f%% is below the floor of %.0f%%\n", pkg, got, floors[pkg])
			failed++
		}
	}
	return failed
}

func percent(c counts) float64 {
	if c.total == 0 {
		return 100
	}
	return float64(c.covered) / float64(c.total) * 100
}

// readProfile sums the statements per package. A block seen more than once
// counts as covered when any of its entries ran.
func readProfile(name string) (map[string]counts, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	type block struct {
		pkg     string
		stmts   int
		covered bool
	}
	blocks := map[string]*block{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "mode:") || line == "" {
			continue
		}
		// file.go:10.2,12.3 2 1
		sp := strings.Fields(line)
		if len(sp) != 3 {
			return nil, fmt.Errorf("unexpected profile line %q", line)
		}
		stmts, err1 := strconv.Atoi(sp[1])
		count, err2 := strconv.Atoi(sp[2])
		if err1 != nil || err2 != nil {
			return nil, fmt.Errorf("unexpected profile line %q", line)
		}
		file := sp[0][:strings.LastIndex(sp[0], ":")]
		b := blocks[sp[0]]
		if b == nil {
			b = &block{pkg: path.Dir(file), stmts: stmts}
			blocks[sp[0]] = b
		}
		b.covered = b.covered || count > 0
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	cov := map[string]counts{}
	for _, b := range blocks {
		c := cov[b.pkg]
		c.total += b.stmts
		if b.covered {
			c.covered += b.stmts
		}
		cov[b.pkg] = c
	}
	return cov, nil
}

func readFloors(name string) (map[string]float64, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	floors := map[string]float64{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		sp := strings.Fields(line)
		if len(sp) != 2 {
			return nil, fmt.Errorf("unexpected floors line %q", line)
		}
		v, err := strconv.ParseFloat(sp[1], 64)
		if err != nil {
			return nil, fmt.Errorf("unexpected floors line %q: %w", line, err)
		}
		floors[sp[0]] = v
	}
	return floors, sc.Err()
}

func writeFloors(name string, cov map[string]counts, prefix string) error {
	pkgs := make([]string, 0, len(cov))
	for pkg := range cov {
		if strings.HasPrefix(pkg, prefix) {
			pkgs = append(pkgs, pkg)
		}
	}
	sort.Strings(pkgs)
	var sb strings.Builder
	sb.WriteString("# Minimum statement coverage per package, checked by tools/coverfloor against the\n")
	sb.WriteString("# coverage.out of `task test-ci` (the offline suite). Regenerate with\n")
	sb.WriteString("#   go run ./tools/coverfloor -profile coverage.out -update\n")
	sb.WriteString("# Raise a floor when you add tests; lowering one needs a reason in the pull request.\n")
	for _, pkg := range pkgs {
		fmt.Fprintf(&sb, "%s %d\n", pkg, max(0, int(math.Floor(percent(cov[pkg])))-floorMargin))
	}
	return os.WriteFile(name, []byte(sb.String()), 0o644)
}
