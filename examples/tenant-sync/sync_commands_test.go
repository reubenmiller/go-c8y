package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestSyncer(t *testing.T) (*Syncer, string) {
	t.Helper()
	dir := t.TempDir()
	return &Syncer{
		Resolver: NewSourceResolver(dir, t.TempDir(), false),
	}, dir
}

func readLines(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	return strings.Split(strings.TrimSpace(string(raw)), "\n")
}

func TestSyncCommandsRunsActionsInSequence(t *testing.T) {
	syncer, dir := newTestSyncer(t)

	err := syncer.SyncCommands(context.Background(), []CommandGroupSpec{
		{Name: "ordered", Actions: []string{
			"echo a >> out.txt",
			"echo b >> out.txt",
			"echo c >> out.txt",
		}},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"a", "b", "c"}, readLines(t, filepath.Join(dir, "out.txt")))
	require.Len(t, syncer.Results, 3)
	for _, result := range syncer.Results {
		assert.Equal(t, ActionExecuted, result.Action)
		assert.Equal(t, SectionCommands, result.Section)
	}
}

func TestSyncCommandsFailureSkipsRestOfGroup(t *testing.T) {
	syncer, dir := newTestSyncer(t)

	err := syncer.SyncCommands(context.Background(), []CommandGroupSpec{
		{Name: "broken", Actions: []string{
			"echo a >> out.txt",
			"exit 1",
			"echo c >> out.txt",
		}},
	})
	require.NoError(t, err)

	assert.Equal(t, []string{"a"}, readLines(t, filepath.Join(dir, "out.txt")))
	require.Len(t, syncer.Results, 3)
	assert.Equal(t, ActionExecuted, syncer.Results[0].Action)
	assert.Equal(t, ActionFailed, syncer.Results[1].Action)
	assert.Equal(t, ActionSkipped, syncer.Results[2].Action)
	assert.Contains(t, syncer.Results[2].Detail, "previous action failed")
}

func TestSyncCommandsGroupsRunConcurrently(t *testing.T) {
	syncer, _ := newTestSyncer(t)

	// Each group signals the other and then waits for the other's signal:
	// this only completes when both groups run at the same time
	waitFor := func(file string) string {
		return "for i in $(seq 1 100); do [ -f " + file + " ] && exit 0; sleep 0.1; done; exit 1"
	}
	err := syncer.SyncCommands(context.Background(), []CommandGroupSpec{
		{Name: "one", Actions: []string{"touch one.flag", waitFor("two.flag")}},
		{Name: "two", Actions: []string{"touch two.flag", waitFor("one.flag")}},
	})
	require.NoError(t, err)

	require.Len(t, syncer.Results, 4)
	for _, result := range syncer.Results {
		assert.Equal(t, ActionExecuted, result.Action, "item %s should have executed", result.Item)
	}
}

func TestSyncCommandsDryRun(t *testing.T) {
	syncer, dir := newTestSyncer(t)
	syncer.DryRun = true

	err := syncer.SyncCommands(context.Background(), []CommandGroupSpec{
		{Name: "group", Actions: []string{"echo a >> out.txt"}},
	})
	require.NoError(t, err)

	assert.NoFileExists(t, filepath.Join(dir, "out.txt"))
	require.Len(t, syncer.Results, 1)
	assert.Equal(t, ActionPlanned, syncer.Results[0].Action)
}

func TestApplyRunsSectionHooksAroundSection(t *testing.T) {
	syncer, dir := newTestSyncer(t)

	manifest := &Manifest{
		Commands: []CommandGroupSpec{
			{Name: "work", Actions: []string{"echo section >> out.txt"}},
		},
		Hooks: HooksSpec{
			Pre:  []HookSpec{{Run: "echo global-pre >> out.txt"}},
			Post: []HookSpec{{Run: "echo global-post >> out.txt"}},
			Sections: map[string]SectionHooksSpec{
				SectionCommands: {
					Pre:  []HookSpec{{Run: "echo section-pre >> out.txt"}},
					Post: []HookSpec{{Run: "echo section-post >> out.txt"}},
				},
				// Hooks of sections excluded by --only must not run
				SectionSoftware: {
					Pre: []HookSpec{{Run: "echo software-pre >> out.txt"}},
				},
			},
		},
	}

	err := syncer.Apply(context.Background(), manifest, []string{SectionCommands})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"global-pre",
		"section-pre",
		"section",
		"section-post",
		"global-post",
	}, readLines(t, filepath.Join(dir, "out.txt")))
}

func TestApplyFailingSectionPreHookAbortsRun(t *testing.T) {
	syncer, dir := newTestSyncer(t)

	manifest := &Manifest{
		Commands: []CommandGroupSpec{
			{Name: "work", Actions: []string{"echo section >> out.txt"}},
		},
		Hooks: HooksSpec{
			Sections: map[string]SectionHooksSpec{
				SectionCommands: {
					Pre: []HookSpec{{Run: "exit 1"}},
				},
			},
		},
	}

	err := syncer.Apply(context.Background(), manifest, []string{SectionCommands})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "sections.commands.pre hook failed")
	assert.NoFileExists(t, filepath.Join(dir, "out.txt"))
}
