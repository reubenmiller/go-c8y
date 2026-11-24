package dtm_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/dtm"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// Server error bodies as returned by a tenant (2026-11-24).
const (
	errExternalIDUsed  = `{"messages":["The externalId 'pump-07' is already used by ManagedObject '535469'."]}`
	errNotAChild       = `{"messages":["managedObject with id: 433659, is not a child of object with id: 535469 : Parent child relationship validation failed"]}`
	errStillReferenced = `{"messages":["Property Definition with identifier 'maxFlow' is still referenced in Asset Definition 'Pump'."]}`
	errTitleExists     = `{"messages":["Property Definition with title 'Maximum flow' already exists."]}`
	errBothExist       = `{"messages":["Property Definition with title 'Maximum flow' already exists.","Property Definition with identifier 'maxFlow' already exists in the given context."]}`
)

func reply(status int, body string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, status, body) }
}

func decode(t *testing.T, body string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		t.Fatalf("decode %q: %v", body, err)
	}
	return m
}

func wantStatus[T any](t *testing.T, r op.Result[T], want op.Status) {
	t.Helper()
	if r.Err != nil {
		t.Fatalf("unexpected error: %v", r.Err)
	}
	if r.Status != want {
		t.Fatalf("status = %s, want %s", r.Status, want)
	}
}

// --- assets ---

func TestAssetCreateAndUpdate(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodPost, dtm.ApiAssets, reply(http.StatusCreated, `{"id":"1","name":"Pump"}`))
	f.handle(http.MethodPut, "/service/dtm/assets/1", reply(http.StatusOK, `{"id":"1","name":"Pump 2"}`))
	svc := f.client()

	wantStatus(t, svc.Assets.Create(context.Background(), map[string]any{"name": "Pump"}, dtm.WriteOptions{FetchTypeFromSource: true}), op.StatusCreated)
	reqs, _ := f.sent(http.MethodPost, dtm.ApiAssets)
	if reqs[0].URL.Query().Get("fetchTypeFromSource") != "true" || reqs[0].Header.Get(dtm.HeaderUpsertMode) != "" {
		t.Errorf("create request = %v %v", reqs[0].URL.Query(), reqs[0].Header)
	}

	wantStatus(t, svc.Assets.Update(context.Background(), "1", map[string]any{"name": "Pump 2"}, dtm.WriteOptions{}), op.StatusUpdated)
}

func TestAssetDelete(t *testing.T) {
	t.Run("deleted", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodDelete, "/service/dtm/assets/1", reply(http.StatusNoContent, ""))
		wantStatus(t, f.client().Assets.Delete(context.Background(), "1", dtm.DeleteOptions{}), op.StatusNoContent)
		if q := f.last("/service/dtm/assets/1").URL.Query(); len(q) != 0 {
			t.Errorf("no cascade must send no query, got %v", q)
		}
	})
	t.Run("cascade", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodDelete, "/service/dtm/assets/1", reply(http.StatusNoContent, ""))
		mustOK(t, f.client().Assets.Delete(context.Background(), "1", dtm.DeleteOptions{Cascade: dtm.CascadeSubAssets | dtm.CascadeDevices}))
		q := f.last("/service/dtm/assets/1").URL.Query()
		if q.Get("deleteSubAssets") != "true" || q.Get("deleteDevices") != "true" {
			t.Errorf("cascade query = %v", q)
		}
	})
	t.Run("already gone", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodDelete, "/service/dtm/assets/1", reply(http.StatusNotFound, `{"messages":["Asset with id '1' not found."]}`))
		wantStatus(t, f.client().Assets.Delete(context.Background(), "1", dtm.DeleteOptions{}), op.StatusSkipped)
	})
	t.Run("unknown external id sends no delete", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/assets/externalIds/gone", reply(http.StatusNotFound, `{"messages":["Asset with externalId 'gone' not found."]}`))
		wantStatus(t, f.client().Assets.Delete(context.Background(), dtm.AssetByExternalID("gone"), dtm.DeleteOptions{}), op.StatusSkipped)
		for _, r := range f.reqs() {
			if r.Method == http.MethodDelete {
				t.Error("DELETE sent for a missing asset")
			}
		}
	})
	t.Run("other errors are returned", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodDelete, "/service/dtm/assets/1", reply(http.StatusForbidden, `{"messages":["Access denied"]}`))
		if res := f.client().Assets.Delete(context.Background(), "1", dtm.DeleteOptions{}); !core.ErrHasStatus(res.Err, 403) {
			t.Errorf("err = %v", res.Err)
		}
	})
}

func TestAssetGetOrCreate(t *testing.T) {
	t.Run("creates", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodPost, dtm.ApiAssets, reply(http.StatusCreated, `{"id":"1","c8y_ExternalId":"pump-07"}`))
		res := f.client().Assets.GetOrCreate(context.Background(), dtm.AssetCreateOptions{
			ExternalID: "pump-07",
			Body:       map[string]any{"name": "Pump 07"},
		}, model.Frag("c8y_ExternalAsset", map[string]any{"system": "SAP"}))
		wantStatus(t, res, op.StatusCreated)
		_, bodies := f.sent(http.MethodPost, dtm.ApiAssets)
		body := decode(t, bodies[0])
		if body["c8y_ExternalId"] != "pump-07" || body["name"] != "Pump 07" || body["c8y_ExternalAsset"] == nil {
			t.Errorf("create body = %v", body)
		}
	})
	t.Run("returns existing on conflict", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodPost, dtm.ApiAssets, reply(http.StatusConflict, errExternalIDUsed))
		f.handle(http.MethodGet, "/service/dtm/assets/externalIds/pump-07", reply(http.StatusOK, `{"id":"535469","c8y_ExternalId":"pump-07"}`))
		res := f.client().Assets.GetOrCreate(context.Background(), dtm.AssetCreateOptions{ExternalID: "pump-07", Body: map[string]any{"name": "x"}})
		wantStatus(t, res, op.StatusOK)
		if res.Data.ID() != "535469" || res.Meta["existing"] != true {
			t.Errorf("result = %s %v", res.Data.Bytes(), res.Meta)
		}
	})
	t.Run("validates before sending", func(t *testing.T) {
		f := newFakeDTM(t)
		svc := f.client()
		if res := svc.Assets.GetOrCreate(context.Background(), dtm.AssetCreateOptions{Body: map[string]any{}}); res.Err == nil {
			t.Error("expected an error without ExternalID")
		}
		if res := svc.Assets.GetOrCreate(context.Background(), dtm.AssetCreateOptions{
			ExternalID: "a",
			Body:       map[string]any{"c8y_ExternalId": "b"},
		}); res.Err == nil {
			t.Error("expected an error for a conflicting c8y_ExternalId")
		}
		if n := len(f.reqs()); n != 0 {
			t.Errorf("sent %d requests", n)
		}
	})
}

func TestAssetUpsert(t *testing.T) {
	existing := `{"id":"1","name":"Pump 07","type":"Pump","c8y_ExternalId":"pump-07","maxFlow":120,"lastUpdated":"2026-11-24T00:00:00Z","c8y_ExternalAsset":{"syncedAt":"old"}}`
	syncFrag := model.Frag("c8y_ExternalAsset", map[string]any{"syncedAt": time.Now().Format(time.RFC3339)})

	t.Run("skips when unchanged, ignoring annotations", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/assets/externalIds/pump-07", reply(http.StatusOK, existing))
		res := f.client().Assets.Upsert(context.Background(), dtm.AssetUpsertOptions{
			AssetCreateOptions: dtm.AssetCreateOptions{ExternalID: "pump-07", Body: map[string]any{"name": "Pump 07", "maxFlow": 120}},
		}, syncFrag)
		wantStatus(t, res, op.StatusSkipped)
		if reqs, _ := f.sent(http.MethodPost, dtm.ApiAssets); len(reqs) != 0 {
			t.Error("POST sent for an unchanged asset")
		}
	})
	t.Run("updates with server upsert", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/assets/externalIds/pump-07", reply(http.StatusOK, existing))
		f.handle(http.MethodPost, dtm.ApiAssets, reply(http.StatusOK, `{"id":"1","maxFlow":150}`))
		res := f.client().Assets.Upsert(context.Background(), dtm.AssetUpsertOptions{
			AssetCreateOptions: dtm.AssetCreateOptions{ExternalID: "pump-07", Body: map[string]any{"maxFlow": 150}},
		}, syncFrag)
		wantStatus(t, res, op.StatusUpdated)
		reqs, bodies := f.sent(http.MethodPost, dtm.ApiAssets)
		if reqs[0].Header.Get(dtm.HeaderUpsertMode) != "true" {
			t.Error("missing X-Upsert-Mode header")
		}
		if body := decode(t, bodies[0]); body["c8y_ExternalAsset"] == nil || body["c8y_ExternalId"] != "pump-07" {
			t.Errorf("annotations and external id must be written on update: %v", body)
		}
	})
	t.Run("creates when missing", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/assets/externalIds/pump-07", reply(http.StatusNotFound, `{"messages":["not found"]}`))
		f.handle(http.MethodPost, dtm.ApiAssets, reply(http.StatusCreated, `{"id":"1"}`))
		res := f.client().Assets.Upsert(context.Background(), dtm.AssetUpsertOptions{
			AssetCreateOptions: dtm.AssetCreateOptions{ExternalID: "pump-07", Body: map[string]any{"name": "Pump 07"}},
		})
		wantStatus(t, res, op.StatusCreated)
	})
	t.Run("skip compare sends one request", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodPost, dtm.ApiAssets, reply(http.StatusOK, `{"id":"1"}`))
		res := f.client().Assets.Upsert(context.Background(), dtm.AssetUpsertOptions{
			AssetCreateOptions: dtm.AssetCreateOptions{ExternalID: "pump-07", Body: map[string]any{"name": "Pump 07"}},
			SkipCompare:        true,
		})
		wantStatus(t, res, op.StatusUpdated)
		if n := len(f.reqs()); n != 1 {
			t.Errorf("sent %d requests, want 1", n)
		}
	})
	t.Run("deferred sends nothing until Execute", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/assets/externalIds/pump-07", reply(http.StatusOK, existing))
		ctx := api.WithDeferredExecution(context.Background(), true)
		prepared := f.client().Assets.Upsert(ctx, dtm.AssetUpsertOptions{
			AssetCreateOptions: dtm.AssetCreateOptions{ExternalID: "pump-07", Body: map[string]any{"name": "Pump 07"}},
		})
		if !prepared.IsDeferred() || len(f.reqs()) != 0 {
			t.Fatalf("deferred=%v requests=%d", prepared.IsDeferred(), len(f.reqs()))
		}
		wantStatus(t, prepared.Execute(context.Background()), op.StatusSkipped)
	})
}

// --- sub-assets and devices ---

func TestUnassign(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   op.Status
		err    bool
	}{
		{"removed", http.StatusNoContent, "", op.StatusNoContent, false},
		{"not assigned", http.StatusUnprocessableEntity, errNotAChild, op.StatusSkipped, false},
		{"missing", http.StatusNotFound, `{"messages":["not found"]}`, op.StatusSkipped, false},
		{"other 422 is an error", http.StatusUnprocessableEntity, `{"messages":["managedObject with id: 1 is not an asset"]}`, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakeDTM(t)
			f.handle(http.MethodDelete, "/service/dtm/assets/1/subAssets/2", reply(tt.status, tt.body))
			f.handle(http.MethodDelete, "/service/dtm/assets/1/devices/3", reply(tt.status, tt.body))
			svc := f.client()

			for _, res := range []op.Result[core.NoContent]{
				svc.Assets.SubAssets.Unassign(context.Background(), "1", "2"),
				svc.Assets.Devices.Unassign(context.Background(), "1", "3"),
			} {
				if tt.err {
					if res.Err == nil {
						t.Error("expected an error")
					}
					continue
				}
				wantStatus(t, res, tt.want)
			}
		})
	}
}

func TestAssignDeviceByName(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodGet, "/inventory/managedObjects", reply(http.StatusOK, `{"managedObjects":[{"id":"555476","name":"pump-ctrl"}],"statistics":{"currentPage":1,"pageSize":5}}`))
	f.handle(http.MethodPost, "/service/dtm/assets/1/devices/555476", reply(http.StatusCreated, `{"id":"1"}`))

	res := f.client().Assets.Devices.Assign(context.Background(), "1", "name:pump-ctrl")
	wantStatus(t, res, op.StatusCreated)
	if res.Meta["deviceId"] != "555476" {
		t.Errorf("meta = %v", res.Meta)
	}
}

func TestSubAssetGetOrCreate(t *testing.T) {
	t.Run("existing is assigned", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/assets/externalIds/hall-1", reply(http.StatusOK, `{"id":"20","c8y_ExternalId":"hall-1"}`))
		f.handle(http.MethodPost, "/service/dtm/assets/10/subAssets/20", reply(http.StatusCreated, `{"id":"10"}`))
		res := f.client().Assets.SubAssets.GetOrCreate(context.Background(), "10", dtm.AssetCreateOptions{ExternalID: "hall-1", Body: map[string]any{"name": "Hall 1"}})
		wantStatus(t, res, op.StatusOK)
		if res.Data.ID() != "20" {
			t.Errorf("data = %s, want the sub-asset", res.Data.Bytes())
		}
		if reqs, _ := f.sent(http.MethodPost, "/service/dtm/assets/10/subAssets"); len(reqs) != 0 {
			t.Error("an existing sub-asset must not be created again")
		}
	})
	t.Run("missing is created", func(t *testing.T) {
		f := newFakeDTM(t)
		gets := 0
		f.handle(http.MethodGet, "/service/dtm/assets/externalIds/hall-1", func(w http.ResponseWriter, _ *http.Request) {
			gets++
			if gets == 1 {
				writeJSON(w, http.StatusNotFound, `{"messages":["not found"]}`)
				return
			}
			writeJSON(w, http.StatusOK, `{"id":"20","c8y_ExternalId":"hall-1"}`)
		})
		f.handle(http.MethodPost, "/service/dtm/assets/10/subAssets", reply(http.StatusCreated, `{"id":"10"}`))
		res := f.client().Assets.SubAssets.GetOrCreate(context.Background(), "10", dtm.AssetCreateOptions{ExternalID: "hall-1", Body: map[string]any{"name": "Hall 1"}})
		wantStatus(t, res, op.StatusCreated)
		if res.Data.ID() != "20" {
			t.Errorf("data = %s, want the new sub-asset", res.Data.Bytes())
		}
		_, bodies := f.sent(http.MethodPost, "/service/dtm/assets/10/subAssets")
		if body := decode(t, bodies[0]); body["c8y_ExternalId"] != "hall-1" {
			t.Errorf("create body = %v", body)
		}
	})
}

// --- linked series ---

func TestLink(t *testing.T) {
	current := `[{"fragment":"c8y_Flow","series":"F","label":"Flow rate","source":{"id":"555476","fragment":"c8y_Flow","series":"F","type":"c8y_FlowMeasurement"}}]`

	t.Run("unchanged is skipped", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/assets/1/linkedSeries", reply(http.StatusOK, current))
		res := f.client().Assets.LinkedSeries.Link(context.Background(), "1", dtm.LinkOptions{Fragment: "c8y_Flow", Series: "F", Device: "555476"})
		wantStatus(t, res, op.StatusSkipped)
		if reqs, _ := f.sent(http.MethodPost, "/service/dtm/assets/1/linkedSeries"); len(reqs) != 0 {
			t.Error("POST sent for an unchanged link")
		}
	})
	t.Run("new source is updated", func(t *testing.T) {
		f := newFakeDTM(t)
		calls := 0
		f.handle(http.MethodGet, "/service/dtm/assets/1/linkedSeries", func(w http.ResponseWriter, _ *http.Request) {
			calls++
			if calls == 1 {
				writeJSON(w, http.StatusOK, current)
				return
			}
			writeJSON(w, http.StatusOK, `[{"fragment":"c8y_Flow","series":"F","label":"Flow","source":{"id":"777","fragment":"c8y_FlowMeter","series":"rate"}}]`)
		})
		f.handle(http.MethodPost, "/service/dtm/assets/1/linkedSeries", reply(http.StatusOK, `{"id":"1"}`))
		res := f.client().Assets.LinkedSeries.Link(context.Background(), "1", dtm.LinkOptions{
			Fragment: "c8y_Flow", Series: "F", Label: "Flow",
			Device: "777", SourceFragment: "c8y_FlowMeter", SourceSeries: "rate",
		})
		wantStatus(t, res, op.StatusUpdated)
		if res.Data.Source().ID() != "777" {
			t.Errorf("data = %s", res.Data.Bytes())
		}
		_, bodies := f.sent(http.MethodPost, "/service/dtm/assets/1/linkedSeries")
		var sent []map[string]any
		_ = json.Unmarshal([]byte(bodies[0]), &sent)
		src, _ := sent[0]["source"].(map[string]any)
		if len(sent) != 1 || src["id"] != "777" || src["fragment"] != "c8y_FlowMeter" || src["series"] != "rate" {
			t.Errorf("POST body = %s", bodies[0])
		}
	})
	t.Run("missing is created", func(t *testing.T) {
		f := newFakeDTM(t)
		calls := 0
		f.handle(http.MethodGet, "/service/dtm/assets/1/linkedSeries", func(w http.ResponseWriter, _ *http.Request) {
			calls++
			if calls == 1 {
				writeJSON(w, http.StatusOK, `[]`)
				return
			}
			writeJSON(w, http.StatusOK, current)
		})
		f.handle(http.MethodPost, "/service/dtm/assets/1/linkedSeries", reply(http.StatusOK, `{"id":"1"}`))
		res := f.client().Assets.LinkedSeries.Link(context.Background(), "1", dtm.LinkOptions{Fragment: "c8y_Flow", Series: "F", Device: "555476"})
		wantStatus(t, res, op.StatusCreated)
	})
}

func TestLinkedSeriesDeleteAndLabel(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodDelete, "/service/dtm/assets/1/linkedSeries", reply(http.StatusNotFound, `{"messages":["not found"]}`))
	f.handle(http.MethodPut, "/service/dtm/assets/1/linkedSeries/c8y_Flow/F/label", reply(http.StatusOK, `{"fragment":"c8y_Flow","series":"F","label":"Flow"}`))
	svc := f.client()

	wantStatus(t, svc.Assets.LinkedSeries.Delete(context.Background(), "1", "c8y_Flow", "F"), op.StatusSkipped)
	q := f.last("/service/dtm/assets/1/linkedSeries").URL.Query()
	if q.Get("fragment") != "c8y_Flow" || q.Get("series") != "F" {
		t.Errorf("delete query = %v", q)
	}

	res := svc.Assets.LinkedSeries.SetLabel(context.Background(), "1", "c8y_Flow", "F", "Flow")
	mustOK(t, res)
	reqs, bodies := f.sent(http.MethodPut, "/service/dtm/assets/1/linkedSeries/c8y_Flow/F/label")
	if !strings.HasPrefix(reqs[0].Header.Get("Content-Type"), "text/plain") || bodies[0] != "Flow" {
		t.Errorf("label request = %q %q", reqs[0].Header.Get("Content-Type"), bodies[0])
	}
}

// --- definitions ---

func TestDefinitionErrors(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodDelete, "/service/dtm/definitions/properties/maxFlow", reply(http.StatusConflict, errStillReferenced))
	f.handle(http.MethodDelete, "/service/dtm/definitions/assets/gone", reply(http.StatusNotFound, `{"messages":["Asset Definition with identifier 'gone' not found."]}`))
	f.handle(http.MethodPost, "/service/dtm/definitions/properties", reply(http.StatusConflict, errTitleExists))
	f.handle(http.MethodPost, "/service/dtm/definitions/events", reply(http.StatusConflict, errBothExist))
	svc := f.client()

	res := svc.Definitions.Properties.Delete(context.Background(), "maxFlow", dtm.DeleteDefinitionOptions{Contexts: []string{"asset", "event"}})
	if !errors.Is(res.Err, dtm.ErrDefinitionInUse) || !core.ErrHasStatus(res.Err, http.StatusConflict) {
		t.Errorf("in-use err = %v", res.Err)
	}
	if got := f.last("/service/dtm/definitions/properties/maxFlow").URL.Query().Get("contexts"); got != "asset,event" {
		t.Errorf("contexts = %q", got)
	}

	wantStatus(t, svc.Definitions.Assets.Delete(context.Background(), "gone", dtm.DeleteDefinitionOptions{}), op.StatusSkipped)

	res2 := svc.Definitions.Properties.Create(context.Background(), dtm.Definition{Identifier: "newId", JSONSchema: map[string]any{"title": "Maximum flow"}})
	if !errors.Is(res2.Err, dtm.ErrDuplicateTitle) {
		t.Errorf("title err = %v", res2.Err)
	}
	// Identifier and title both taken: not a title-only conflict.
	res3 := svc.Definitions.Events.Create(context.Background(), dtm.Definition{Identifier: "maxFlow"})
	if errors.Is(res3.Err, dtm.ErrDuplicateTitle) || !core.ErrHasStatus(res3.Err, http.StatusConflict) {
		t.Errorf("both-taken err = %v", res3.Err)
	}
}

func TestDefinitionUpdatePaths(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodPut, "/service/dtm/definitions/properties/maxFlow", reply(http.StatusOK, `{"identifier":"maxFlow"}`))
	f.handle(http.MethodPut, "/service/dtm/definitions/alarms", reply(http.StatusOK, `{"identifier":"overheat"}`))
	svc := f.client()

	mustOK(t, svc.Definitions.Properties.Update(context.Background(), "maxFlow", map[string]any{"tags": []string{"a"}}, dtm.UpdateDefinitionOptions{Contexts: []string{"asset"}}))
	if got := f.last("/service/dtm/definitions/properties/maxFlow").URL.Query().Get("contexts"); got != "asset" {
		t.Errorf("contexts = %q", got)
	}

	mustOK(t, svc.Definitions.Alarms.Update(context.Background(), "overheat", map[string]any{"tags": []string{"a"}}, dtm.UpdateDefinitionOptions{}))
	_, bodies := f.sent(http.MethodPut, "/service/dtm/definitions/alarms")
	if body := decode(t, bodies[0]); body["identifier"] != "overheat" {
		t.Errorf("alarm update body must carry the identifier: %v", body)
	}
}

func TestDefinitionUpsert(t *testing.T) {
	// Stored as the server returns it: extra schema fields, defaults filled
	// in, tags in another order.
	storedAsset := `{"identifier":"Pump","tags":["b","a"],"c8y_AvailableActions":{"editAllowed":true},
		"composition":{"additionalProperties":false,"additionalSubAssets":false,"allowedProperties":[{"identifier":"maxFlow","minOccurs":"0"},{"identifier":"rpm","minOccurs":"1"}],"allowedSubAssets":[]},
		"jsonSchema":{"$schema":"http://json-schema.org/draft-07/schema#","title":"Pump","additionalProperties":false,"properties":{"maxFlow":{"$ref":"#/definitions/maxFlow"}}}}`
	pump := dtm.Definition{
		Identifier: "Pump",
		Tags:       []string{"a", "b"},
		JSONSchema: map[string]any{"title": "Pump"},
		Composition: &dtm.Composition{AllowedProperties: []dtm.AllowedItem{
			{Identifier: "rpm", MinOccurs: "1"},
			{Identifier: "maxFlow"},
		}},
	}

	t.Run("server defaults and order are not changes", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/definitions/assets/Pump", reply(http.StatusOK, storedAsset))
		wantStatus(t, f.client().Definitions.Assets.Upsert(context.Background(), pump), op.StatusSkipped)
		if reqs, _ := f.sent(http.MethodPut, "/service/dtm/definitions/assets"); len(reqs) != 0 {
			t.Error("PUT sent for an unchanged definition")
		}
	})
	t.Run("asset definitions are sent without merging the schema", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/definitions/assets/Pump", reply(http.StatusOK, storedAsset))
		f.handle(http.MethodPut, "/service/dtm/definitions/assets", reply(http.StatusOK, `{"identifier":"Pump"}`))
		changed := pump
		changed.Composition = &dtm.Composition{AllowedProperties: []dtm.AllowedItem{{Identifier: "rpm", MinOccurs: "1"}}}
		res := f.client().Definitions.Assets.Upsert(context.Background(), changed)
		wantStatus(t, res, op.StatusUpdated)
		if got := res.Meta["changed"]; !slices.Equal(got.([]string), []string{"composition"}) {
			t.Errorf("changed = %v", got)
		}
		_, bodies := f.sent(http.MethodPut, "/service/dtm/definitions/assets")
		schema, _ := decode(t, bodies[0])["jsonSchema"].(map[string]any)
		if _, has := schema["properties"]; has {
			t.Errorf("asset definition schema must not include stale generated properties: %v", schema)
		}
	})
	t.Run("measurement schema is merged with the stored one", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/definitions/measurements/flow", reply(http.StatusOK,
			`{"identifier":"flow","c8y_AvailableActions":{"editAllowed":true},"jsonSchema":{"title":"Flow","required":["source"],"properties":{"source":{"$ref":"#/definitions/source"}}}}`))
		f.handle(http.MethodPut, "/service/dtm/definitions/measurements", reply(http.StatusOK, `{"identifier":"flow"}`))
		res := f.client().Definitions.Measurements.Upsert(context.Background(), dtm.Definition{
			Identifier: "flow",
			JSONSchema: map[string]any{"title": "Flow", "description": "Litres per minute"},
		})
		wantStatus(t, res, op.StatusUpdated)
		_, bodies := f.sent(http.MethodPut, "/service/dtm/definitions/measurements")
		schema, _ := decode(t, bodies[0])["jsonSchema"].(map[string]any)
		props, _ := schema["properties"].(map[string]any)
		if schema["description"] != "Litres per minute" || props["source"] == nil || schema["required"] == nil {
			t.Errorf("merged schema = %v", schema)
		}
	})
	t.Run("read-only definitions are not written", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/definitions/alarms/c8y_Overheat", reply(http.StatusOK,
			`{"identifier":"c8y_Overheat","c8y_Origin":"library","c8y_AvailableActions":{"editAllowed":false,"editMetadataAllowed":true},"jsonSchema":{"title":"Overheat"}}`))
		svc := f.client()
		wantStatus(t, svc.Definitions.Alarms.Upsert(context.Background(), dtm.Definition{Identifier: "c8y_Overheat", JSONSchema: map[string]any{"title": "Overheat"}}), op.StatusSkipped)
		res := svc.Definitions.Alarms.Upsert(context.Background(), dtm.Definition{Identifier: "c8y_Overheat", JSONSchema: map[string]any{"title": "Too hot"}})
		if !errors.Is(res.Err, dtm.ErrReadOnlyDefinition) {
			t.Errorf("err = %v", res.Err)
		}
		if reqs, _ := f.sent(http.MethodPut, "/service/dtm/definitions/alarms"); len(reqs) != 0 {
			t.Error("PUT sent for a read-only definition")
		}
	})
	t.Run("property definitions match on contexts", func(t *testing.T) {
		f := newFakeDTM(t)
		f.handle(http.MethodGet, "/service/dtm/definitions/properties", reply(http.StatusOK,
			`{"definitions":[{"identifier":"maxFlow","contexts":["event"],"jsonSchema":{"title":"Max flow (event)"}}],"statistics":{"currentPage":1,"pageSize":100}}`))
		f.handle(http.MethodPost, "/service/dtm/definitions/properties", reply(http.StatusCreated, `{"identifier":"maxFlow","contexts":["asset"]}`))
		res := f.client().Definitions.Properties.Upsert(context.Background(), dtm.Definition{
			Identifier: "maxFlow",
			Contexts:   []string{"asset"},
			JSONSchema: map[string]any{"title": "Maximum flow", "type": "number"},
		})
		wantStatus(t, res, op.StatusCreated)
		gets, _ := f.sent(http.MethodGet, "/service/dtm/definitions/properties")
		if q := gets[0].URL.Query(); q.Get("identifiers") != "maxFlow" {
			t.Errorf("lookup query = %v", q)
		}
	})
}

// Regression: for kinds other than properties, a missing definition used to
// come back from Find with its 404 error, so Upsert and GetOrCreate failed
// instead of creating it.
func TestDefinitionUpsertCreatesMissing(t *testing.T) {
	for _, kind := range []string{"assets", "measurements", "events", "alarms"} {
		t.Run(kind, func(t *testing.T) {
			f := newFakeDTM(t)
			f.handle(http.MethodGet, "/service/dtm/definitions/"+kind+"/new", reply(http.StatusNotFound, `{"messages":["not found"]}`))
			f.handle(http.MethodPost, "/service/dtm/definitions/"+kind, reply(http.StatusCreated, `{"identifier":"new"}`))
			svc := f.client()
			byKind := map[string]*dtm.DefinitionService{
				"assets": svc.Definitions.Assets, "measurements": svc.Definitions.Measurements,
				"events": svc.Definitions.Events, "alarms": svc.Definitions.Alarms,
			}
			def := dtm.Definition{Identifier: "new", JSONSchema: map[string]any{"title": "New"}}
			wantStatus(t, byKind[kind].Upsert(context.Background(), def), op.StatusCreated)
			wantStatus(t, byKind[kind].GetOrCreate(context.Background(), def), op.StatusCreated)
		})
	}
}

func TestDefinitionGetOrCreate(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodGet, "/service/dtm/definitions/events/door", reply(http.StatusOK, `{"identifier":"door","jsonSchema":{"title":"Door"}}`))
	res := f.client().Definitions.Events.GetOrCreate(context.Background(), dtm.Definition{Identifier: "door", JSONSchema: map[string]any{"title": "Changed"}})
	wantStatus(t, res, op.StatusOK)
	if reqs, _ := f.sent(http.MethodPost, "/service/dtm/definitions/events"); len(reqs) != 0 {
		t.Error("GetOrCreate must not write an existing definition")
	}
}

func TestDefinitionExtraFragments(t *testing.T) {
	b, err := json.Marshal(dtm.Definition{Identifier: "x", Extra: map[string]any{"icon": map[string]any{"name": "pump"}, "identifier": "ignored"}})
	if err != nil {
		t.Fatal(err)
	}
	m := decode(t, string(b))
	if m["identifier"] != "x" || m["icon"] == nil {
		t.Errorf("marshalled = %s", b)
	}
}

func TestCompose(t *testing.T) {
	f := newFakeDTM(t)
	f.handle(http.MethodPost, dtm.ApiDefinitionsCompose, reply(http.StatusOK, `{"type":"object","properties":{"maxFlow":{"type":"number"}}}`))
	svc := f.client()

	res := svc.Definitions.Properties.Compose(context.Background(), []dtm.ComposeItem{{Identifier: "maxFlow", Context: dtm.ContextAsset}})
	if mustOK(t, res).Get("properties.maxFlow.type").String() != "number" {
		t.Errorf("schema = %s", res.Data.Bytes())
	}
	if _, bodies := f.sent(http.MethodPost, dtm.ApiDefinitionsCompose); strings.TrimSpace(bodies[0]) != `[{"identifier":"maxFlow","context":"asset"}]` {
		t.Errorf("compose body = %s", bodies[0])
	}
	if res := svc.Definitions.Events.Compose(context.Background(), []dtm.ComposeItem{{Identifier: "a"}}); res.Err == nil {
		t.Error("compose must fail for non-property definitions")
	}
}
