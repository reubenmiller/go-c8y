package agents_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/internal/pkg/examplesetup"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/agents"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The inventory query is sent as `q` (not `query`), like go-c8y-cli does.
func TestList_SendsQueryAsQ(t *testing.T) {
	client, stop := examplesetup.New()
	defer stop()

	var got *http.Request
	ctx := api.WithDryRun(context.Background(), true)
	ctx = api.WithDryRunHandler(ctx, func(r *http.Request) { got = r })

	result := client.Agents.List(ctx, agents.ListOptions{Query: agents.ScopeToAgents("(name eq 'a')")})
	require.NoError(t, result.Err)
	require.NotNil(t, got)
	assert.Equal(t, agents.ScopeToAgents("(name eq 'a')"), got.URL.Query().Get("q"))
	assert.False(t, got.URL.Query().Has("query"))

	got = nil
	require.NoError(t, client.Agents.List(ctx, agents.ListOptions{}).Err)
	assert.False(t, got.URL.Query().Has("q"), "no empty q parameter")
}
