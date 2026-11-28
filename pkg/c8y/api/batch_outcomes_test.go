package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pipeline"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recorded struct {
	Method string
	Path   string
	Query  string
	Body   string
}

// scriptServer answers each request with handler and records all requests.
type scriptServer struct {
	mu       sync.Mutex
	requests []recorded
	server   *httptest.Server
}

func newScriptServer(t *testing.T, handler func(r recorded, n int) (int, string)) *scriptServer {
	t.Helper()
	ss := &scriptServer{}
	ss.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		rec := recorded{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(b)}
		ss.mu.Lock()
		ss.requests = append(ss.requests, rec)
		n := len(ss.requests)
		ss.mu.Unlock()
		status, body := handler(rec, n)
		w.Header().Set("Content-Type", "application/json")
		if status >= 400 {
			w.Header().Set("Content-Type", "application/vnd.com.nsn.cumulocity.error+json")
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(ss.server.Close)
	return ss
}

func (ss *scriptServer) matching(method, path string) []recorded {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	var out []recorded
	for _, r := range ss.requests {
		if r.Method == method && r.Path == path {
			out = append(out, r)
		}
	}
	return out
}

func (ss *scriptServer) total() int {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return len(ss.requests)
}

func newScriptClient(ss *scriptServer) *Client {
	c := mustNewClient(ClientOptions{
		BaseURL:          ss.server.URL,
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant: "t100", Username: "user", Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})
	c.Resty().SetRetryWaitTime(0).SetRetryMaxWaitTime(0)
	return c
}

func bulkErr(errType string, failedIndex int, marker string) string {
	bulk := map[string]any{"failedIndex": failedIndex}
	if marker != "" {
		bulk["fragmentType"] = marker
	}
	b, _ := json.Marshal(map[string]any{"error": errType, "message": "bulk failure", "bulkRequest": bulk})
	return string(b)
}

func mos(objs ...map[string]any) string {
	b, _ := json.Marshal(map[string]any{"managedObjects": objs})
	return string(b)
}

func items(n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = map[string]any{"name": fmt.Sprintf("item-%d", i)}
	}
	return out
}

const moPath = "/inventory/managedObjects"

// --- batch-outcomes ---

func Test_BatchError_ValidationFailure(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		return 422, bulkErr("undefined/validationError", 2, "")
	})
	result := newScriptClient(ss).ManagedObjects.CreateBulk(context.Background(), items(3))

	var be *core.BatchError
	require.True(t, errors.As(result.Err, &be), "expected *core.BatchError, got %T", result.Err)
	assert.Equal(t, 2, be.Index)
	assert.Equal(t, "", be.Marker)
	assert.Equal(t, core.BatchAppliedNone, be.Applied)

	var ce *core.Error
	require.True(t, errors.As(result.Err, &ce), "errors.As(*core.Error) must still work")
	assert.Equal(t, 422, ce.StatusCode())
	assert.Equal(t, op.StatusFailed, result.Status)
}

func Test_BatchError_WholeRequestErrorIsPlain(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		return 422, `{"error":"undefined/validationError","message":"too many items"}`
	})
	result := newScriptClient(ss).ManagedObjects.UpdateBulk(context.Background(), []any{map[string]any{"id": "1"}})

	var be *core.BatchError
	assert.False(t, errors.As(result.Err, &be))
	var ce *core.Error
	assert.True(t, errors.As(result.Err, &ce))
}

func Test_BatchError_MeasurementsUseSameType(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		return 422, bulkErr("undefined/validationError", 0, "")
	})
	result := newScriptClient(ss).Measurements.CreateList(context.Background(), map[string]any{
		"measurements": []any{map[string]any{"type": "t"}},
	})
	var be *core.BatchError
	require.True(t, errors.As(result.Err, &be), "expected *core.BatchError, got %T: %v", result.Err, result.Err)
	assert.Equal(t, 0, be.Index)
}

func Test_BatchError_PartialCreate(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		return 409, bulkErr("identity/Duplicate", 3, "c8y_BatchId_abc")
	})
	result := newScriptClient(ss).ManagedObjects.CreateBulk(context.Background(), items(5))

	var be *core.BatchError
	require.True(t, errors.As(result.Err, &be))
	assert.Equal(t, core.BatchAppliedPartial, be.Applied)
	assert.Equal(t, "c8y_BatchId_abc", be.Marker)
	assert.Equal(t, op.StatusPartial, result.Status)
	assert.True(t, result.IsError())
	assert.False(t, result.Retryable)
}

func Test_BatchError_UnknownWithoutMarker(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		return 500, bulkErr("inventory/Internal Server Error", 17, "")
	})
	client := newScriptClient(ss)
	client.Resty().SetRetryCount(0)
	result := client.ManagedObjects.UpdateBulk(context.Background(), []any{map[string]any{"id": "1"}})

	var be *core.BatchError
	require.True(t, errors.As(result.Err, &be))
	assert.Equal(t, core.BatchAppliedUnknown, be.Applied)
	assert.Equal(t, 17, be.Index)
}

func Test_BatchError_BulkCreateNeverResent(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		return 500, bulkErr("inventory/Internal Server Error", 1, "c8y_BatchId_abc")
	})
	result := newScriptClient(ss).ManagedObjects.CreateBulk(context.Background(), items(3))
	require.Error(t, result.Err)
	assert.Len(t, ss.matching(http.MethodPost, moPath), 1, "a failed bulk create must not be resent")
}

func Test_BatchRecoverer_Signature(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) { return 200, "{}" })
	var recoverer pipeline.BatchRecoverer[jsonmodels.ManagedObject] = newScriptClient(ss).ManagedObjects.RecoverCreateBulk
	assert.NotNil(t, recoverer)
}

// --- managed-object-bulk ---

func Test_Bulk_SizeLimits(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) { return 200, mos() })
	client := newScriptClient(ss)

	tooMany := client.ManagedObjects.CreateBulk(context.Background(), items(51))
	require.Error(t, tooMany.Err)
	assert.Contains(t, tooMany.Err.Error(), "50")

	empty := client.ManagedObjects.UpdateBulk(context.Background(), nil)
	require.Error(t, empty.Err)
	assert.Contains(t, empty.Err.Error(), "at least 1")

	assert.Equal(t, 0, ss.total(), "invalid bulk requests must not be sent")
}

func Test_Bulk_WithExternalIDs(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		return 201, `{"id":"1","name":"dev1"}`
	})
	body := managedobjects.WithExternalIDs(map[string]any{"name": "dev1"},
		managedobjects.ExternalID{Type: "c8y_Serial", ExternalID: "SN1"})
	result := newScriptClient(ss).ManagedObjects.Create(context.Background(), body)
	require.NoError(t, result.Err)

	sent := ss.matching(http.MethodPost, moPath)
	require.Len(t, sent, 1)
	var got map[string]any
	require.NoError(t, json.Unmarshal([]byte(sent[0].Body), &got))
	assert.Equal(t, []any{map[string]any{"type": "c8y_Serial", "externalId": "SN1"}}, got["externalIds"])
}

func Test_Bulk_DuplicateExternalIDInRequest(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) { return 200, mos() })
	id := managedobjects.ExternalID{Type: "c8y_Serial", ExternalID: "SN1"}
	result := newScriptClient(ss).ManagedObjects.CreateBulk(context.Background(), []any{
		managedobjects.WithExternalIDs(map[string]any{"name": "a"}, id),
		map[string]any{"name": "b"},
		managedobjects.WithExternalIDs(map[string]any{"name": "c"}, id),
	})
	require.Error(t, result.Err)
	assert.Contains(t, result.Err.Error(), "items 0 and 2")
	assert.Equal(t, 0, ss.total())
}

// recoveryServer simulates a partial bulk create: positions in `created` exist
// with the marker; the first lookupFailures lookups fail with 503.
func recoveryServer(t *testing.T, marker string, created []int, lookupFailures int) *scriptServer {
	lookups := 0
	var mu sync.Mutex
	return newScriptServer(t, func(r recorded, n int) (int, string) {
		mu.Lock()
		defer mu.Unlock()
		switch {
		case r.Method == http.MethodGet && r.Path == moPath:
			lookups++
			if lookups <= lookupFailures {
				return 503, `{"error":"general/serviceUnavailable","message":"busy"}`
			}
			var objs []map[string]any
			for _, i := range created {
				objs = append(objs, map[string]any{"id": fmt.Sprintf("old-%d", i), "name": fmt.Sprintf("item-%d", i), marker: map[string]any{"index": i}})
			}
			return 200, mos(objs...)
		case r.Method == http.MethodPost && r.Path == moPath:
			var env struct {
				ManagedObjects []map[string]any `json:"managedObjects"`
			}
			_ = json.Unmarshal([]byte(r.Body), &env)
			out := make([]map[string]any, 0, len(env.ManagedObjects))
			for _, mo := range env.ManagedObjects {
				out = append(out, map[string]any{"id": "new-" + fmt.Sprint(mo["name"]), "name": mo["name"]})
			}
			return 200, mos(out...)
		case r.Method == http.MethodPut && r.Path == moPath:
			return 200, mos()
		}
		return 404, `{"error":"x/notFound"}`
	})
}

func Test_RecoverCreateBulk_CreatesOnlyMissing(t *testing.T) {
	marker := "c8y_BatchId_abc"
	ss := recoveryServer(t, marker, []int{0, 1}, 0)
	client := newScriptClient(ss)
	in := items(5)
	be := &core.BatchError{Index: 3, Applied: core.BatchAppliedPartial, Marker: marker}

	out, err := client.ManagedObjects.RecoverCreateBulk(context.Background(), in, be)
	require.NoError(t, err)
	require.Len(t, out, 5)
	assert.Equal(t, []string{"old-0", "old-1", "new-item-2", "new-item-3", "new-item-4"},
		[]string{out[0].ID(), out[1].ID(), out[2].ID(), out[3].ID(), out[4].ID()})

	posts := ss.matching(http.MethodPost, moPath)
	require.Len(t, posts, 1)
	for _, name := range []string{"item-2", "item-3", "item-4"} {
		assert.Contains(t, posts[0].Body, name)
	}
	assert.NotContains(t, posts[0].Body, `"item-0"`)

	puts := ss.matching(http.MethodPut, moPath)
	require.Len(t, puts, 1, "the marker is removed with one bulk update")
	assert.Contains(t, puts[0].Body, `"`+marker+`":null`)
	assert.Contains(t, puts[0].Body, "old-0")
	assert.Contains(t, puts[0].Body, "old-1")
}

func Test_RecoverCreateBulk_RejectsNonPartial(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) { return 200, mos() })
	_, err := newScriptClient(ss).ManagedObjects.RecoverCreateBulk(context.Background(), items(2),
		&core.BatchError{Applied: core.BatchAppliedNone})
	require.Error(t, err)
	assert.Equal(t, 0, ss.total())
}

func Test_CreateBulkStream_OffsetOnFailure(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		if r.Method == http.MethodPost && n == 3 {
			return 422, bulkErr("undefined/validationError", 4, "")
		}
		var env struct {
			ManagedObjects []map[string]any `json:"managedObjects"`
		}
		_ = json.Unmarshal([]byte(r.Body), &env)
		return 200, mos(env.ManagedObjects...)
	})
	stream := newScriptClient(ss).ManagedObjects.CreateBulkStream(context.Background(),
		pipeline.FromSlice(items(120)), managedobjects.BulkStreamOptions{})

	var gotErr error
	count := 0
	for _, err := range stream.Items() {
		if err != nil {
			gotErr = err
			break
		}
		count++
	}
	assert.Equal(t, 100, count)
	var be *core.BatchError
	require.True(t, errors.As(gotErr, &be), "expected *core.BatchError, got %v", gotErr)
	assert.Equal(t, 100, be.Offset)
	assert.Equal(t, 4, be.Index)
	assert.Contains(t, be.Error(), "item 104")
}

func Test_CreateBulkStream_RecoversPartial(t *testing.T) {
	marker := "c8y_BatchId_abc"
	var mu sync.Mutex
	posts := 0
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		mu.Lock()
		defer mu.Unlock()
		var env struct {
			ManagedObjects []map[string]any `json:"managedObjects"`
		}
		_ = json.Unmarshal([]byte(r.Body), &env)
		switch r.Method {
		case http.MethodPost:
			posts++
			if posts == 2 { // second batch: items 2,3 failed after item 2 was written
				return 409, bulkErr("identity/Duplicate", 1, marker)
			}
			out := []map[string]any{}
			for _, mo := range env.ManagedObjects {
				out = append(out, map[string]any{"id": "id-" + fmt.Sprint(mo["name"]), "name": mo["name"]})
			}
			return 200, mos(out...)
		case http.MethodGet:
			return 200, mos(map[string]any{"id": "id-item-2", "name": "item-2", marker: map[string]any{"index": 0}})
		case http.MethodPut:
			return 200, mos()
		}
		return 404, "{}"
	})
	stream := newScriptClient(ss).ManagedObjects.CreateBulkStream(context.Background(),
		pipeline.FromSlice(items(6)), managedobjects.BulkStreamOptions{BatchSize: 2, RecoverPartial: true})

	var ids []string
	for mo, err := range stream.Items() {
		require.NoError(t, err)
		ids = append(ids, mo.ID())
	}
	assert.Equal(t, []string{"id-item-0", "id-item-1", "id-item-2", "id-item-3", "id-item-4", "id-item-5"}, ids)
	assert.Equal(t, 1, countValue(ids, "id-item-2"), "the written item must not be created twice")
}

func countValue(values []string, v string) int {
	n := 0
	for _, x := range values {
		if x == v {
			n++
		}
	}
	return n
}

func Test_GetOrCreateByExternalID_Atomic(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.Path, "/identity/externalIds/"):
			return 404, `{"error":"identity/Not Found","message":"not found"}`
		case r.Method == http.MethodPost && r.Path == moPath:
			return 201, `{"id":"42","name":"dev"}`
		}
		return 500, `{"error":"unexpected"}`
	})
	result := newScriptClient(ss).ManagedObjects.GetOrCreateByExternalID(context.Background(),
		managedobjects.GetOrCreateByExternalIDOptions{ExternalID: "SN1", ExternalIDType: "c8y_Serial", Body: map[string]any{"name": "dev"}})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusCreated, result.Status)

	posts := ss.matching(http.MethodPost, moPath)
	require.Len(t, posts, 1)
	assert.Contains(t, posts[0].Body, `"externalIds"`)
	assert.Empty(t, ss.matching(http.MethodPost, "/identity/globalIds/42/externalIds"), "no separate identity request")
}

func Test_GetOrCreateByExternalID_ConcurrentCreation(t *testing.T) {
	lookups := 0
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.Path, "/identity/externalIds/"):
			lookups++
			if lookups == 1 {
				return 404, `{"error":"identity/Not Found","message":"not found"}`
			}
			return 200, `{"externalId":"SN1","type":"c8y_Serial","managedObject":{"id":"7"}}`
		case r.Method == http.MethodPost && r.Path == moPath:
			return 409, `{"error":"identity/Duplicate","message":"already bound"}`
		case r.Method == http.MethodGet && r.Path == moPath+"/7":
			return 200, `{"id":"7","name":"existing"}`
		}
		return 500, `{"error":"unexpected"}`
	})
	result := newScriptClient(ss).ManagedObjects.GetOrCreateByExternalID(context.Background(),
		managedobjects.GetOrCreateByExternalIDOptions{ExternalID: "SN1", ExternalIDType: "c8y_Serial", Body: map[string]any{"name": "dev"}})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusDuplicate, result.Status)
	assert.Equal(t, "7", result.Data.ID())
	ss.mu.Lock()
	defer ss.mu.Unlock()
	assert.False(t, slices.ContainsFunc(ss.requests, func(r recorded) bool { return r.Method == http.MethodDelete }), "nothing may be deleted")
}

func Test_GetOrCreateByExternalID_FallbackOnOlderPlatform(t *testing.T) {
	ss := newScriptServer(t, func(r recorded, n int) (int, string) {
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.Path, "/identity/externalIds/"):
			return 404, `{"error":"identity/Not Found","message":"not found"}`
		case r.Method == http.MethodPost && r.Path == moPath:
			if strings.Contains(r.Body, "externalIds") {
				return 422, `{"error":"undefined/validationError","message":"Unknown field externalIds"}`
			}
			return 201, `{"id":"42","name":"dev"}`
		case r.Method == http.MethodPost && r.Path == "/identity/globalIds/42/externalIds":
			return 201, `{"externalId":"SN1","type":"c8y_Serial","managedObject":{"id":"42"}}`
		}
		return 500, `{"error":"unexpected"}`
	})
	result := newScriptClient(ss).ManagedObjects.GetOrCreateByExternalID(context.Background(),
		managedobjects.GetOrCreateByExternalIDOptions{ExternalID: "SN1", ExternalIDType: "c8y_Serial", Body: map[string]any{"name": "dev"}})
	require.NoError(t, result.Err)
	assert.Equal(t, op.StatusCreated, result.Status)
	assert.Len(t, ss.matching(http.MethodPost, moPath), 2, "first with externalIds, then without")
	assert.Len(t, ss.matching(http.MethodPost, "/identity/globalIds/42/externalIds"), 1)
}
