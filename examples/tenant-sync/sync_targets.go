package main

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/tenants/currenttenant"
)

// Target is a single tenant the manifest is applied to
type Target struct {
	// TenantID of the target. Empty for the implicit current-tenant target
	// when no targets are configured (resolved lazily where needed).
	TenantID string

	// Domain of the tenant (informational; used for session matching)
	Domain string

	// Current marks the current tenant (the one the base credentials belong to)
	Current bool

	// Auth carries the tenant-specific credentials. Nil means the base
	// client credentials are used (only valid for the current tenant).
	Auth *authentication.AuthOptions
}

// Label returns the display name used to prefix result items
func (t Target) Label() string {
	if t.TenantID != "" {
		return t.TenantID
	}
	return "current"
}

// resolveTargets expands the targets spec into the list of tenants the
// manifest is applied to. The current tenant (when included) is always first.
func (s *Syncer) resolveTargets(ctx context.Context, spec *TargetsSpec) ([]Target, error) {
	if !spec.HasRemoteSelection() {
		// Current tenant only: the base credentials apply and no tenant
		// lookups are needed (preserves the single-tenant behaviour)
		return []Target{{Current: true}}, nil
	}

	current := s.Client.Tenants.Current.Get(ctx, currenttenant.GetOptions{})
	if current.Err != nil {
		return nil, fmt.Errorf("failed to get current tenant: %w", current.Err)
	}
	currentID := current.Data.Name()

	var targets []Target
	seen := map[string]bool{}
	add := func(t Target) {
		if t.TenantID == "" || seen[t.TenantID] {
			return
		}
		seen[t.TenantID] = true
		t.Current = t.TenantID == currentID
		targets = append(targets, t)
	}

	if spec.IncludesCurrent() {
		add(Target{TenantID: currentID, Domain: current.Data.DomainName()})
	}

	if spec.AllChildren {
		it := s.Client.Tenants.ListAll(ctx, tenants.ListOptions{
			Parent:            currentID,
			PaginationOptions: pagination.PaginationOptions{PageSize: 100},
		})
		count := 0
		for item, err := range it.Items() {
			if err != nil {
				return nil, fmt.Errorf("failed to list child tenants of %s: %w", currentID, err)
			}
			add(Target{TenantID: item.ID(), Domain: item.Domain()})
			count++
		}
		if count == 0 {
			fmt.Printf("⚠️  targets.allChildren matched no child tenants of %s\n", currentID)
		}
	}

	for _, entry := range spec.Tenants {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !isDomainReference(entry) {
			add(Target{TenantID: entry})
			continue
		}
		it := s.Client.Tenants.ListAll(ctx, tenants.ListOptions{
			Domain:            entry,
			PaginationOptions: pagination.PaginationOptions{PageSize: 2},
		})
		found := false
		for item, err := range it.Items() {
			if err != nil {
				return nil, fmt.Errorf("failed to look up tenant by domain %q: %w", entry, err)
			}
			add(Target{TenantID: item.ID(), Domain: item.Domain()})
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf("no tenant found with domain %q", entry)
		}
	}

	if spec.Selector != nil {
		it := s.Client.Tenants.ListAll(ctx, tenants.ListOptions{
			Company:           spec.Selector.Company,
			PaginationOptions: pagination.PaginationOptions{PageSize: 100},
		})
		count := 0
		for item, err := range it.Items() {
			if err != nil {
				return nil, fmt.Errorf("failed to list tenants for selector: %w", err)
			}
			if !matchesDomainGlob(spec.Selector.Domain, item.Domain()) {
				continue
			}
			add(Target{TenantID: item.ID(), Domain: item.Domain()})
			count++
		}
		if count == 0 {
			fmt.Printf("⚠️  targets.selector matched no tenants\n")
		}
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("targets did not match any tenant")
	}
	return targets, nil
}

// dryRunTargets returns placeholder targets describing the selection without
// resolving it (dry-run mode performs no API calls)
func dryRunTargets(spec *TargetsSpec) []Target {
	if !spec.HasRemoteSelection() {
		return []Target{{Current: true}}
	}

	var targets []Target
	if spec.IncludesCurrent() {
		targets = append(targets, Target{Current: true, TenantID: "current tenant"})
	}
	if spec.AllChildren {
		targets = append(targets, Target{TenantID: "all child tenants"})
	}
	for _, entry := range spec.Tenants {
		if entry = strings.TrimSpace(entry); entry != "" {
			targets = append(targets, Target{TenantID: entry})
		}
	}
	if spec.Selector != nil {
		criteria := make([]string, 0, 2)
		if spec.Selector.Domain != "" {
			criteria = append(criteria, "domain="+spec.Selector.Domain)
		}
		if spec.Selector.Company != "" {
			criteria = append(criteria, "company="+spec.Selector.Company)
		}
		targets = append(targets, Target{TenantID: "tenants matching " + strings.Join(criteria, ",")})
	}
	return targets
}

// isDomainReference reports whether a targets entry refers to a tenant by
// domain (rather than by ID). Tenant IDs cannot contain dots.
func isDomainReference(entry string) bool {
	return strings.Contains(entry, ".")
}

// matchesDomainGlob matches a tenant domain against a glob pattern.
// An empty pattern matches everything.
func matchesDomainGlob(pattern, domain string) bool {
	if pattern == "" {
		return true
	}
	matched, err := path.Match(pattern, domain)
	if err != nil {
		// Invalid pattern: fall back to an exact comparison
		return pattern == domain
	}
	return matched
}
