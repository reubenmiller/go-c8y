package dtm

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// Action is what Apply does, or Plan predicts, for one item.
type Action string

const (
	ActionNone   Action = "none"
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionAssign Action = "assign"
	ActionDelete Action = "delete"
)

// Item kinds in an ApplyReport, besides the Kind* model kinds.
const (
	KindSubAssetAssignment = "SubAssetAssignment"
	KindDeviceAssignment   = "DeviceAssignment"
	KindLinkedSeries       = "LinkedSeries"
)

// ApplyItem is the outcome (Apply) or the prediction (Plan) for one object.
type ApplyItem struct {
	Kind string
	// Key identifies the object, for example "maxFlow[asset]", "Pump",
	// "ext:pump-07", "ext:hall-1 > ext:pump-07" or "ext:pump-07 c8y_Flow.F".
	Key    string
	Action Action
	// Changed lists the fields that differ, for updates.
	Changed []string
	// Status is the result status. Set by Apply only.
	Status op.Status
	// Method and Path describe the write that is (or would be) sent.
	// Empty for ActionNone.
	Method string
	Path   string
	Err    error
}

// ApplyReport lists one item per object, in the order they are applied.
type ApplyReport struct {
	Items []ApplyItem
	// Planned is true for a report from Plan, where nothing was written.
	Planned bool
}

// Changes returns the number of items that write (or would write).
func (r ApplyReport) Changes() int {
	n := 0
	for _, it := range r.Items {
		if it.Action != ActionNone && it.Err == nil {
			n++
		}
	}
	return n
}

// Errors returns the items that failed.
func (r ApplyReport) Errors() []ApplyItem {
	var out []ApplyItem
	for _, it := range r.Items {
		if it.Err != nil {
			out = append(out, it)
		}
	}
	return out
}

// ApplyOptions controls Apply and Plan.
type ApplyOptions struct {
	// ManagedBy is a tag added to every definition Apply writes. Prune uses
	// it to find definitions this model owns.
	ManagedBy string
	// Prune deletes definitions tagged ManagedBy that are not in the model.
	// Assets are never pruned, because deleting them can cascade to
	// devices. Requires ManagedBy.
	Prune bool
	// ContinueOnError keeps going after a failed item. By default Apply
	// stops at the first failure.
	ContinueOnError bool

	WriteOptions
}

// Plan reads the tenant and reports what Apply would change, without
// writing anything. Every item has the action Apply would take, the fields
// that differ, and the request it would send.
//
// The returned result can be executed: plan.Execute(ctx) runs Apply with
// the same model and options (it reads the tenant again, so changes made
// in between are taken into account).
func (s *Service) Plan(ctx context.Context, m *Model, opt ApplyOptions) op.Result[ApplyReport] {
	steps, err := s.steps(m, opt)
	if err != nil {
		return op.Failed[ApplyReport](err, false)
	}
	report := ApplyReport{Planned: true}
	for _, st := range steps {
		items := st.plan(ctx)
		report.Items = append(report.Items, items...)
	}
	if opt.Prune {
		report.Items = append(report.Items, s.prune(ctx, m, opt, false)...)
	}
	res := op.OK(report).WithMeta("changes", report.Changes())
	if errs := report.Errors(); len(errs) > 0 {
		res.Err = itemsError(errs)
		res.Status = op.StatusFailed
	}
	return res.WithExecutor(func(ctx context.Context) op.Result[ApplyReport] {
		return s.Apply(ctx, m, opt)
	})
}

// Apply makes the tenant match the model, in dependency order: property,
// measurement, event and alarm definitions; asset definitions (definitions
// used in another's composition first); assets (parents first); sub-asset
// assignments; device assignments; linked series; and, with Prune, deletes
// of definitions that are no longer in the model.
//
// Each item is idempotent: objects that already match are reported with
// ActionNone and are not written, so applying the same model twice writes
// nothing the second time.
//
// result.Err joins the errors of the failed items; result.Data always
// holds the report of what was done.
func (s *Service) Apply(ctx context.Context, m *Model, opt ApplyOptions) op.Result[ApplyReport] {
	steps, err := s.steps(m, opt)
	if err != nil {
		return op.Failed[ApplyReport](err, false)
	}
	return op.Result[ApplyReport]{}.WithExecutor(func(ctx context.Context) op.Result[ApplyReport] {
		var report ApplyReport
		for _, st := range steps {
			item := st.apply(ctx)
			report.Items = append(report.Items, item)
			if item.Err != nil && !opt.ContinueOnError {
				return failedReport(report)
			}
		}
		if opt.Prune {
			report.Items = append(report.Items, s.prune(ctx, m, opt, true)...)
		}
		if len(report.Errors()) > 0 {
			return failedReport(report)
		}
		return op.OK(report).WithMeta("changes", report.Changes())
	}).WithMeta("operation", "apply").ExecuteOrDefer(ctx)
}

func failedReport(report ApplyReport) op.Result[ApplyReport] {
	res := op.Failed[ApplyReport](itemsError(report.Errors()), false)
	res.Data = report
	return res
}

func itemsError(items []ApplyItem) error {
	errs := make([]error, 0, len(items))
	for _, it := range items {
		errs = append(errs, fmt.Errorf("%s %s: %w", it.Kind, it.Key, it.Err))
	}
	return errors.Join(errs...)
}

// step is one object: plan predicts without writing, apply writes.
type step struct {
	plan  func(context.Context) []ApplyItem
	apply func(context.Context) ApplyItem
}

func (s *Service) steps(m *Model, opt ApplyOptions) ([]step, error) {
	if m == nil {
		return nil, fmt.Errorf("model is nil")
	}
	if opt.Prune && opt.ManagedBy == "" {
		return nil, fmt.Errorf("Prune requires ManagedBy")
	}
	var steps []step
	defs := s.Definitions
	for _, group := range []struct {
		svc  *DefinitionService
		kind string
		list []Definition
	}{
		{defs.Properties, KindPropertyDefinition, m.PropertyDefinitions},
		{defs.Measurements, KindMeasurementDefinition, m.MeasurementDefinitions},
		{defs.Events, KindEventDefinition, m.EventDefinitions},
		{defs.Alarms, KindAlarmDefinition, m.AlarmDefinitions},
	} {
		for _, d := range group.list {
			steps = append(steps, s.definitionStep(group.svc, group.kind, d, opt))
		}
	}
	assetDefs, err := orderAssetDefinitions(m.AssetDefinitions)
	if err != nil {
		return nil, err
	}
	for _, d := range assetDefs {
		steps = append(steps, s.definitionStep(defs.Assets, KindAssetDefinition, d, opt))
	}

	assets, err := orderAssets(m.Assets)
	if err != nil {
		return nil, err
	}
	for _, a := range assets {
		steps = append(steps, s.assetStep(a, opt))
	}
	for _, a := range assets {
		if a.Parent != "" {
			steps = append(steps, s.assignmentStep(a))
		}
	}
	for _, a := range assets {
		for _, dev := range a.Devices {
			steps = append(steps, s.deviceStep(a, dev))
		}
	}
	for _, a := range assets {
		for _, l := range a.LinkedSeries {
			steps = append(steps, s.linkStep(a, l, opt))
		}
	}
	return steps, nil
}

// --- definitions ---

func definitionItemKey(d Definition) string {
	if len(d.Contexts) == 0 {
		return d.Identifier
	}
	ctx := slices.Clone(d.Contexts)
	slices.Sort(ctx)
	return d.Identifier + "[" + strings.Join(ctx, ",") + "]"
}

func (s *Service) definitionStep(svc *DefinitionService, kind string, d Definition, opt ApplyOptions) step {
	if opt.ManagedBy != "" && !slices.Contains(d.Tags, opt.ManagedBy) {
		d.Tags = append(slices.Clone(d.Tags), opt.ManagedBy)
	}
	key := definitionItemKey(d)
	createPath := "/service/dtm/definitions/" + string(svc.kind)
	updatePath := createPath
	if svc.kind == KindProperties {
		updatePath += "/" + d.Identifier
	}
	item := func() ApplyItem { return ApplyItem{Kind: kind, Key: key} }

	return step{
		plan: func(ctx context.Context) []ApplyItem {
			it := item()
			desired, err := toMap(d)
			if err != nil {
				it.Err = err
				return []ApplyItem{it}
			}
			existing, found := svc.Find(ctx, d.Identifier, d.Contexts)
			switch {
			case !found && existing.Err != nil:
				it.Err = existing.Err
			case !found:
				it.Action, it.Method, it.Path = ActionCreate, http.MethodPost, createPath
			default:
				it.Changed = definitionChanges(desired, existing.Data.Bytes())
				switch {
				case len(it.Changed) == 0:
					it.Action = ActionNone
				case existing.Data.IsShared() || !existing.Data.CanEdit():
					it.Err = fmt.Errorf("%w: differs in %s", ErrReadOnlyDefinition, strings.Join(it.Changed, ", "))
				default:
					it.Action, it.Method, it.Path = ActionUpdate, http.MethodPut, updatePath
				}
			}
			return []ApplyItem{it}
		},
		apply: func(ctx context.Context) ApplyItem {
			it := item()
			res := svc.Upsert(ctx, d)
			it.Status, it.Err = res.Status, res.Err
			if changed, ok := res.Meta["changed"].([]string); ok {
				it.Changed = changed
			}
			switch res.Status {
			case op.StatusCreated:
				it.Action, it.Method, it.Path = ActionCreate, http.MethodPost, createPath
			case op.StatusUpdated:
				it.Action, it.Method, it.Path = ActionUpdate, http.MethodPut, updatePath
			default:
				it.Action = ActionNone
			}
			return it
		},
	}
}

// --- assets ---

func (s *Service) assetStep(a ModelAsset, opt ApplyOptions) step {
	key := "ext:" + a.ExternalID
	create := AssetCreateOptions{ExternalID: a.ExternalID, Body: a.Asset, WriteOptions: opt.WriteOptions}
	return step{
		plan: func(ctx context.Context) []ApplyItem {
			it := ApplyItem{Kind: KindAsset, Key: key}
			desired, err := create.body()
			if err != nil {
				it.Err = err
				return []ApplyItem{it}
			}
			existing := s.Assets.GetByExternalID(ctx, a.ExternalID, GetOptions{})
			switch {
			case core.IsNotFound(existing.Err):
				it.Action, it.Method, it.Path = ActionCreate, http.MethodPost, ApiAssets
			case existing.Err != nil:
				it.Err = existing.Err
			case op.DesiredStateMatches(desired, existing.Data.Bytes()):
				it.Action = ActionNone
			default:
				it.Action, it.Method, it.Path = ActionUpdate, http.MethodPost, ApiAssets
				it.Changed = changedKeys(desired, existing.Data.Bytes())
			}
			return []ApplyItem{it}
		},
		apply: func(ctx context.Context) ApplyItem {
			it := ApplyItem{Kind: KindAsset, Key: key}
			res := s.Assets.Upsert(ctx, AssetUpsertOptions{AssetCreateOptions: create})
			it.Status, it.Err = res.Status, res.Err
			switch res.Status {
			case op.StatusCreated:
				it.Action, it.Method, it.Path = ActionCreate, http.MethodPost, ApiAssets
			case op.StatusUpdated:
				it.Action, it.Method, it.Path = ActionUpdate, http.MethodPost, ApiAssets
			default:
				it.Action = ActionNone
			}
			return it
		},
	}
}

// changedKeys lists the top-level keys of desired that differ.
func changedKeys(desired map[string]any, existing []byte) []string {
	stored := jsonObject(existing)
	var out []string
	for k, v := range desired {
		if !op.DesiredStateMatches(map[string]any{k: v}, mustJSON(map[string]any{k: stored[k]})) {
			out = append(out, k)
		}
	}
	slices.Sort(out)
	return out
}

// assignment checks whether child is a sub-asset of parent. It returns the
// action and the ids (or the references, when an asset does not exist yet).
func (s *Service) assignment(ctx context.Context, a ModelAsset) (Action, string, string, error) {
	parentID, err := s.Assets.ResolveID(ctx, a.Parent, nil)
	parentMissing := core.IsNotFound(err)
	if err != nil && !parentMissing {
		return "", "", "", err
	}
	child := s.Assets.GetByExternalID(ctx, a.ExternalID, GetOptions{WithParents: true})
	if core.IsNotFound(child.Err) || parentMissing {
		return ActionAssign, orDefault(parentID, string(a.Parent)), orDefault(child.Data.ID(), "ext:"+a.ExternalID), nil
	}
	if child.Err != nil {
		return "", "", "", child.Err
	}
	for p := range child.Data.Parents() {
		if p.ID() == parentID {
			return ActionNone, parentID, child.Data.ID(), nil
		}
	}
	return ActionAssign, parentID, child.Data.ID(), nil
}

func (s *Service) assignmentStep(a ModelAsset) step {
	key := string(a.Parent) + " > ext:" + a.ExternalID
	item := func(action Action, parent, child string, err error) ApplyItem {
		it := ApplyItem{Kind: KindSubAssetAssignment, Key: key, Action: action, Err: err}
		if action == ActionAssign {
			it.Method, it.Path = http.MethodPost, "/service/dtm/assets/"+parent+"/subAssets/"+child
		}
		return it
	}
	return step{
		plan: func(ctx context.Context) []ApplyItem {
			return []ApplyItem{item(s.assignment(ctx, a))}
		},
		apply: func(ctx context.Context) ApplyItem {
			action, parent, child, err := s.assignment(ctx, a)
			it := item(action, parent, child, err)
			if err != nil || action == ActionNone {
				it.Status = op.StatusSkipped
				return it
			}
			res := s.Assets.SubAssets.Assign(ctx, AssetByID(parent), AssetByID(child))
			it.Status, it.Err = res.Status, res.Err
			return it
		},
	}
}

// --- devices ---

// deviceAssignment checks whether the device is assigned to the asset.
func (s *Service) deviceAssignment(ctx context.Context, a ModelAsset, device string) (Action, string, string, error) {
	deviceID, err := s.Assets.resolveDevice(ctx, device, nil)
	if err != nil {
		return "", "", "", err
	}
	asset := s.Assets.GetByExternalID(ctx, a.ExternalID, GetOptions{})
	if core.IsNotFound(asset.Err) {
		return ActionAssign, "ext:" + a.ExternalID, deviceID, nil
	}
	if asset.Err != nil {
		return "", "", "", asset.Err
	}
	for d, err := range s.Assets.Devices.ListAll(ctx, AssetByID(asset.Data.ID()), DeviceListOptions{
		PaginationOptions: pagination.PaginationOptions{PageSize: 2000},
	}).Items() {
		if err != nil {
			return "", "", "", err
		}
		if d.ID() == deviceID {
			return ActionNone, asset.Data.ID(), deviceID, nil
		}
	}
	return ActionAssign, asset.Data.ID(), deviceID, nil
}

func (s *Service) deviceStep(a ModelAsset, device string) step {
	key := "ext:" + a.ExternalID + " < " + device
	item := func(action Action, assetID, deviceID string, err error) ApplyItem {
		it := ApplyItem{Kind: KindDeviceAssignment, Key: key, Action: action, Err: err}
		if action == ActionAssign {
			it.Method, it.Path = http.MethodPost, "/service/dtm/assets/"+assetID+"/devices/"+deviceID
		}
		return it
	}
	return step{
		plan: func(ctx context.Context) []ApplyItem {
			return []ApplyItem{item(s.deviceAssignment(ctx, a, device))}
		},
		apply: func(ctx context.Context) ApplyItem {
			action, assetID, deviceID, err := s.deviceAssignment(ctx, a, device)
			it := item(action, assetID, deviceID, err)
			if err != nil || action == ActionNone {
				it.Status = op.StatusSkipped
				return it
			}
			res := s.Assets.Devices.Assign(ctx, AssetByID(assetID), deviceID)
			it.Status, it.Err = res.Status, res.Err
			return it
		},
	}
}

// --- linked series ---

func (s *Service) linkStep(a ModelAsset, l ModelLink, opt ApplyOptions) step {
	key := "ext:" + a.ExternalID + " " + l.Fragment + "." + l.Series
	asset := AssetByExternalID(a.ExternalID)
	linkOpt := l.options()
	linkOpt.WriteOptions = opt.WriteOptions
	path := func(id string) string { return "/service/dtm/assets/" + id + "/linkedSeries" }

	return step{
		plan: func(ctx context.Context) []ApplyItem {
			it := ApplyItem{Kind: KindLinkedSeries, Key: key}
			if l.Fragment == "" || l.Series == "" || l.Device == "" {
				it.Err = fmt.Errorf("fragment, series and device are required")
				return []ApplyItem{it}
			}
			deviceID, err := s.Assets.resolveDevice(ctx, l.Device, nil)
			if err != nil {
				it.Err = err
				return []ApplyItem{it}
			}
			existing := s.Assets.GetByExternalID(ctx, a.ExternalID, GetOptions{})
			if core.IsNotFound(existing.Err) {
				it.Action, it.Method, it.Path = ActionCreate, http.MethodPost, path(string(asset))
				return []ApplyItem{it}
			}
			if existing.Err != nil {
				it.Err = existing.Err
				return []ApplyItem{it}
			}
			id := existing.Data.ID()
			current, found, res := s.Assets.LinkedSeries.find(ctx, id, l.Fragment, l.Series)
			switch {
			case res.Err != nil:
				it.Err = res.Err
			case !found:
				it.Action, it.Method, it.Path = ActionCreate, http.MethodPost, path(id)
			case linkMatches(desiredLink(linkOpt, deviceID), current):
				it.Action = ActionNone
			default:
				it.Action, it.Method, it.Path = ActionUpdate, http.MethodPost, path(id)
			}
			return []ApplyItem{it}
		},
		apply: func(ctx context.Context) ApplyItem {
			it := ApplyItem{Kind: KindLinkedSeries, Key: key}
			res := s.Assets.LinkedSeries.Link(ctx, asset, linkOpt)
			it.Status, it.Err = res.Status, res.Err
			switch res.Status {
			case op.StatusCreated:
				it.Action = ActionCreate
			case op.StatusUpdated:
				it.Action = ActionUpdate
			default:
				it.Action = ActionNone
			}
			if it.Action != ActionNone {
				id, _ := res.Meta["assetId"].(string)
				it.Method, it.Path = http.MethodPost, path(id)
			}
			return it
		},
	}
}

// desiredLink builds the linked series Link would send.
func desiredLink(opt LinkOptions, deviceID string) LinkedSeries {
	return LinkedSeries{
		Fragment: opt.Fragment,
		Series:   opt.Series,
		Label:    opt.Label,
		Source: &SeriesSource{
			ID:       deviceID,
			Fragment: orDefault(opt.SourceFragment, opt.Fragment),
			Series:   orDefault(opt.SourceSeries, opt.Series),
			Type:     opt.SourceType,
		},
	}
}

// --- prune ---

// prune deletes (write) or lists (plan) definitions tagged ManagedBy that
// are not in the model. Asset definitions go first, a definition before
// the ones its composition uses; property definitions go last.
func (s *Service) prune(ctx context.Context, m *Model, opt ApplyOptions, write bool) []ApplyItem {
	var items []ApplyItem
	defs := s.Definitions
	for _, group := range []struct {
		svc  *DefinitionService
		kind string
		want []Definition
	}{
		{defs.Assets, KindAssetDefinition, m.AssetDefinitions},
		{defs.Alarms, KindAlarmDefinition, m.AlarmDefinitions},
		{defs.Events, KindEventDefinition, m.EventDefinitions},
		{defs.Measurements, KindMeasurementDefinition, m.MeasurementDefinitions},
		{defs.Properties, KindPropertyDefinition, m.PropertyDefinitions},
	} {
		keep := map[string]bool{}
		for _, d := range group.want {
			keep[definitionItemKey(d)] = true
		}
		var stale []Definition
		var staleRaw []jsonmodels.DigitalTwinDefinition
		for def, err := range group.svc.ListAll(ctx, ListDefinitionsOptions{
			Tags:              []string{opt.ManagedBy},
			PaginationOptions: pagination.PaginationOptions{PageSize: 2000},
		}).Items() {
			if err != nil {
				items = append(items, ApplyItem{Kind: group.kind, Key: "(list)", Err: err})
				break
			}
			d := Definition{Identifier: def.Identifier(), Contexts: def.Contexts()}
			if !keep[definitionItemKey(d)] {
				stale = append(stale, d)
				staleRaw = append(staleRaw, def)
			}
		}
		if group.kind == KindAssetDefinition {
			stale = orderForDelete(stale, staleRaw)
		}
		for _, d := range stale {
			it := ApplyItem{
				Kind: group.kind, Key: definitionItemKey(d), Action: ActionDelete,
				Method: http.MethodDelete, Path: "/service/dtm/definitions/" + string(group.svc.kind) + "/" + d.Identifier,
			}
			if write {
				res := group.svc.Delete(ctx, d.Identifier, DeleteDefinitionOptions{Contexts: d.Contexts})
				it.Status, it.Err = res.Status, res.Err
			}
			items = append(items, it)
		}
	}
	return items
}

// --- ordering ---

// orderAssetDefinitions puts each asset definition after the asset
// definitions its composition allows as sub-assets, so they exist first.
func orderAssetDefinitions(defs []Definition) ([]Definition, error) {
	byID := map[string]Definition{}
	for _, d := range defs {
		if _, dup := byID[d.Identifier]; dup {
			return nil, fmt.Errorf("asset definition %q is defined twice", d.Identifier)
		}
		byID[d.Identifier] = d
	}
	deps := func(d Definition) []string {
		if d.Composition == nil {
			return nil
		}
		var out []string
		for _, sub := range d.Composition.AllowedSubAssets {
			if _, inModel := byID[sub.Identifier]; inModel {
				out = append(out, sub.Identifier)
			}
		}
		return out
	}
	ids := make([]string, 0, len(defs))
	for _, d := range defs {
		ids = append(ids, d.Identifier)
	}
	order, err := topoSort(ids, func(id string) []string { return deps(byID[id]) })
	if err != nil {
		return nil, fmt.Errorf("asset definitions: %w", err)
	}
	out := make([]Definition, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out, nil
}

// orderAssets puts each asset after its parent when the parent is in the
// model.
func orderAssets(assets []ModelAsset) ([]ModelAsset, error) {
	byID := map[string]ModelAsset{}
	ids := make([]string, 0, len(assets))
	for _, a := range assets {
		if _, dup := byID[a.ExternalID]; dup {
			return nil, fmt.Errorf("asset %q is defined twice", a.ExternalID)
		}
		byID[a.ExternalID] = a
		ids = append(ids, a.ExternalID)
	}
	order, err := topoSort(ids, func(id string) []string {
		if parent, ok := byID[id].Parent.externalID(); ok {
			if _, inModel := byID[parent]; inModel {
				return []string{parent}
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("assets: %w", err)
	}
	out := make([]ModelAsset, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out, nil
}

// orderForDelete orders asset definitions so a definition is deleted
// before the definitions its composition uses.
func orderForDelete(defs []Definition, raw []jsonmodels.DigitalTwinDefinition) []Definition {
	byID := map[string]int{}
	ids := make([]string, len(defs))
	for i, d := range defs {
		byID[d.Identifier] = i
		ids[i] = d.Identifier
	}
	// Reverse of the create order: "uses" edges point the other way.
	usedBy := map[string][]string{}
	for i, d := range defs {
		for _, sub := range raw[i].AllowedSubAssets() {
			if _, ok := byID[sub.Identifier]; ok {
				usedBy[sub.Identifier] = append(usedBy[sub.Identifier], d.Identifier)
			}
		}
	}
	order, err := topoSort(ids, func(id string) []string { return usedBy[id] })
	if err != nil {
		return defs
	}
	out := make([]Definition, 0, len(order))
	for _, id := range order {
		out = append(out, defs[byID[id]])
	}
	return out
}

// topoSort returns ids with every id after the ids deps returns for it,
// keeping the input order otherwise. A cycle is an error.
func topoSort(ids []string, deps func(string) []string) ([]string, error) {
	const (
		unvisited = iota
		visiting
		done
	)
	state := map[string]int{}
	var out []string
	var visit func(id string, path []string) error
	visit = func(id string, path []string) error {
		switch state[id] {
		case done:
			return nil
		case visiting:
			return fmt.Errorf("cycle: %s", strings.Join(append(path, id), " -> "))
		}
		state[id] = visiting
		for _, d := range deps(id) {
			if err := visit(d, append(path, id)); err != nil {
				return err
			}
		}
		state[id] = done
		out = append(out, id)
		return nil
	}
	for _, id := range ids {
		if err := visit(id, nil); err != nil {
			return nil, err
		}
	}
	return out, nil
}
