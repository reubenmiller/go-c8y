// Package refresolve holds the matching rules shared by the ResolveID
// implementations of the services whose resources are found by name
// (users, user roles, tenants, binaries).
package refresolve

import (
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/matcher"
)

// Candidate is a resource that identifier may refer to.
type Candidate struct {
	// ID is the value returned by ResolveID.
	ID string
	// Names are the properties the identifier is matched against (e.g.
	// user name, tenant id, domain and company).
	Names []string
}

// HasWildcard reports whether s contains a "*" wildcard.
func HasWildcard(s string) bool {
	return strings.Contains(s, "*")
}

// IsDigits reports whether s is a non-empty string of decimal digits.
func IsDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Select picks the single candidate identifier refers to. An exact match on any
// name wins over wildcard matches. It returns a 404 core.Error when nothing
// matches and an error when more than one candidate matches.
func Select(kind, identifier string, candidates []Candidate) (Candidate, error) {
	var exact, wild []Candidate
	for _, c := range candidates {
		isExact, isWild := false, false
		for _, n := range c.Names {
			if n == "" {
				continue
			}
			if n == identifier {
				isExact = true
				break
			}
			if HasWildcard(identifier) {
				if ok, _ := matcher.MatchWithWildcards(n, identifier); ok {
					isWild = true
				}
			}
		}
		switch {
		case isExact:
			exact = append(exact, c)
		case isWild:
			wild = append(wild, c)
		}
	}
	matches := exact
	if len(matches) == 0 {
		matches = wild
	}
	switch len(matches) {
	case 0:
		return Candidate{}, core.NewNameNotFoundError(kind, identifier, "%s not found: %s", kind, identifier)
	case 1:
		return matches[0], nil
	}
	ids := make([]string, 0, len(matches))
	for i, m := range matches {
		if i == 5 {
			ids = append(ids, "...")
			break
		}
		ids = append(ids, m.ID)
	}
	return Candidate{}, fmt.Errorf("ambiguous %s %q: %d matches (%s)", kind, identifier, len(matches), strings.Join(ids, ", "))
}
