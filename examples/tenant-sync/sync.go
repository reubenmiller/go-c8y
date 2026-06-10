package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
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
	SectionSoftware       = "software"
	SectionFirmware       = "firmware"
	SectionConfiguration  = "configuration"
	SectionDeviceProfiles = "deviceProfiles"
)

// SectionOrder is the order sections are applied in. Repositories are synced
// before device profiles so profiles can reference the synced items.
var SectionOrder = []string{
	SectionTenantOptions,
	SectionFeatures,
	SectionApplications,
	SectionSoftware,
	SectionFirmware,
	SectionConfiguration,
	SectionDeviceProfiles,
}

// ChangeResult records what happened to a single item
type ChangeResult struct {
	Section string
	Item    string
	Action  Action
	Detail  string
	Err     error
}

// Syncer applies a manifest to a tenant
type Syncer struct {
	Client      *api.Client
	Resolver    *SourceResolver
	DryRun      bool
	Force       bool
	Concurrency int

	Results []ChangeResult
}

func (s *Syncer) record(section, item string, action Action, detail string, err error) {
	if err != nil {
		action = ActionFailed
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

// Apply runs the pre hooks, all requested sections of the manifest in order,
// and finally the post hooks. Hooks always run when defined, regardless of
// any --only section filter; a failing pre hook aborts the run.
func (s *Syncer) Apply(ctx context.Context, manifest *Manifest, only []string) error {
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

		var err error
		switch section {
		case SectionTenantOptions:
			err = s.SyncTenantOptions(ctx, manifest.TenantOptions)
		case SectionFeatures:
			err = s.SyncFeatures(ctx, manifest.Features)
		case SectionApplications:
			err = s.SyncApplications(ctx, manifest.Applications)
		case SectionSoftware:
			err = s.SyncSoftware(ctx, manifest.Software)
		case SectionFirmware:
			err = s.SyncFirmware(ctx, manifest.Firmware)
		case SectionConfiguration:
			err = s.SyncConfiguration(ctx, manifest.Configuration)
		case SectionDeviceProfiles:
			err = s.SyncDeviceProfiles(ctx, manifest.DeviceProfiles)
		}
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
