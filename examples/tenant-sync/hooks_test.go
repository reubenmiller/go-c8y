package main

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHooks(t *testing.T) {
	syncer := &Syncer{
		Resolver: NewSourceResolver(t.TempDir(), "", false),
	}

	err := syncer.runHooks(context.Background(), "pre", []HookSpec{
		{Name: "ok", Run: "echo hello"},
	})
	require.NoError(t, err)
	require.Len(t, syncer.Results, 1)
	assert.Equal(t, ActionExecuted, syncer.Results[0].Action)

	// A failing pre hook aborts
	err = syncer.runHooks(context.Background(), "pre", []HookSpec{
		{Run: "exit 1"},
	})
	require.Error(t, err)

	// A failing post hook is recorded but does not abort
	syncer.Results = nil
	err = syncer.runHooks(context.Background(), "post", []HookSpec{
		{Run: "exit 1"},
		{Run: "echo still runs"},
	})
	require.NoError(t, err)
	require.Len(t, syncer.Results, 2)
	assert.Equal(t, ActionFailed, syncer.Results[0].Action)
	assert.Equal(t, ActionExecuted, syncer.Results[1].Action)
}

func TestRunHooksDryRun(t *testing.T) {
	syncer := &Syncer{
		Resolver: NewSourceResolver(t.TempDir(), "", true),
		DryRun:   true,
	}

	err := syncer.runHooks(context.Background(), "pre", []HookSpec{
		{Run: "exit 1"}, // not executed in dry-run
	})
	require.NoError(t, err)
	require.Len(t, syncer.Results, 1)
	assert.Equal(t, ActionPlanned, syncer.Results[0].Action)
}
