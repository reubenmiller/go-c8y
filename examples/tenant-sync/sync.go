package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// Action describes the outcome of syncing a single item
type Action string

const (
	ActionCreated   Action = "created"
	ActionUpdated   Action = "updated"
	ActionUnchanged Action = "unchanged"
	ActionSkipped   Action = "skipped"  // optional source with nothing to do
	ActionExecuted  Action = "executed" // hook commands
	ActionPlanned   Action = "planned"  // dry-run
	ActionFailed    Action = "failed"
)

// SyncToolFragment marks managed objects created by this tool so they can be
// queried, e.g. c8y inventory list --query "has(c8y_TenantSync)"
const SyncToolFragment = "c8y_TenantSync"

// Section names, used by --only filtering and in the apply order
const (
	SectionTenantOptions  = "tenantOptions"
	SectionFeatures       = "features"
	SectionApplications   = "applications"
	SectionUserGroups     = "userGroups"
	SectionUsers          = "users"
	SectionSoftware       = "software"
	SectionFirmware       = "firmware"
	SectionConfiguration  = "configuration"
	SectionSmartRest      = "smartrestTemplates"
	SectionDeviceProfiles = "deviceProfiles"
	SectionCommands       = "commands"
)

// SectionOrder is the order sections are applied in. User groups are synced
// before users so users can be assigned to groups created in the same run;
// repositories are synced before device profiles so profiles can reference
// the synced items; custom commands run last so they can build on everything
// the manifest declares.
var SectionOrder = []string{
	SectionTenantOptions,
	SectionFeatures,
	SectionApplications,
	SectionUserGroups,
	SectionUsers,
	SectionSoftware,
	SectionFirmware,
	SectionConfiguration,
	SectionSmartRest,
	SectionDeviceProfiles,
	SectionCommands,
}

// ChangeResult records what happened to a single item
type ChangeResult struct {
	Section string
	Item    string
	Action  Action
	Detail  string
	Err     error
}

// Syncer applies a manifest to one or more tenants
type Syncer struct {
	Client      *api.Client
	Resolver    *SourceResolver
	DryRun      bool
	Force       bool
	Concurrency int

	// Targets overrides the manifest targets section (set from CLI flags)
	Targets *TargetsSpec

	Results []ChangeResult

	// activeTarget is the tenant currently being applied
	activeTarget *Target
	// multiTarget is set when the run covers tenants other than the current
	// one, enabling per-tenant labels in the output
	multiTarget bool
	// appSelfLinks caches application self links by name so create/upload
	// only happens once when applying to multiple tenants
	appSelfLinks map[string]string
	// featureOverrides caches the per-tenant feature toggle overrides by
	// feature key (tenant ID -> active)
	featureOverrides map[string]map[string]bool

	// mu guards Results and the printed output: command groups record
	// results from concurrent goroutines
	mu sync.Mutex
}

func (s *Syncer) record(section, item string, action Action, detail string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recordLocked(section, item, action, detail, err)
}

func (s *Syncer) recordLocked(section, item string, action Action, detail string, err error) {
	if err != nil {
		action = ActionFailed
	}
	if s.multiTarget && s.activeTarget != nil {
		item = s.activeTarget.Label() + ": " + item
	}
	s.Results = append(s.Results, ChangeResult{
		Section: section,
		Item:    item,
		Action:  action,
		Detail:  detail,
		Err:     err,
	})

	symbol := map[Action]string{
		ActionCreated:   "✚",
		ActionUpdated:   "↻",
		ActionUnchanged: "✓",
		ActionSkipped:   "⏭",
		ActionExecuted:  "▸",
		ActionPlanned:   "→",
		ActionFailed:    "✗",
	}[action]

	line := fmt.Sprintf("  %s [%s] %s (%s)", symbol, section, item, action)
	if detail != "" {
		line += ": " + detail
	}
	if err != nil {
		line += ": " + err.Error()
	}
	fmt.Println(line)
}

// syncMeta returns the bookkeeping fragment attached to created/updated
// items. It is passed as an upsert *annotation*, so it is written whenever
// something real changes but never participates in change detection — the
// syncedAt timestamp therefore records when the desired state last changed,
// without breaking idempotency.
func syncMeta() map[string]any {
	return map[string]any{
		"tool":     "tenant-sync",
		"syncedAt": time.Now().Format(time.RFC3339),
	}
}

// resolveSource resolves a source and reports how to proceed: ok means the
// files should be processed; otherwise the outcome (skipped for optional
// sources with nothing to do, failed for real errors) has been recorded.
func (s *Syncer) resolveSource(section, item string, source Source) (files []ResolvedFile, ok bool) {
	files, err := s.Resolver.Resolve(source)
	if err != nil {
		if source.Optional && errors.Is(err, ErrNothingToDo) {
			s.record(section, item, ActionSkipped, err.Error(), nil)
			return nil, false
		}
		s.record(section, item, ActionFailed, "resolve source", err)
		return nil, false
	}
	return files, true
}

// jsonEqual compares a desired Go value with an existing JSON fragment by
// normalising both through JSON. Used to decide whether an update is needed.
func jsonEqual(desired any, existingRaw string) bool {
	desiredJSON, err := json.Marshal(desired)
	if err != nil {
		return false
	}
	var a, b any
	if err := json.Unmarshal(desiredJSON, &a); err != nil {
		return false
	}
	if err := json.Unmarshal([]byte(existingRaw), &b); err != nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

// actionFromResult maps an op.Result status + meta to a sync action
func actionFromResult(status op.Status, meta map[string]any) Action {
	switch status {
	case op.StatusCreated:
		return ActionCreated
	case op.StatusUpdated:
		return ActionUpdated
	case op.StatusSkipped:
		return ActionUnchanged
	}
	if meta != nil {
		if found, ok := meta["found"].(bool); ok && !found {
			return ActionCreated
		}
	}
	return ActionUnchanged
}

// Apply resolves the target tenants and applies the manifest to each of them
// in turn. The applications and features sections always run with the base
// credentials (their APIs address other tenants by ID from the parent); all
// other sections run with the target tenant's own credentials.
func (s *Syncer) Apply(ctx context.Context, manifest *Manifest, only []string) error {
	spec := s.Targets
	if spec == nil {
		spec = manifest.Targets
	}

	var targets []Target
	if s.DryRun {
		// No API calls in dry-run mode: describe the selection instead of
		// resolving it
		targets = dryRunTargets(spec)
	} else {
		var err error
		targets, err = s.resolveTargets(ctx, spec)
		if err != nil {
			return err
		}
		if err := s.resolveTargetCredentials(ctx, targets, spec); err != nil {
			return err
		}
	}

	s.multiTarget = len(targets) > 1 || (len(targets) == 1 && !targets[0].Current)

	for i := range targets {
		s.activeTarget = &targets[i]
		targetCtx := ctx
		if targets[i].Auth != nil {
			targetCtx = api.WithAuth(ctx, *targets[i].Auth)
		}
		if s.multiTarget {
			fmt.Printf("\n🏢 Tenant: %s\n", targets[i].Label())
		}
		if err := s.applyTarget(targetCtx, ctx, manifest, only); err != nil {
			return err
		}
	}
	return nil
}

// applyTarget runs the pre hooks, all requested sections of the manifest in
// order, and finally the post hooks for a single target tenant. Hooks always
// run when defined, regardless of any --only section filter; a failing pre
// hook aborts the run.
//
// ctx carries the target tenant's credentials; baseCtx the base credentials
// of the current tenant (used by the sections that address the target tenant
// by ID from the parent).
func (s *Syncer) applyTarget(ctx, baseCtx context.Context, manifest *Manifest, only []string) error {
	if err := s.runHooks(ctx, "pre", manifest.Hooks.Pre); err != nil {
		return err
	}
	// Post hooks run even when a section fails, so cleanup-style hooks
	// always get a chance to execute
	defer s.runHooks(ctx, "post", manifest.Hooks.Post)
	enabled := func(section string) bool {
		if len(only) == 0 {
			return true
		}
		for _, name := range only {
			if strings.EqualFold(strings.TrimSpace(name), section) {
				return true
			}
		}
		return false
	}

	for _, section := range SectionOrder {
		if !enabled(section) {
			continue
		}

		// Section hooks run around the section (and only when the section
		// itself runs). Like the global hooks, a failing pre hook aborts and
		// post hooks always run.
		sectionHooks := manifest.Hooks.Sections[section]
		if err := s.runHooks(ctx, "sections."+section+".pre", sectionHooks.Pre); err != nil {
			return err
		}

		var err error
		switch section {
		case SectionTenantOptions:
			err = s.SyncTenantOptions(ctx, manifest.TenantOptions)
		case SectionFeatures:
			err = s.SyncFeatures(baseCtx, manifest.Features)
		case SectionApplications:
			err = s.SyncApplications(baseCtx, manifest.Applications)
		case SectionUserGroups:
			err = s.SyncUserGroups(ctx, manifest.UserGroups)
		case SectionUsers:
			err = s.SyncUsers(ctx, manifest.Users)
		case SectionSoftware:
			err = s.SyncSoftware(ctx, manifest.Software)
		case SectionFirmware:
			err = s.SyncFirmware(ctx, manifest.Firmware)
		case SectionConfiguration:
			err = s.SyncConfiguration(ctx, manifest.Configuration)
		case SectionSmartRest:
			err = s.SyncSmartRestTemplates(ctx, manifest.SmartRestTemplates)
		case SectionDeviceProfiles:
			err = s.SyncDeviceProfiles(ctx, manifest.DeviceProfiles)
		case SectionCommands:
			err = s.SyncCommands(ctx, manifest.Commands)
		}

		s.runHooks(ctx, "sections."+section+".post", sectionHooks.Post)

		if err != nil {
			return fmt.Errorf("%s: %w", section, err)
		}
	}
	return nil
}

// Summary aggregates results per action
func (s *Syncer) Summary() (counts map[Action]int, failures []ChangeResult) {
	counts = make(map[Action]int)
	for _, result := range s.Results {
		counts[result.Action]++
		if result.Action == ActionFailed {
			failures = append(failures, result)
		}
	}
	return counts, failures
}

// PrintSummary prints the aggregated results
func (s *Syncer) PrintSummary(elapsed time.Duration) {
	counts, failures := s.Summary()

	fmt.Println("\n" + strings.Repeat("─", 50))

	keys := make([]string, 0, len(counts))
	for action := range counts {
		keys = append(keys, string(action))
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	total := 0
	for _, key := range keys {
		count := counts[Action(key)]
		total += count
		parts = append(parts, fmt.Sprintf("%d %s", count, key))
	}
	fmt.Printf("Total: %d item(s) (%s)\n", total, strings.Join(parts, ", "))
	fmt.Printf("⏱️  Total time: %.1fs\n", elapsed.Seconds())

	if len(failures) > 0 {
		fmt.Printf("\n❌ Failures:\n")
		for _, failure := range failures {
			fmt.Printf("  • [%s] %s: %v\n", failure.Section, failure.Item, failure.Err)
		}
	}
}
