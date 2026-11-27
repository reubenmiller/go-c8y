package binaries

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/refresolve"
)

// ResolveID resolves a binary id or name to a binary id. "*" wildcards are
// supported in names.
//
// A numeric identifier is returned as-is without a lookup (meta: {"source":
// "direct-id"}). Anything else is matched (case-insensitive) against the names
// of the binaries (meta: {"source": "name", "name": "<name>"}). It returns a
// not found error when nothing matches and an error when more than one binary
// matches. meta may be nil.
func (s *Service) ResolveID(ctx context.Context, identifier string, meta map[string]any) (_ string, resolveErr error) {
	defer func() { resolveErr = core.WrapLookup("binary", identifier, resolveErr) }()
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return "", fmt.Errorf("empty binary identifier")
	}
	if refresolve.IsDigits(identifier) {
		if meta != nil {
			meta["source"] = "direct-id"
		}
		return identifier, nil
	}

	ctx = ctxhelpers.ResolutionContext(ctx)
	it := s.ListAll(ctx, ListOptions{PaginationOptions: pagination.PaginationOptions{MaxItems: 4000}})
	var candidates []refresolve.Candidate
	for item := range it.Items() {
		candidates = append(candidates, refresolve.Candidate{ID: item.ID(), Names: []string{item.Name()}})
	}
	if err := it.Err(); err != nil {
		return "", err
	}
	found, err := refresolve.Select("binary", identifier, candidates)
	if err != nil {
		return "", err
	}
	if meta != nil {
		meta["source"] = "name"
		meta["name"] = found.Names[0]
	}
	return found.ID, nil
}
