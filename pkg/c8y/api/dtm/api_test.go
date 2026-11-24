package dtm_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/dtm"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// fakeDTM serves recorded responses keyed by "METHOD /path" and records the
// requests it receives.
type fakeDTM struct {
	t      *testing.T
	server *httptest.Server
	routes map[string]func(w http.ResponseWriter, r *http.Request)

	mu       sync.Mutex
	requests []*http.Request
}

func newFakeDTM(t *testing.T) *fakeDTM {
	t.Helper()
	f := &fakeDTM{t: t, routes: map[string]func(http.ResponseWriter, *http.Request){}}
	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, r.Clone(context.Background()))
		f.mu.Unlock()
		h, ok := f.routes[r.Method+" "+r.URL.Path]
		if !ok {
			writeJSON(w, http.StatusNotFound, `{"messages":["no route for `+r.Method+" "+r.URL.Path+`"]}`)
			return
		}
		h(w, r)
	}))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeDTM) handle(method, path string, h func(w http.ResponseWriter, r *http.Request)) {
	f.routes[method+" "+path] = h
}

// fixture serves a testdata file.
func (f *fakeDTM) fixture(method, path, file string) {
	body, err := os.ReadFile(filepath.Join("testdata", file))
	if err != nil {
		f.t.Fatalf("read fixture %s: %v", file, err)
	}
	f.handle(method, path, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, string(body))
	})
}

func (f *fakeDTM) reqs() []*http.Request {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.requests)
}

// last returns the last request to path, or nil.
func (f *fakeDTM) last(path string) *http.Request {
	reqs := f.reqs()
	for i := len(reqs) - 1; i >= 0; i-- {
		if reqs[i].URL.Path == path {
			return reqs[i]
		}
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func (f *fakeDTM) client() *dtm.Service {
	c := api.NewClient(api.ClientOptions{
		BaseURL:          f.server.URL,
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "user",
			Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})
	return c.DigitalTwin
}

func mustOK[T any](t *testing.T, r op.Result[T]) T {
	t.Helper()
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	return r.Data
}

func collect[T any](r op.Result[T]) []T {
	var out []T
	for item := range r.Seq() {
		out = append(out, item)
	}
	return out
}

func TestAssetsList(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, dtm.ApiAssets, "assets_list.json")
	svc := f.client()

	res := svc.Assets.List(context.Background(), dtm.ListOptions{
		Query:             "name eq 'gocy-proposal*'",
		WithParents:       true,
		WithChildrenCount: true,
	})
	mustOK(t, res)

	assets := collect(res)
	if len(assets) != 2 {
		t.Fatalf("got %d assets, want 2", len(assets))
	}
	site, pump := assets[0], assets[1]
	if site.ExternalID() != "gocy-proposal-site" || site.Notes() != "Test site" || !site.IsDeviceGroup() {
		t.Errorf("site = %s", site.Bytes())
	}
	if n, ok := site.ChildrenCount(); !ok || n != 1 {
		t.Errorf("site ChildrenCount = %d, %v", n, ok)
	}
	var parents []string
	for p := range pump.Parents() {
		parents = append(parents, p.ID())
	}
	if !slices.Equal(parents, []string{"383668"}) {
		t.Errorf("pump parents = %v", parents)
	}

	q := f.last(dtm.ApiAssets).URL.Query()
	if got := q.Get("query"); got != "$filter=(name eq 'gocy-proposal*')" {
		t.Errorf("query = %q, want bare filter wrapped in $filter=(...)", got)
	}
	if q.Get("withParents") != "true" || q.Get("withChildrenCount") != "true" {
		t.Errorf("query params = %v", q)
	}
	if q.Has("onlyRoots") || q.Has("withLinkedSeries") {
		t.Errorf("false options must not be sent: %v", q)
	}
}

func TestAssetsGetByID(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, "/service/dtm/assets/205477", "asset_pump.json")
	svc := f.client()

	asset := mustOK(t, svc.Assets.Get(context.Background(), "205477", dtm.GetOptions{WithLinkedSeries: true}))
	if asset.Name() != "gocy-proposal pump" {
		t.Errorf("name = %q", asset.Name())
	}
	var series []string
	for ls := range asset.LinkedSeries() {
		series = append(series, ls.Fragment()+"."+ls.Series()+"="+ls.Source().ID()+":"+ls.Source().Type())
	}
	if !slices.Equal(series, []string{"c8y_Flow.F=555476:c8y_FlowMeasurement"}) {
		t.Errorf("linked series = %v", series)
	}
	if len(f.reqs()) != 1 {
		t.Errorf("a plain id must not trigger a lookup, got %d requests", len(f.reqs()))
	}
}

func TestAssetsGetByExternalIDUsesDirectEndpoint(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, "/service/dtm/assets/externalIds/gocy-proposal-pump", "asset_pump.json")
	svc := f.client()

	asset := mustOK(t, svc.Assets.Get(context.Background(), dtm.AssetByExternalID("gocy-proposal-pump"), dtm.GetOptions{
		WithParents:      true,
		WithLatestValues: true, // not supported by the endpoint, must be dropped
	}))
	if asset.ID() != "205477" {
		t.Errorf("id = %q", asset.ID())
	}
	reqs := f.reqs()
	if len(reqs) != 1 {
		t.Fatalf("got %d requests, want 1 (no separate lookup)", len(reqs))
	}
	if q := reqs[0].URL.Query(); q.Get("withParents") != "true" || q.Has("withLatestValues") {
		t.Errorf("query = %v", q)
	}
}

func TestAssetsGetByName(t *testing.T) {
	tests := []struct {
		name     string
		list     string
		wantErr  func(error) bool
		wantGets int
	}{
		{
			name:     "one match",
			list:     `{"assets":[{"id":"205477","name":"gocy-proposal pump"}],"statistics":{"currentPage":1,"pageSize":2}}`,
			wantGets: 1,
		},
		{
			name:    "no match is a 404",
			list:    `{"assets":[],"statistics":{"currentPage":1,"pageSize":2}}`,
			wantErr: core.IsNotFound,
		},
		{
			name:    "several matches are ambiguous",
			list:    `{"assets":[{"id":"1"},{"id":"2"}],"statistics":{"currentPage":1,"pageSize":2}}`,
			wantErr: func(err error) bool { return errors.Is(err, dtm.ErrAmbiguousAssetRef) },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeDTM(t)
			f.handle(http.MethodGet, dtm.ApiAssets, func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusOK, tt.list)
			})
			gets := 0
			f.handle(http.MethodGet, "/service/dtm/assets/205477", func(w http.ResponseWriter, _ *http.Request) {
				gets++
				writeJSON(w, http.StatusOK, `{"id":"205477","name":"gocy-proposal pump"}`)
			})
			svc := f.client()

			res := svc.Assets.Get(context.Background(), dtm.AssetByName("gocy-proposal pump"), dtm.GetOptions{})
			if tt.wantErr != nil {
				if res.Err == nil || !tt.wantErr(res.Err) {
					t.Fatalf("err = %v", res.Err)
				}
			} else {
				mustOK(t, res)
				if res.Meta["id"] != "205477" || res.Meta["source"] != "name" {
					t.Errorf("meta = %v", res.Meta)
				}
			}
			if gets != tt.wantGets {
				t.Errorf("GET by id calls = %d, want %d", gets, tt.wantGets)
			}
			q := f.last(dtm.ApiAssets).URL.Query()
			if q.Get("query") != "$filter=(name eq 'gocy-proposal pump')" || q.Get("pageSize") != "2" {
				t.Errorf("lookup query = %v", q)
			}
		})
	}
}

func TestDeferredGetResolvesButDoesNotSend(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodGet, dtm.ApiAssets, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, `{"assets":[{"id":"205477"}],"statistics":{"currentPage":1,"pageSize":2}}`)
	})
	f.fixture(http.MethodGet, "/service/dtm/assets/205477", "asset_pump.json")
	svc := f.client()

	ctx := api.WithDeferredExecution(context.Background(), true)
	prepared := svc.Assets.Get(ctx, dtm.AssetByName("gocy-proposal pump"), dtm.GetOptions{})
	if prepared.Err != nil {
		t.Fatalf("prepare: %v", prepared.Err)
	}
	if !prepared.IsDeferred() {
		t.Fatal("result is not deferred")
	}
	if got := prepared.Request.URL.Path; got != "/service/dtm/assets/205477" {
		t.Errorf("prepared path = %q, want the resolved id", got)
	}
	if f.last("/service/dtm/assets/205477") != nil {
		t.Fatal("GET was sent before Execute")
	}

	res := prepared.Execute(context.Background())
	if mustOK(t, res).Name() != "gocy-proposal pump" {
		t.Errorf("executed result = %s", res.Data.Bytes())
	}
}

func TestAssetsCount(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodGet, dtm.ApiAssetsCount, func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, "2\n")
	})
	n := mustOK(t, f.client().Assets.Count(context.Background(), dtm.CountOptions{Query: "name eq 'x'", OnlyRoots: true}))
	if n != 2 {
		t.Errorf("count = %d", n)
	}
	q := f.last(dtm.ApiAssetsCount).URL.Query()
	if q.Get("query") != "$filter=(name eq 'x')" || q.Get("onlyRoots") != "true" {
		t.Errorf("query = %v", q)
	}
}

func TestErrorMessagesAreParsed(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodGet, "/service/dtm/assets/externalIds/missing", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusNotFound, `{"messages":["Asset with externalId 'missing' not found."]}`)
	})
	res := f.client().Assets.Get(context.Background(), dtm.AssetByExternalID("missing"), dtm.GetOptions{})
	if !core.IsNotFound(res.Err) {
		t.Fatalf("err = %v, want 404", res.Err)
	}
	if !strings.Contains(res.Err.Error(), "Asset with externalId 'missing' not found.") {
		t.Errorf("error %q does not include the server message", res.Err)
	}
}

func TestSubAssetsByExternalIDParent(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, "/service/dtm/assets/externalIds/gocy-proposal-site/subAssets", "subassets_by_external_id.json")
	svc := f.client()

	res := svc.Assets.SubAssets.List(context.Background(), dtm.AssetByExternalID("gocy-proposal-site"), dtm.SubAssetListOptions{})
	children := collect(res)
	if mustOK(t, res); len(children) != 1 || children[0].ExternalID() != "gocy-proposal-pump" {
		t.Fatalf("children = %v", children)
	}

	// ListAll keeps using the external id endpoint (no lookup) and stops
	// after the last page.
	var ids []string
	for a, err := range svc.Assets.SubAssets.ListAll(context.Background(), dtm.AssetByExternalID("gocy-proposal-site"), dtm.SubAssetListOptions{}).Items() {
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, a.ID())
	}
	if !slices.Equal(ids, []string{"205477"}) {
		t.Errorf("ListAll ids = %v", ids)
	}
	for _, r := range f.reqs() {
		if r.URL.Path != "/service/dtm/assets/externalIds/gocy-proposal-site/subAssets" {
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}
}

func TestListByParentsRequiresParents(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, dtm.ApiSubAssetsByParents, "subassets_by_external_id.json")
	svc := f.client()

	if res := svc.Assets.SubAssets.ListByParents(context.Background(), dtm.ListByParentsOptions{}); res.Err == nil {
		t.Error("expected an error without parents")
	}
	mustOK(t, svc.Assets.SubAssets.ListByParents(context.Background(), dtm.ListByParentsOptions{Parents: []string{"1", "2"}}))
	if got := f.last(dtm.ApiSubAssetsByParents).URL.Query().Get("parents"); got != "1,2" {
		t.Errorf("parents = %q, want comma-separated", got)
	}
}

func TestAssetDevices(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, "/service/dtm/assets/205477/devices", "asset_devices.json")
	res := f.client().Assets.Devices.List(context.Background(), "205477", dtm.DeviceListOptions{})
	devices := collect(res)
	if mustOK(t, res); len(devices) != 1 || devices[0].Name() != "gocy-proposal-device" {
		t.Fatalf("devices = %v", devices)
	}
}

func TestLinkedSeriesBareArray(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, "/service/dtm/assets/205477/linkedSeries", "linked_series.json")
	svc := f.client()

	res := svc.Assets.LinkedSeries.List(context.Background(), "205477", dtm.LinkedSeriesListOptions{WithMeasurementType: true})
	series := collect(res)
	if mustOK(t, res); len(series) != 1 {
		t.Fatalf("got %d series, want 1", len(series))
	}
	s := series[0]
	if s.Label() != "Flow rate" || !s.HasSource() || s.Source().ID() != "555476" || s.Source().Series() != "F" {
		t.Errorf("series = %s", s.Bytes())
	}

	var n int
	for _, err := range svc.Assets.LinkedSeries.ListAll(context.Background(), "205477", dtm.LinkedSeriesListOptions{}).Items() {
		if err != nil {
			t.Fatal(err)
		}
		n++
	}
	if n != 1 {
		t.Errorf("ListAll yielded %d items, want 1", n)
	}
}

func TestLinkedSeriesEnvelopeAlsoWorks(t *testing.T) {
	// The spec describes a paged envelope. Accept it too, in case the server
	// is changed to match the spec.
	f := newFakeDTM(t)
	f.handle(http.MethodGet, "/service/dtm/assets/205477/linkedSeries", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, `{"linkedSeries":[{"fragment":"c8y_Flow","series":"F"}],"statistics":{"currentPage":1,"pageSize":5}}`)
	})
	res := f.client().Assets.LinkedSeries.List(context.Background(), "205477", dtm.LinkedSeriesListOptions{})
	mustOK(t, res)
	if series := collect(res); len(series) != 1 || series[0].Fragment() != "c8y_Flow" {
		t.Fatalf("series = %v", series)
	}
}

func TestLinkedSeriesSourceAndAcrossAssets(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, "/service/dtm/assets/205477/linkedSeries/c8y_Flow/F/source", "linked_series_source.json")
	f.fixture(http.MethodGet, dtm.ApiAllLinkedSeries, "linked_series_across_assets.json")
	svc := f.client()

	src := mustOK(t, svc.Assets.LinkedSeries.GetSource(context.Background(), "205477", "c8y_Flow", "F"))
	if src.ID() != "555476" || src.Type() != "c8y_FlowMeasurement" {
		t.Errorf("source = %s", src.Bytes())
	}
	if res := svc.Assets.LinkedSeries.GetSource(context.Background(), "205477", "", "F"); res.Err == nil {
		t.Error("expected an error for an empty fragment")
	}

	res := svc.Assets.LinkedSeries.ListAcrossAssets(context.Background(), dtm.ListAcrossAssetsOptions{AssetIDs: []string{"205477"}})
	assets := collect(res)
	if mustOK(t, res); len(assets) != 1 {
		t.Fatalf("assets = %v", assets)
	}
	if got := f.last(dtm.ApiAllLinkedSeries).URL.Query().Get("assetIds"); got != "205477" {
		t.Errorf("assetIds = %q", got)
	}
}

func TestOpposites(t *testing.T) {
	f := newFakeDTM(t)
	var accept string
	f.handle(http.MethodGet, "/service/dtm/assets/linkedSeries/opposites/555476", func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		body, _ := os.ReadFile(filepath.Join("testdata", "opposites.json"))
		writeJSON(w, http.StatusOK, string(body))
	})
	res := f.client().Assets.LinkedSeries.Opposites(context.Background(), "555476", dtm.OppositesOptions{})
	links := collect(res)
	if mustOK(t, res); len(links) != 1 {
		t.Fatalf("links = %v", links)
	}
	l := links[0]
	if l.AssetID() != "205477" || l.AssetLabel() != "Flow rate" || l.Type() != "c8y_FlowMeasurement" {
		t.Errorf("link = %s", l.Bytes())
	}
	if accept != dtm.MimeTypeLinkedAssetsCollection {
		t.Errorf("Accept = %q", accept)
	}
}

func TestDefinitionsListScopesOptionsByKind(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, "/service/dtm/definitions/properties", "definitions_properties.json")
	f.handle(http.MethodGet, "/service/dtm/definitions/measurements", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, `{"definitions":[],"statistics":{"currentPage":1,"pageSize":5}}`)
	})
	svc := f.client()

	opt := dtm.ListDefinitionsOptions{
		Tags:         []string{"a", "b"},
		Origin:       "library",
		ApplicableTo: dtm.ContextAsset,
		OnlyRoots:    true,
	}
	res := svc.Definitions.Properties.List(context.Background(), opt)
	defs := collect(res)
	if mustOK(t, res); len(defs) != 2 {
		t.Fatalf("got %d definitions", len(defs))
	}
	pos := defs[0]
	if pos.Identifier() != "c8y_Position" || pos.Title() != "Location" || pos.Origin() != "library" {
		t.Errorf("definition = %s", pos.Bytes())
	}
	if !slices.Contains(pos.Contexts(), dtm.ContextAsset) || pos.CanEdit() || !pos.CanEditMetadata() {
		t.Errorf("contexts/actions = %v %v %v", pos.Contexts(), pos.CanEdit(), pos.CanEditMetadata())
	}
	if got := pos.JSONSchema().Get("properties.lat.maximum").Int(); got != 90 {
		t.Errorf("schema lat.maximum = %d", got)
	}
	if res.Meta["totalPages"] != int64(4) {
		t.Errorf("meta = %v", res.Meta)
	}

	pq := f.last("/service/dtm/definitions/properties").URL.Query()
	wantProps := url.Values{"tags": {"a,b"}, "origin": {"library"}, "applicableTo": {"asset"}}
	for k, v := range wantProps {
		if pq.Get(k) != v[0] {
			t.Errorf("properties %s = %q, want %q", k, pq.Get(k), v[0])
		}
	}
	if pq.Has("onlyRoots") {
		t.Error("onlyRoots must not be sent for property definitions")
	}

	mustOK(t, svc.Definitions.Measurements.List(context.Background(), opt))
	mq := f.last("/service/dtm/definitions/measurements").URL.Query()
	if mq.Has("origin") || mq.Has("applicableTo") || mq.Has("onlyRoots") || mq.Get("tags") != "a,b" {
		t.Errorf("measurement query = %v", mq)
	}
}

func TestDefinitionGetAndComposition(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, "/service/dtm/definitions/assets/c8y_GenericAsset", "definition_asset_generic.json")
	f.fixture(http.MethodGet, "/service/dtm/definitions/assets/gocyProposalAsset", "definition_asset_composed.json")
	svc := f.client()

	generic := mustOK(t, svc.Definitions.Assets.Get(context.Background(), "c8y_GenericAsset", dtm.GetDefinitionOptions{ApplicableTo: "asset"}))
	if generic.Title() != "Generic Asset" || !generic.AdditionalProperties() || !generic.AdditionalSubAssets() {
		t.Errorf("generic = %s", generic.Bytes())
	}
	if len(generic.AllowedProperties()) != 0 {
		t.Errorf("allowed properties = %v", generic.AllowedProperties())
	}
	if f.last("/service/dtm/definitions/assets/c8y_GenericAsset").URL.Query().Has("applicableTo") {
		t.Error("applicableTo must only be sent for property definitions")
	}

	composed := mustOK(t, svc.Definitions.Assets.Get(context.Background(), "gocyProposalAsset", dtm.GetDefinitionOptions{}))
	want := []string{"gocyProposalProp:1"}
	var got []string
	for _, p := range composed.AllowedProperties() {
		got = append(got, p.Identifier+":"+p.MinOccurs)
	}
	if !slices.Equal(got, want) {
		t.Errorf("allowed properties = %v, want %v", got, want)
	}
	if !slices.Equal(composed.Tags(), []string{"gocy-proposal", "changed"}) {
		t.Errorf("tags = %v", composed.Tags())
	}
}

func TestDefinitionTagsAndCount(t *testing.T) {
	f := newFakeDTM(t)
	f.fixture(http.MethodGet, "/service/dtm/definitions/properties/tags", "definitions_properties_tags.json")
	f.handle(http.MethodGet, "/service/dtm/definitions/alarms/count", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, "9")
	})
	svc := f.client()

	tags := mustOK(t, svc.Definitions.Properties.Tags(context.Background(), dtm.TagsOptions{}))
	if !slices.Equal(tags, []string{"thin-edge.io-zephyr-snmp", "thin-edge.io-posix", "tiny-edge"}) {
		t.Errorf("tags = %v", tags)
	}

	// Asset definitions have no tags endpoint: fail without a request.
	before := len(f.reqs())
	if res := svc.Definitions.Assets.Tags(context.Background(), dtm.TagsOptions{}); res.Err == nil {
		t.Error("expected an error for asset definition tags")
	}
	if len(f.reqs()) != before {
		t.Error("asset definition Tags sent a request")
	}

	n := mustOK(t, svc.Definitions.Alarms.Count(context.Background(), dtm.CountDefinitionsOptions{OnlyRoots: true, ApplicableTo: "asset"}))
	if n != 9 {
		t.Errorf("count = %d", n)
	}
	if q := f.last("/service/dtm/definitions/alarms/count").URL.Query(); len(q) != 0 {
		t.Errorf("alarm count query = %v, want no kind-specific params", q)
	}
}

func TestDryRunSendsNothing(t *testing.T) {
	f := newFakeDTM(t)
	svc := f.client()
	ctx := api.WithDryRun(context.Background(), true)

	res := svc.Definitions.Events.List(ctx, dtm.ListDefinitionsOptions{Identifiers: []string{"a"}})
	if res.Request == nil || res.Request.URL.Path != "/service/dtm/definitions/events" {
		t.Fatalf("dry run request = %v", res.Request)
	}
	if res.Request.URL.Query().Get("identifiers") != "a" {
		t.Errorf("dry run query = %v", res.Request.URL.Query())
	}
	if n := len(f.reqs()); n != 0 {
		t.Errorf("dry run sent %d requests", n)
	}
}
