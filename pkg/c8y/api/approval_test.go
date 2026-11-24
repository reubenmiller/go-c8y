package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/events"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// approvalServer records every request that reaches the server.
type approvalServer struct {
	mu       sync.Mutex
	requests []string
	server   *httptest.Server
}

func newApprovalServer(t *testing.T) *approvalServer {
	t.Helper()
	s := &approvalServer{}
	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.requests = append(s.requests, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/inventory/managedObjects"):
			// name lookup
			_, _ = w.Write([]byte(`{"managedObjects":[{"id":"4242","name":"dev01"}],"statistics":{"pageSize":5,"currentPage":1}}`))
		case r.Method == http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"1","type":"t","text":"x","source":{"id":"4242"}}`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(s.server.Close)
	return s
}

func (s *approvalServer) sent() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.requests...)
}

func (s *approvalServer) client() *Client {
	return NewClient(ClientOptions{
		BaseURL:          s.server.URL,
		SkipTenantLookup: true,
		Auth: authentication.AuthOptions{
			Tenant:   "t100",
			Username: "user",
			Password: "pass",
			AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
		},
	})
}

func Test_RequestApprover_DeclineDoesNotSend(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()

	var seen []string
	ctx := WithRequestApprover(context.Background(), func(req *http.Request) (bool, error) {
		seen = append(seen, req.Method+" "+req.URL.Path)
		return false, nil
	})

	res := client.Events.CreateRaw(ctx, map[string]any{"type": "t", "text": "x", "source": map[string]any{"id": "4242"}})
	require.Error(t, res.Err)
	assert.True(t, errors.Is(res.Err, ErrRequestNotApproved), "got %v", res.Err)
	assert.Equal(t, []string{"POST /event/events"}, seen)
	assert.Empty(t, srv.sent(), "a declined request must not reach the server")
}

func Test_RequestApprover_ApproveSendsAndSkipsReads(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()

	calls := 0
	ctx := context.Background()

	// The name lookup (GET) must not be offered for approval; the POST must
	// carry the resolved id when it is.
	var approved *http.Request
	ctx = WithRequestApprover(ctx, func(req *http.Request) (bool, error) {
		calls++
		approved = req
		return true, nil
	})
	res := client.Events.Create(ctx, events.CreateOptions{Source: "name:dev01", Type: "t", Text: "x"})
	require.NoError(t, res.Err)
	assert.Equal(t, 1, calls)
	require.NotNil(t, approved)
	assert.Equal(t, http.MethodPost, approved.Method)

	sent := srv.sent()
	require.Len(t, sent, 2)
	assert.True(t, strings.HasPrefix(sent[0], "GET /inventory/managedObjects"), sent[0])
	assert.True(t, strings.HasPrefix(sent[1], "POST /event/events"), sent[1])
}

func Test_RequestApprover_ErrorIsReturned(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()
	abort := errors.New("aborted by user")
	ctx := WithRequestApprover(context.Background(), func(*http.Request) (bool, error) { return false, abort })
	res := client.Events.Delete(ctx, "1")
	require.Error(t, res.Err)
	assert.True(t, errors.Is(res.Err, abort), "got %v", res.Err)
	assert.Empty(t, srv.sent())
}

func Test_RequestApprover_ScopeAsksOnce(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()
	calls := 0
	ctx := WithRequestApprover(context.Background(), func(*http.Request) (bool, error) {
		calls++
		return true, nil
	})
	ctx = WithApprovalScope(ctx)
	require.NoError(t, client.Events.Delete(ctx, "1").Err)
	require.NoError(t, client.Events.Delete(ctx, "2").Err)
	assert.Equal(t, 1, calls)
	assert.Len(t, srv.sent(), 2)

	// A new scope asks again.
	require.NoError(t, client.Events.Delete(WithApprovalScope(ctx), "3").Err)
	assert.Equal(t, 2, calls)
}

func Test_RequestApprover_NotCalledForDryRun(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()
	ctx := WithRequestApprover(context.Background(), func(*http.Request) (bool, error) {
		t.Fatal("approver must not be called for a dry run")
		return false, nil
	})
	ctx = WithMockResponses(WithDryRun(ctx, true), false)
	require.NoError(t, client.Events.Delete(ctx, "1").Err)
	assert.Empty(t, srv.sent())
}

func Test_RequestApprover_DeferredAsksOnExecute(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()
	calls := 0
	ctx := WithRequestApprover(context.Background(), func(*http.Request) (bool, error) {
		calls++
		return true, nil
	})
	prepared := client.Events.Delete(WithDeferredExecution(ctx, true), "1")
	require.True(t, prepared.IsDeferred())
	assert.Equal(t, 0, calls, "preparing must not ask")
	assert.Empty(t, srv.sent())
	require.NoError(t, prepared.Execute(ctx).Err)
	assert.Equal(t, 1, calls)
	assert.Len(t, srv.sent(), 1)
}

func Test_WithDryRun_MockOrderIndependent(t *testing.T) {
	a := WithMockResponses(WithDryRun(context.Background(), true), false)
	b := WithDryRun(WithMockResponses(context.Background(), false), true)
	c := WithDryRun(context.Background(), true)
	assert.False(t, IsMockResponses(a))
	assert.False(t, IsMockResponses(b), "an explicit WithMockResponses(false) must survive a later WithDryRun")
	assert.True(t, IsMockResponses(c), "dry run alone still enables mocks (backward compatible)")
}

func Test_ResolutionContext_KeepsRequestOptions(t *testing.T) {
	ctx := WithTenant(context.Background(), "t555")
	ctx = WithExtraHeaders(ctx, map[string]string{"X-Test": "1"})
	ctx = WithProcessingModeTransient(ctx)
	ctx = WithRequestApprover(ctx, func(*http.Request) (bool, error) { return true, nil })

	for name, in := range map[string]context.Context{
		"deferred": WithDeferredExecution(ctx, true),
		"dryRun":   WithDryRun(ctx, true),
	} {
		t.Run(name, func(t *testing.T) {
			rc := ResolutionContext(in)
			assert.False(t, IsDryRun(rc))
			assert.False(t, IsDeferredExecution(rc))
			assert.False(t, IsMockResponses(rc))
			assert.Nil(t, ctxhelpers.GetRequestApprover(rc))
			tenant, _ := ctxhelpers.TenantFromContext(rc)
			assert.Equal(t, "t555", tenant, "tenant must be kept")
			assert.Equal(t, map[string]string{"X-Test": "1"}, ExtraHeaders(rc))
			assert.NotEmpty(t, ProcessingMode(rc))
		})
	}

	explicit := WithMockResponses(WithDryRun(ctx, true), true)
	assert.True(t, IsMockResponses(ResolutionContext(explicit)), "explicit mocks are kept for tests")
}
