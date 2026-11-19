package api

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_CreateRaw_ResolvesBodyDeviceRef(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()

	res := client.Events.CreateRaw(context.Background(), json.RawMessage(`{"type":"t","text":"x","source":{"id":"dev01"}}`))
	require.NoError(t, res.Err)
	sent := srv.sent()
	require.Len(t, sent, 2)
	assert.True(t, strings.HasPrefix(sent[0], "GET /inventory/managedObjects"), sent[0])
	assert.Contains(t, sent[0], "dev01")
}

func Test_CreateRaw_IDNeedsNoLookup(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()
	body := map[string]any{"type": "t", "text": "x", "source": map[string]any{"id": "4242"}}
	require.NoError(t, client.Events.CreateRaw(context.Background(), body).Err)
	assert.Len(t, srv.sent(), 1, "an id must be sent as is, without a lookup")
}

func Test_CreateRaw_DryRunStillResolves(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()
	ctx := WithMockResponses(WithDryRun(context.Background(), true), false)
	require.NoError(t, client.Events.CreateRaw(ctx, json.RawMessage(`{"source":{"id":"name:dev01"}}`)).Err)
	sent := srv.sent()
	require.Len(t, sent, 1, "only the lookup is sent under dry run")
	assert.True(t, strings.HasPrefix(sent[0], "GET /inventory/managedObjects"), sent[0])
}

func Test_ListAll_ResolvesSourceOnce(t *testing.T) {
	srv := newApprovalServer(t)
	client := srv.client()
	ctx := WithMockResponses(WithDryRun(context.Background(), true), false)
	it := client.Events.ListAll(ctx, eventsListOptions("name:dev01"))
	for range it.Items() {
	}
	lookups := 0
	for _, s := range srv.sent() {
		if strings.HasPrefix(s, "GET /inventory/managedObjects") {
			lookups++
		}
	}
	assert.Equal(t, 1, lookups)
}

func eventsListOptions(source string) events.ListOptions {
	return events.ListOptions{Source: managedobjects.DeviceRef(source)}
}
