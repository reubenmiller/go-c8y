package dtm_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/testclient"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/dtm"
)

// statefulDTM is an in-memory DTM with the server behaviours seen on a
// tenant: tags come back in another order, composition entries get
// minOccurs "0", the schema gets $schema, repeat assignments are no-ops, and
// a referenced property definition cannot be deleted.
type statefulDTM struct {
	t      *testing.T
	server *httptest.Server

	mu      sync.Mutex
	nextID  int
	defs    map[string]map[string]map[string]any // kind -> key -> definition
	assets  map[string]*fakeAsset                // id -> asset
	devices map[string]string                    // name -> id
	writes  []string
}

type fakeAsset struct {
	body    map[string]any
	parents []string
	devices []string
	series  []map[string]any
}

func newStatefulDTM(t *testing.T) *statefulDTM {
	f := &statefulDTM{
		t: t, nextID: 100,
		defs:    map[string]map[string]map[string]any{},
		assets:  map[string]*fakeAsset{},
		devices: map[string]string{"pump-ctrl": "900"},
	}
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	return f
}

func (f *statefulDTM) client() *dtm.Service {
	return testclient.Must(api.ClientOptions{
		BaseURL:          f.server.URL,
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant: "t100", Username: "user", Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	}).DigitalTwin
}

func (f *statefulDTM) writeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.writes)
}

func defKey(d map[string]any) string {
	id, _ := d["identifier"].(string)
	var ctx []string
	if raw, ok := d["contexts"].([]any); ok {
		for _, c := range raw {
			ctx = append(ctx, c.(string))
		}
	}
	slices.Sort(ctx)
	return id + "|" + strings.Join(ctx, ",")
}

// serverShape applies the changes the real server makes to a stored
// definition.
func serverShape(d map[string]any) map[string]any {
	if tags, ok := d["tags"].([]any); ok {
		slices.Reverse(tags)
	}
	schema, _ := d["jsonSchema"].(map[string]any)
	if schema == nil {
		schema = map[string]any{}
	}
	schema["$schema"] = "http://json-schema.org/draft-07/schema#"
	d["jsonSchema"] = schema
	if comp, ok := d["composition"].(map[string]any); ok {
		for _, list := range []string{"allowedProperties", "allowedSubAssets"} {
			items, _ := comp[list].([]any)
			for _, it := range items {
				m := it.(map[string]any)
				if _, has := m["minOccurs"]; !has {
					m["minOccurs"] = "0"
				}
			}
			comp[list] = items
		}
	}
	d["c8y_AvailableActions"] = map[string]any{"editAllowed": true, "editMetadataAllowed": true, "deleteAllowed": true}
	return d
}

var (
	reDef        = regexp.MustCompile(`^/service/dtm/definitions/([a-z]+)(?:/([^/]+))?$`)
	reExtAsset   = regexp.MustCompile(`^/service/dtm/assets/externalIds/([^/]+)$`)
	reSubAssign  = regexp.MustCompile(`^/service/dtm/assets/([^/]+)/subAssets/([^/]+)$`)
	reDevices    = regexp.MustCompile(`^/service/dtm/assets/([^/]+)/devices(?:/([^/]+))?$`)
	reSeries     = regexp.MustCompile(`^/service/dtm/assets/([^/]+)/linkedSeries$`)
	reNameFilter = regexp.MustCompile(`name eq '([^']+)'`)
)

func (f *statefulDTM) serve(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(r.Body)
	if r.Method != http.MethodGet {
		f.writes = append(f.writes, r.Method+" "+r.URL.Path)
	}
	p, q := r.URL.Path, r.URL.Query()
	js := func(status int, v any) {
		b, _ := json.Marshal(v)
		write := w.Header()
		write.Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write(b)
	}
	notFound := func() { js(http.StatusNotFound, map[string]any{"messages": []string{"not found: " + p}}) }

	switch {
	case p == "/inventory/managedObjects":
		var mos []any
		if m := reNameFilter.FindStringSubmatch(q.Get("query")); m != nil {
			if id, ok := f.devices[m[1]]; ok {
				mos = append(mos, map[string]any{"id": id, "name": m[1], "c8y_IsDevice": map[string]any{}})
			}
		}
		js(http.StatusOK, map[string]any{"managedObjects": mos, "statistics": map[string]any{"currentPage": 1, "pageSize": 5}})

	case reDef.MatchString(p):
		m := reDef.FindStringSubmatch(p)
		kind, ident := m[1], m[2]
		store := f.defs[kind]
		if store == nil {
			store = map[string]map[string]any{}
			f.defs[kind] = store
		}
		switch {
		case r.Method == http.MethodGet && ident != "":
			for _, d := range store {
				if d["identifier"] == ident {
					js(http.StatusOK, d)
					return
				}
			}
			notFound()
		case r.Method == http.MethodGet:
			var list []any
			ids := q.Get("identifiers")
			tag := q.Get("tags")
			for _, k := range slices.Sorted(func(yield func(string) bool) {
				for k := range store {
					if !yield(k) {
						return
					}
				}
			}) {
				d := store[k]
				if ids != "" && d["identifier"] != ids {
					continue
				}
				if tag != "" && !slices.Contains(d["tags"].([]any), any(tag)) {
					continue
				}
				list = append(list, d)
			}
			js(http.StatusOK, map[string]any{"definitions": list, "statistics": map[string]any{"currentPage": 1, "pageSize": 2000}})
		case r.Method == http.MethodPost:
			var d map[string]any
			_ = json.Unmarshal(body, &d)
			if _, exists := store[defKey(d)]; exists {
				js(http.StatusConflict, map[string]any{"messages": []string{"already exists"}})
				return
			}
			store[defKey(d)] = serverShape(d)
			js(http.StatusCreated, d)
		case r.Method == http.MethodPut:
			var d map[string]any
			_ = json.Unmarshal(body, &d)
			if ident != "" {
				d["identifier"] = ident
			}
			if _, exists := store[defKey(d)]; !exists {
				notFound()
				return
			}
			store[defKey(d)] = serverShape(d)
			js(http.StatusOK, d)
		case r.Method == http.MethodDelete:
			for k, d := range store {
				if d["identifier"] != ident {
					continue
				}
				for _, ad := range f.defs["assets"] {
					comp, _ := ad["composition"].(map[string]any)
					for _, list := range []string{"allowedProperties", "allowedSubAssets"} {
						items, _ := comp[list].([]any)
						for _, it := range items {
							if it.(map[string]any)["identifier"] == ident {
								js(http.StatusConflict, map[string]any{"messages": []string{fmt.Sprintf("Definition with identifier '%s' is still referenced in Asset Definition '%s'.", ident, ad["identifier"])}})
								return
							}
						}
					}
				}
				delete(store, k)
				w.WriteHeader(http.StatusNoContent)
				return
			}
			notFound()
		}

	case reExtAsset.MatchString(p) && r.Method == http.MethodGet:
		ext := reExtAsset.FindStringSubmatch(p)[1]
		for id, a := range f.assets {
			if a.body["c8y_ExternalId"] == ext {
				js(http.StatusOK, f.render(id, q.Get("withParents") == "true"))
				return
			}
		}
		notFound()

	case p == "/service/dtm/assets" && r.Method == http.MethodPost:
		var b map[string]any
		_ = json.Unmarshal(body, &b)
		for id, a := range f.assets {
			if a.body["c8y_ExternalId"] == b["c8y_ExternalId"] {
				if r.Header.Get(dtm.HeaderUpsertMode) != "true" {
					js(http.StatusConflict, map[string]any{"messages": []string{"externalId already used"}})
					return
				}
				for k, v := range b {
					a.body[k] = v
				}
				js(http.StatusOK, f.render(id, false))
				return
			}
		}
		f.nextID++
		id := fmt.Sprint(f.nextID)
		f.assets[id] = &fakeAsset{body: b}
		js(http.StatusCreated, f.render(id, false))

	case reSubAssign.MatchString(p) && r.Method == http.MethodPost:
		m := reSubAssign.FindStringSubmatch(p)
		child, ok1 := f.assets[m[2]]
		if _, ok2 := f.assets[m[1]]; !ok1 || !ok2 {
			notFound()
			return
		}
		if !slices.Contains(child.parents, m[1]) {
			child.parents = append(child.parents, m[1])
		}
		js(http.StatusCreated, f.render(m[1], false))

	case reDevices.MatchString(p):
		m := reDevices.FindStringSubmatch(p)
		a, ok := f.assets[m[1]]
		if !ok {
			notFound()
			return
		}
		if r.Method == http.MethodPost {
			if !slices.Contains(a.devices, m[2]) {
				a.devices = append(a.devices, m[2])
			}
			js(http.StatusCreated, f.render(m[1], false))
			return
		}
		var list []any
		for _, d := range a.devices {
			list = append(list, map[string]any{"id": d})
		}
		js(http.StatusOK, map[string]any{"devices": list, "statistics": map[string]any{"currentPage": 1, "pageSize": 2000}})

	case reSeries.MatchString(p):
		id := reSeries.FindStringSubmatch(p)[1]
		a, ok := f.assets[id]
		if !ok {
			notFound()
			return
		}
		if r.Method == http.MethodPost {
			var items []map[string]any
			_ = json.Unmarshal(body, &items)
			for _, it := range items {
				replaced := false
				for i, s := range a.series {
					if s["fragment"] == it["fragment"] && s["series"] == it["series"] {
						a.series[i], replaced = it, true
					}
				}
				if !replaced {
					a.series = append(a.series, it)
				}
			}
			js(http.StatusOK, f.render(id, false))
			return
		}
		var list []any
		for _, s := range a.series {
			if (q.Get("fragment") == "" || s["fragment"] == q.Get("fragment")) && (q.Get("series") == "" || s["series"] == q.Get("series")) {
				list = append(list, s)
			}
		}
		js(http.StatusOK, list) // bare array, as the real server

	default:
		notFound()
	}
}

func (f *statefulDTM) render(id string, withParents bool) map[string]any {
	a := f.assets[id]
	out := map[string]any{"id": id}
	for k, v := range a.body {
		out[k] = v
	}
	if withParents {
		var parents []any
		for _, p := range a.parents {
			parents = append(parents, map[string]any{"id": p})
		}
		out["assetParents"] = parents
	}
	return out
}

const modelYAML = `
kind: PropertyDefinition
identifier: maxFlow
contexts: [asset]
tags: [pumps]
jsonSchema: {title: Maximum flow, type: number}
---
kind: PropertyDefinition
identifier: rpm
contexts: [asset]
jsonSchema: {title: Speed, type: number}
---
- kind: AssetDefinition
  identifier: Site
  jsonSchema: {title: Site}
  composition:
    allowedSubAssets: [{identifier: Pump}]
- kind: AssetDefinition
  identifier: Pump
  tags: [pumps]
  jsonSchema: {title: Pump}
  composition:
    allowedProperties: [{identifier: maxFlow}, {identifier: rpm, minOccurs: "1"}]
`

const assetsJSON = `[
  {"kind": "Asset", "externalId": "pump-07", "parent": "ext:site-berlin",
   "asset": {"name": "Pump 07", "type": "Pump", "maxFlow": 120},
   "devices": ["name:pump-ctrl"],
   "linkedSeries": [{"fragment": "c8y_Flow", "series": "F", "label": "Flow", "device": "name:pump-ctrl", "sourceType": "c8y_FlowMeasurement"}]},
  {"kind": "Asset", "externalId": "site-berlin", "asset": {"name": "Berlin", "type": "Site"}}
]`

func writeModel(t *testing.T) string {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "definitions.yaml"), []byte(modelYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "assets.json"), []byte(assetsJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(filepath.Join(dir, "README.md"), []byte("ignored"), 0o644)
	return dir
}

func summary(r *dtm.ApplyReport) []string {
	if r == nil {
		return nil
	}
	var out []string
	for _, it := range r.Items {
		s := fmt.Sprintf("%s %s %s", it.Action, it.Kind, it.Key)
		if it.Err != nil {
			s += " ERR " + it.Err.Error()
		}
		out = append(out, s)
	}
	return out
}

func TestLoadModel(t *testing.T) {
	m, err := dtm.LoadModel(writeModel(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.PropertyDefinitions) != 2 || len(m.AssetDefinitions) != 2 || len(m.Assets) != 2 {
		t.Fatalf("model = %+v", m)
	}
	pump := m.Assets[0]
	if pump.ExternalID != "pump-07" || pump.Parent != "ext:site-berlin" || pump.Asset["maxFlow"] != float64(120) || len(pump.LinkedSeries) != 1 {
		t.Errorf("pump = %+v", pump)
	}

	for name, doc := range map[string]string{
		"unknown kind":        "kind: Gadget\nidentifier: x\n",
		"missing identifier":  "kind: AlarmDefinition\njsonSchema: {title: x}\n",
		"asset field at root": "kind: Asset\nexternalId: x\nname: wrong place\n",
		"missing external id": "kind: Asset\nasset: {name: x}\n",
	} {
		t.Run(name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "m.yaml")
			_ = os.WriteFile(file, []byte(doc), 0o644)
			if _, err := dtm.LoadModel(file); err == nil {
				t.Error("expected an error")
			}
		})
	}
}

func TestPlanApplyIsIdempotent(t *testing.T) {
	f := newStatefulDTM(t)
	svc := f.client()
	m, err := dtm.LoadModel(writeModel(t))
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	opt := dtm.ApplyOptions{ManagedBy: "plant-model"}

	plan := svc.Plan(ctx, m, opt)
	if plan.Err != nil {
		t.Fatalf("plan: %v\n%s", plan.Err, strings.Join(summary(plan.Data), "\n"))
	}
	want := []string{
		"create PropertyDefinition maxFlow[asset]",
		"create PropertyDefinition rpm[asset]",
		"create AssetDefinition Pump", // used by Site, so first
		"create AssetDefinition Site",
		"create Asset ext:site-berlin", // parent first
		"create Asset ext:pump-07",
		"assign SubAssetAssignment ext:site-berlin > ext:pump-07",
		"assign DeviceAssignment ext:pump-07 < name:pump-ctrl",
		"create LinkedSeries ext:pump-07 c8y_Flow.F",
	}
	if got := summary(plan.Data); !slices.Equal(got, want) {
		t.Fatalf("plan =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if !plan.Data.Planned || plan.Data.Changes() != len(want) || f.writeCount() != 0 {
		t.Fatalf("planned=%v changes=%d writes=%d", plan.Data.Planned, plan.Data.Changes(), f.writeCount())
	}

	// Executing the plan applies it.
	applied := plan.Execute(ctx)
	if applied.Err != nil {
		t.Fatalf("apply: %v\n%s", applied.Err, strings.Join(summary(applied.Data), "\n"))
	}
	if got := summary(applied.Data); !slices.Equal(got, want) {
		t.Errorf("apply =\n%s", strings.Join(got, "\n"))
	}
	writes := f.writeCount()

	// The managed-by tag was added to every definition.
	for _, d := range f.defs["properties"] {
		if !slices.Contains(d["tags"].([]any), any("plant-model")) {
			t.Errorf("definition without managed-by tag: %v", d)
		}
	}

	// Applying again changes nothing and writes nothing, despite the
	// server reordering tags and filling in composition defaults.
	again := svc.Apply(ctx, m, opt)
	if again.Err != nil || again.Data.Changes() != 0 {
		t.Fatalf("second apply: err=%v\n%s", again.Err, strings.Join(summary(again.Data), "\n"))
	}
	if f.writeCount() != writes {
		t.Errorf("second apply wrote %d times", f.writeCount()-writes)
	}
	if p := svc.Plan(ctx, m, opt); p.Err != nil || p.Data.Changes() != 0 {
		t.Errorf("plan after apply: %s", strings.Join(summary(p.Data), "\n"))
	}

	// A changed field shows up as an update.
	m.Assets[0].Asset["maxFlow"] = 150
	p := svc.Plan(ctx, m, opt)
	var updates []string
	for _, it := range p.Data.Items {
		if it.Action != dtm.ActionNone {
			updates = append(updates, fmt.Sprintf("%s %s %v", it.Action, it.Key, it.Changed))
		}
	}
	if !slices.Equal(updates, []string{"update ext:pump-07 [maxFlow]"}) {
		t.Errorf("updates = %v", updates)
	}
}

func TestPrune(t *testing.T) {
	f := newStatefulDTM(t)
	svc := f.client()
	ctx := context.Background()
	m, _ := dtm.LoadModel(writeModel(t))
	opt := dtm.ApplyOptions{ManagedBy: "plant-model", Prune: true}
	if res := svc.Apply(ctx, m, opt); res.Err != nil {
		t.Fatal(res.Err)
	}

	// Drop Site and rpm from the model. rpm is still used by Pump, so Pump
	// must lose it first; Site is deleted before nothing else depends on it.
	m.AssetDefinitions = slices.DeleteFunc(m.AssetDefinitions, func(d dtm.Definition) bool { return d.Identifier == "Site" })
	m.PropertyDefinitions = slices.DeleteFunc(m.PropertyDefinitions, func(d dtm.Definition) bool { return d.Identifier == "rpm" })
	for i := range m.AssetDefinitions {
		m.AssetDefinitions[i].Composition.AllowedProperties = []dtm.AllowedItem{{Identifier: "maxFlow"}}
	}

	plan := svc.Plan(ctx, m, opt)
	var deletes []string
	for _, it := range plan.Data.Items {
		if it.Action == dtm.ActionDelete {
			deletes = append(deletes, it.Kind+" "+it.Key)
		}
	}
	if !slices.Equal(deletes, []string{"AssetDefinition Site", "PropertyDefinition rpm[asset]"}) {
		t.Fatalf("deletes = %v", deletes)
	}

	res := svc.Apply(ctx, m, opt)
	if res.Err != nil {
		t.Fatalf("apply: %v\n%s", res.Err, strings.Join(summary(res.Data), "\n"))
	}
	if _, still := f.defs["properties"]["rpm|asset"]; still {
		t.Error("rpm was not deleted")
	}
	if p := svc.Plan(ctx, m, opt); p.Data.Changes() != 0 {
		t.Errorf("plan after prune: %s", strings.Join(summary(p.Data), "\n"))
	}

	if res := svc.Apply(ctx, m, dtm.ApplyOptions{Prune: true}); res.Err == nil {
		t.Error("Prune without ManagedBy must fail")
	}
}

func TestApplyStopsOnErrorUnlessContinue(t *testing.T) {
	f := newStatefulDTM(t)
	// A library definition the user cannot edit.
	f.defs["alarms"] = map[string]map[string]any{"c8y_Overheat|": {
		"identifier": "c8y_Overheat", "c8y_Origin": "library",
		"c8y_AvailableActions": map[string]any{"editAllowed": false},
		"jsonSchema":           map[string]any{"title": "Overheat"},
	}}
	svc := f.client()
	m := &dtm.Model{
		AlarmDefinitions: []dtm.Definition{{Identifier: "c8y_Overheat", JSONSchema: map[string]any{"title": "Too hot"}}},
		Assets:           []dtm.ModelAsset{{ExternalID: "a1", Asset: map[string]any{"name": "A1"}}},
	}

	res := svc.Apply(context.Background(), m, dtm.ApplyOptions{})
	if res.Err == nil || len(res.Data.Items) != 1 || !strings.Contains(res.Err.Error(), "read-only") {
		t.Fatalf("err=%v items=%v", res.Err, summary(res.Data))
	}
	if len(f.assets) != 0 {
		t.Error("asset written after a failure")
	}

	res = svc.Apply(context.Background(), m, dtm.ApplyOptions{ContinueOnError: true})
	if res.Err == nil || len(res.Data.Items) != 2 || len(res.Data.Errors()) != 1 || len(f.assets) != 1 {
		t.Fatalf("err=%v items=%v", res.Err, summary(res.Data))
	}

	// Plan predicts the same failure without writing.
	before := f.writeCount()
	plan := svc.Plan(context.Background(), m, dtm.ApplyOptions{})
	if len(plan.Data.Errors()) != 1 || f.writeCount() != before {
		t.Errorf("plan = %v", summary(plan.Data))
	}
}

func TestApplyRejectsCycles(t *testing.T) {
	svc := newStatefulDTM(t).client()
	defCycle := &dtm.Model{AssetDefinitions: []dtm.Definition{
		{Identifier: "A", Composition: &dtm.Composition{AllowedSubAssets: []dtm.AllowedItem{{Identifier: "B"}}}},
		{Identifier: "B", Composition: &dtm.Composition{AllowedSubAssets: []dtm.AllowedItem{{Identifier: "A"}}}},
	}}
	if res := svc.Plan(context.Background(), defCycle, dtm.ApplyOptions{}); res.Err == nil || !strings.Contains(res.Err.Error(), "cycle") {
		t.Errorf("definition cycle err = %v", res.Err)
	}
	assetCycle := &dtm.Model{Assets: []dtm.ModelAsset{
		{ExternalID: "a", Parent: "ext:b"},
		{ExternalID: "b", Parent: "ext:a"},
	}}
	if res := svc.Apply(context.Background(), assetCycle, dtm.ApplyOptions{}); res.Err == nil || !strings.Contains(res.Err.Error(), "cycle") {
		t.Errorf("asset cycle err = %v", res.Err)
	}
}

func TestApplyDeferred(t *testing.T) {
	f := newStatefulDTM(t)
	m, _ := dtm.LoadModel(writeModel(t))
	ctx := api.WithDeferredExecution(context.Background(), true)
	prepared := f.client().Apply(ctx, m, dtm.ApplyOptions{})
	if !prepared.IsDeferred() || f.writeCount() != 0 {
		t.Fatalf("deferred=%v writes=%d", prepared.IsDeferred(), f.writeCount())
	}
	if res := prepared.Execute(context.Background()); res.Err != nil || res.Data.Changes() == 0 {
		t.Errorf("execute: %v", res.Err)
	}
}
