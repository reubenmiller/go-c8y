package users

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/refresolve"
)

// ResolveID resolves a user name, which may contain "*" wildcards, to a user id.
//
// A user id is the user name, so a name without wildcards is returned as-is
// without a lookup (meta: {"source": "direct-id"}). A pattern such as "adm*" is
// matched (case-insensitive) against the user names of the current tenant
// (meta: {"source": "name", "name": "<userName>"}). It returns a not found
// error when nothing matches and an error when more than one user matches.
// meta may be nil.
func (s *Service) ResolveID(ctx context.Context, identifier string, meta map[string]any) (string, error) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", fmt.Errorf("empty user identifier")
	}
	if !refresolve.HasWildcard(identifier) {
		if meta != nil {
			meta["source"] = "direct-id"
		}
		return identifier, nil
	}

	opts := ListOptions{PaginationOptions: pagination.PaginationOptions{MaxItems: 4000}}
	// The server filters by user name prefix, narrow the list with the part before the first wildcard.
	opts.Username, _, _ = strings.Cut(identifier, "*")

	ctx = ctxhelpers.ResolutionContext(ctx)
	it := s.ListAll(ctx, opts)
	var candidates []refresolve.Candidate
	for item := range it.Items() {
		candidates = append(candidates, refresolve.Candidate{ID: item.ID(), Names: []string{item.UserName()}})
	}
	if err := it.Err(); err != nil {
		return "", err
	}
	found, err := refresolve.Select("user", identifier, candidates)
	if err != nil {
		return "", err
	}
	if meta != nil {
		meta["source"] = "name"
		meta["name"] = found.Names[0]
	}
	return found.ID, nil
}
