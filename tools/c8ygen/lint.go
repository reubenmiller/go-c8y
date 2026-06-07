package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// pathLiteral matches double-quoted strings that look like an API path beginning with
// a leading slash, e.g. "/alarm/alarms/{id}".
var pathLiteral = regexp.MustCompile(`"(/[A-Za-z0-9_./{}-]+)"`)

// LintResult is the outcome of a drift check between the OAS and the SDK source.
type LintResult struct {
	OASPaths     int
	SDKPaths     int
	MissingInSDK []string // present in OAS, no matching path literal in the SDK (unwaived)
	ExtraInSDK   []string // path literal in the SDK with no matching OAS path (unwaived)
	WaivedCount  int      // drift items suppressed by overlay waivers
}

// HasDrift reports whether any undeclared (unwaived) drift was found.
func (r LintResult) HasDrift() bool {
	return len(r.MissingInSDK) > 0 || len(r.ExtraInSDK) > 0
}

// Lint compares OAS paths against the path literals found in the SDK source tree, then
// suppresses any drift declared in the overlay waivers. srcDir is the root to scan
// (e.g. pkg/c8y/api).
func Lint(doc *OAS, srcDir string, waivers driftWaivers) (LintResult, error) {
	res := LintResult{}

	oas := map[string]string{} // normalized -> original
	for _, p := range sortedKeys(doc.Paths) {
		oas[normalizePath(p)] = p
	}
	res.OASPaths = len(oas)

	sdk, err := scanSDKPaths(srcDir)
	if err != nil {
		return res, err
	}
	res.SDKPaths = len(sdk)

	missing := normalizePatterns(waivers.IgnoreMissing)
	extra := normalizePatterns(waivers.IgnoreExtra)

	for norm, orig := range oas {
		if _, ok := sdk[norm]; ok {
			continue
		}
		if matchesAny(norm, missing) {
			res.WaivedCount++
			continue
		}
		res.MissingInSDK = append(res.MissingInSDK, orig)
	}
	for norm, orig := range sdk {
		if _, ok := oas[norm]; ok {
			continue
		}
		if matchesAny(norm, extra) {
			res.WaivedCount++
			continue
		}
		res.ExtraInSDK = append(res.ExtraInSDK, orig)
	}
	sort.Strings(res.MissingInSDK)
	sort.Strings(res.ExtraInSDK)
	return res, nil
}

// normalizePatterns normalizes each waiver pattern's path component the same way SDK/OAS
// paths are normalized ({param} → {}), preserving a trailing "*" wildcard.
func normalizePatterns(patterns []string) []string {
	out := make([]string, len(patterns))
	for i, p := range patterns {
		prefix, wild := strings.CutSuffix(p, "*")
		np := normalizePath(prefix)
		if wild {
			// normalizePath trims trailing slashes; preserve one before a wildcard so
			// "/meta/*" stays a path-segment prefix and does not match "/metabolism".
			if strings.HasSuffix(prefix, "/") {
				np += "/"
			}
			np += "*"
		}
		out[i] = np
	}
	return out
}

// matchesAny reports whether a normalized path matches any waiver pattern. A pattern
// ending in "*" matches by prefix; otherwise the match is exact.
func matchesAny(path string, patterns []string) bool {
	for _, p := range patterns {
		if prefix, ok := strings.CutSuffix(p, "*"); ok {
			if strings.HasPrefix(path, prefix) {
				return true
			}
		} else if path == p {
			return true
		}
	}
	return false
}

// scanSDKPaths walks srcDir and extracts API path literals from non-test, non-generated
// Go source. Returns a map of normalized -> a representative original literal.
func scanSDKPaths(srcDir string) (map[string]string, error) {
	out := map[string]string{}
	err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		if strings.HasSuffix(name, "_test.go") || strings.HasPrefix(name, "zz_generated") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range pathLiteral.FindAllStringSubmatch(string(data), -1) {
			lit := m[1]
			if !looksLikeAPIPath(lit) {
				continue
			}
			out[normalizePath(lit)] = lit
		}
		return nil
	})
	return out, err
}

// looksLikeAPIPath filters out incidental slash-strings (file paths, content types,
// formatting fragments) that are not REST endpoints.
func looksLikeAPIPath(s string) bool {
	if len(s) < 2 || !strings.HasPrefix(s, "/") {
		return false
	}
	// Must have at least two segments (e.g. /alarm/alarms), excluding placeholders.
	segs := strings.Split(strings.Trim(s, "/"), "/")
	real := 0
	for _, seg := range segs {
		if seg == "" {
			continue
		}
		if strings.HasPrefix(seg, "{") {
			continue
		}
		real++
	}
	if real < 2 {
		return false
	}
	// Reject things that are clearly not endpoints.
	for _, bad := range []string{".go", ".json", ".yml", ".yaml", " ", "://", "*"} {
		if strings.Contains(s, bad) {
			return false
		}
	}
	return true
}

// normalizePath collapses path parameters so "/alarm/alarms/{id}" and
// "/alarm/alarms/{alarmId}" compare equal: every "{...}" becomes "{}". Trailing
// slashes are trimmed.
func normalizePath(p string) string {
	var b strings.Builder
	depth := 0
	for _, r := range p {
		switch r {
		case '{':
			depth++
			if depth == 1 {
				b.WriteString("{}")
			}
		case '}':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimRight(b.String(), "/")
}

// PrintLintReport writes a human-readable drift report to stdout.
func PrintLintReport(r LintResult) {
	fmt.Printf("API drift report\n")
	fmt.Printf("  OAS paths:  %d\n", r.OASPaths)
	fmt.Printf("  SDK paths:  %d\n", r.SDKPaths)
	fmt.Printf("  undeclared drift — missing in SDK: %d   extra in SDK: %d   (waived: %d)\n\n",
		len(r.MissingInSDK), len(r.ExtraInSDK), r.WaivedCount)

	if len(r.MissingInSDK) > 0 {
		fmt.Printf("OAS operations with no matching SDK path (potential coverage gaps):\n")
		for _, p := range r.MissingInSDK {
			fmt.Printf("  - %s\n", p)
		}
		fmt.Println()
	}
	if len(r.ExtraInSDK) > 0 {
		fmt.Printf("SDK path literals not found in the OAS (non-OAS features, sub-paths, or typos):\n")
		for _, p := range r.ExtraInSDK {
			fmt.Printf("  + %s\n", p)
		}
		fmt.Println()
	}
	if !r.HasDrift() {
		fmt.Printf("No undeclared drift (%d waived by the overlay).\n", r.WaivedCount)
	}
}
