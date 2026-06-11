package smartresttemplates

import (
	"context"
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"resty.dev/v3"
)

// testServer simulates the identity and inventory endpoints used by the
// service. existing == nil means the external identity does not exist.
type testServer struct {
	server   *httptest.Server
	existing map[string]any // existing managed object (id 123), nil = not found

	createdBody  map[string]any // body of POST /inventory/managedObjects
	updatedBody  map[string]any // body of PUT /inventory/managedObjects/123
	identityBody map[string]any // body of POST /identity/globalIds/.../externalIds
	deletedMO    bool
}

func newTestServer(existing map[string]any) *testServer {
	ts := &testServer{existing: existing}
	ts.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		path := r.URL.Path

		readBody := func() map[string]any {
			raw, _ := io.ReadAll(r.Body)
			var body map[string]any
			_ = json.Unmarshal(raw, &body)
			return body
		}

		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(path, "/identity/externalIds/"+ExternalIDType+"/"):
			if ts.existing == nil {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":"identity/Not Found"}`))
				return
			}
			name := path[strings.LastIndex(path, "/")+1:]
			_ = json.NewEncoder(w).Encode(map[string]any{
				"externalId":    name,
				"type":          ExternalIDType,
				"managedObject": map[string]any{"id": "123"},
			})

		case r.Method == http.MethodGet && path == "/inventory/managedObjects/123":
			_ = json.NewEncoder(w).Encode(ts.existing)

		case r.Method == http.MethodPost && path == "/inventory/managedObjects":
			ts.createdBody = readBody()
			created := map[string]any{}
			maps.Copy(created, ts.createdBody)
			created["id"] = "456"
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(created)

		case r.Method == http.MethodPut && path == "/inventory/managedObjects/123":
			ts.updatedBody = readBody()
			updated := map[string]any{}
			maps.Copy(updated, ts.updatedBody)
			updated["id"] = "123"
			_ = json.NewEncoder(w).Encode(updated)

		case r.Method == http.MethodPost && strings.HasPrefix(path, "/identity/globalIds/"):
			ts.identityBody = readBody()
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(ts.identityBody)

		case r.Method == http.MethodDelete && strings.HasPrefix(path, "/inventory/managedObjects/"):
			ts.deletedMO = true
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"unexpected request: ` + r.Method + ` ` + path + `"}`))
		}
	}))
	return ts
}

func (ts *testServer) service() *Service {
	return NewService(&core.Service{
		Client: resty.New().SetBaseURL(ts.server.URL),
	})
}

func templates(items ...map[string]any) []map[string]any { return items }

func responseTemplate(msgID string, pattern ...string) map[string]any {
	patternValues := make([]any, 0, len(pattern))
	for _, p := range pattern {
		patternValues = append(patternValues, p)
	}
	return map[string]any{
		"msgId":   msgID,
		"name":    "tpl_" + msgID,
		"base":    "base_" + msgID,
		"pattern": patternValues,
	}
}

// existingCollection builds the server-side managed object document for the
// given templates, including platform bookkeeping fields not part of the
// desired state.
func existingCollection(name string, requestTemplates, responseTemplates []map[string]any) map[string]any {
	return map[string]any{
		"id":           "123",
		"name":         name,
		"type":         ManagedObjectType,
		"__externalId": name,
		"lastUpdated":  "2026-01-01T00:00:00.000Z",
		"owner":        "admin",
		FragmentTemplates: map[string]any{
			"requestTemplates":  requestTemplates,
			"responseTemplates": responseTemplates,
		},
	}
}

func TestUpsertCreatesMissingCollection(t *testing.T) {
	ts := newTestServer(nil)
	defer ts.server.Close()

	result := ts.service().Upsert(context.Background(), CreateOptions{
		Name:              "custom_devmgmt",
		ResponseTemplates: templates(responseTemplate("dm101", "name", "ssid")),
		Annotations:       []model.Fragment{model.Frag("c8y_TenantSync", map[string]any{"tool": "tenant-sync"})},
	})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusCreated, result.Status)
	assert.Equal(t, false, result.Meta["found"])

	require.NotNil(t, ts.createdBody)
	assert.Equal(t, "custom_devmgmt", ts.createdBody["name"])
	assert.Equal(t, ManagedObjectType, ts.createdBody["type"])
	assert.Equal(t, "custom_devmgmt", ts.createdBody[FragmentExternalID])
	// Annotations are written on create
	assert.Contains(t, ts.createdBody, "c8y_TenantSync")
	// Empty template lists marshal as [] (like the platform export format)
	fragment := ts.createdBody[FragmentTemplates].(map[string]any)
	assert.Equal(t, []any{}, fragment["requestTemplates"])
	assert.Len(t, fragment["responseTemplates"], 1)

	// The external identity is registered for the created managed object
	require.NotNil(t, ts.identityBody)
	assert.Equal(t, "custom_devmgmt", ts.identityBody["externalId"])
	assert.Equal(t, ExternalIDType, ts.identityBody["type"])
}

func TestUpsertSkipsUnchangedCollection(t *testing.T) {
	// The existing collection has the same templates in a different order,
	// extra platform fields, and an older annotation
	existing := existingCollection("custom_devmgmt",
		templates(),
		templates(responseTemplate("dm102", "b"), responseTemplate("dm101", "a")),
	)
	existing["c8y_TenantSync"] = map[string]any{"tool": "tenant-sync", "syncedAt": "old"}
	ts := newTestServer(existing)
	defer ts.server.Close()

	result := ts.service().Upsert(context.Background(), CreateOptions{
		Name:              "custom_devmgmt",
		ResponseTemplates: templates(responseTemplate("dm101", "a"), responseTemplate("dm102", "b")),
		Annotations:       []model.Fragment{model.Frag("c8y_TenantSync", map[string]any{"tool": "tenant-sync", "syncedAt": "new"})},
	})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusSkipped, result.Status)
	assert.Equal(t, true, result.Meta["found"])
	assert.Nil(t, ts.updatedBody, "no update must be sent when the desired state matches")
}

func TestUpsertUpdatesChangedCollection(t *testing.T) {
	existing := existingCollection("custom_devmgmt",
		templates(),
		templates(responseTemplate("dm101", "a")),
	)
	ts := newTestServer(existing)
	defer ts.server.Close()

	result := ts.service().Upsert(context.Background(), CreateOptions{
		Name:              "custom_devmgmt",
		ResponseTemplates: templates(responseTemplate("dm101", "a", "extra-field")),
		Annotations:       []model.Fragment{model.Frag("c8y_TenantSync", map[string]any{"tool": "tenant-sync"})},
	})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusUpdated, result.Status)

	require.NotNil(t, ts.updatedBody)
	assert.Contains(t, ts.updatedBody, "c8y_TenantSync", "annotations are written on real updates")
	fragment := ts.updatedBody[FragmentTemplates].(map[string]any)
	assert.Len(t, fragment["responseTemplates"], 1)
}

func TestSortedTemplatesDoesNotMutateInput(t *testing.T) {
	input := templates(responseTemplate("dm102"), responseTemplate("dm101"))
	sorted := sortedTemplates(input)

	assert.Equal(t, "dm101", sorted[0]["msgId"])
	assert.Equal(t, "dm102", sorted[1]["msgId"])
	assert.Equal(t, "dm102", input[0]["msgId"], "input order must be preserved")
}

func TestNormalizeTemplateOrder(t *testing.T) {
	doc := []byte(`{
		"id": "123",
		"` + FragmentTemplates + `": {
			"requestTemplates": [],
			"responseTemplates": [{"msgId": "dm102"}, {"msgId": "dm101"}]
		}
	}`)
	var normalized map[string]any
	require.NoError(t, json.Unmarshal(normalizeTemplateOrder(doc), &normalized))

	items := normalized[FragmentTemplates].(map[string]any)["responseTemplates"].([]any)
	assert.Equal(t, "dm101", items[0].(map[string]any)["msgId"])
	assert.Equal(t, "dm102", items[1].(map[string]any)["msgId"])

	// Documents without the fragment are passed through unchanged
	passthrough := []byte(`{"id": "1"}`)
	assert.Equal(t, passthrough, normalizeTemplateOrder(passthrough))
}

func TestDeleteMissingCollectionIsSkipped(t *testing.T) {
	ts := newTestServer(nil)
	defer ts.server.Close()

	result := ts.service().Delete(context.Background(), "does-not-exist")
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusSkipped, result.Status)
	assert.False(t, ts.deletedMO)
}

func TestDeleteExistingCollection(t *testing.T) {
	ts := newTestServer(existingCollection("custom_devmgmt", templates(), templates()))
	defer ts.server.Close()

	result := ts.service().Delete(context.Background(), "custom_devmgmt")
	require.NoError(t, result.Err)
	assert.True(t, ts.deletedMO)
}
