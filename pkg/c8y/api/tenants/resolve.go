package tenants

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/refresolve"
)

var tenantIDPattern = regexp.MustCompile(`^(t\d+|management)$`)

// ResolveID resolves a tenant id, domain or company name to a tenant id. "*"
// wildcards are supported.
//
// An identifier that looks like a tenant id (e.g. "t12345" or "management") is
// returned as-is without a lookup (meta: {"source": "direct-id"}). Anything
// else is matched (case-insensitive) against the id, domain and company of the
// tenants returned by List, which needs permission to list the tenants
// (meta: {"source": "name", "domain": ..., "company": ...}). It returns a not
// found error when nothing matches and an error when more than one tenant
// matches. meta may be nil.
func (s *Service) ResolveID(ctx context.Context, identifier string, meta map[string]any) (_ string, resolveErr error) {
	defer func() { resolveErr = core.WrapLookup("tenant", identifier, resolveErr) }()
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", fmt.Errorf("empty tenant identifier")
	}
	if tenantIDPattern.MatchString(identifier) {
		if meta != nil {
			meta["source"] = "direct-id"
		}
		return identifier, nil
	}

	ctx = ctxhelpers.ResolutionContext(ctx)
	it := s.ListAll(ctx, ListOptions{PaginationOptions: pagination.PaginationOptions{MaxItems: 4000}})
	var candidates []refresolve.Candidate
	byID := map[string][2]string{}
	for item := range it.Items() {
		candidates = append(candidates, refresolve.Candidate{ID: item.ID(), Names: []string{item.ID(), item.Domain(), item.Company()}})
		byID[item.ID()] = [2]string{item.Domain(), item.Company()}
	}
	if err := it.Err(); err != nil {
		return "", err
	}
	found, err := refresolve.Select("tenant", identifier, candidates)
	if err != nil {
		return "", err
	}
	if meta != nil {
		meta["source"] = "name"
		meta["domain"] = byID[found.ID][0]
		meta["company"] = byID[found.ID][1]
	}
	return found.ID, nil
}
