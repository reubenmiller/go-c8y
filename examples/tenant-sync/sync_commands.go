package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"
)

// SyncCommands executes the command groups of the manifest. The groups run
// concurrently with each other; the actions within a group run in sequence,
// and a failing action skips the rest of its group. Commands execute via
// 'sh -c' from the manifest directory with the same environment as hooks
// (including the target tenant's C8Y_* credentials in multi-tenant runs).
func (s *Syncer) SyncCommands(ctx context.Context, groups []CommandGroupSpec) error {
	if len(groups) == 0 {
		return nil
	}

	if s.DryRun {
		for _, group := range groups {
			for i, action := range group.Actions {
				s.record(SectionCommands, commandItem(group, i), ActionPlanned, action, nil)
			}
		}
		return nil
	}

	var wg sync.WaitGroup
	for _, group := range groups {
		wg.Add(1)
		go func(group CommandGroupSpec) {
			defer wg.Done()
			s.runCommandGroup(ctx, group)
		}(group)
	}
	wg.Wait()
	return nil
}

// runCommandGroup executes the actions of one group in sequence
func (s *Syncer) runCommandGroup(ctx context.Context, group CommandGroupSpec) {
	for i, action := range group.Actions {
		cmd := exec.CommandContext(ctx, "sh", "-c", action)
		cmd.Dir = s.Resolver.BaseDir
		cmd.Env = s.hookEnv()

		output, err := cmd.CombinedOutput()
		if err != nil {
			s.record(SectionCommands, commandItem(group, i), ActionFailed, action,
				fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output))))
			for j := i + 1; j < len(group.Actions); j++ {
				s.record(SectionCommands, commandItem(group, j), ActionSkipped,
					group.Actions[j]+" (previous action failed)", nil)
			}
			return
		}

		// Record and print the output under one lock so the lines of
		// concurrent groups don't interleave
		s.mu.Lock()
		s.recordLocked(SectionCommands, commandItem(group, i), ActionExecuted, action, nil)
		if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
			for line := range strings.SplitSeq(trimmed, "\n") {
				fmt.Printf("      %s\n", line)
			}
		}
		s.mu.Unlock()
	}
}

func commandItem(group CommandGroupSpec, index int) string {
	return fmt.Sprintf("%s[%d]", group.Name, index)
}
