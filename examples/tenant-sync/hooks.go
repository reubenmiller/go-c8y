package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runHooks executes the hooks of a stage ("pre" or "post") sequentially.
// Hooks run via 'sh -c' with the manifest directory as the working directory
// and the current environment (including C8Y_* session variables) passed
// through, so go-c8y-cli sessions work out of the box.
//
// A failing pre hook returns an error (aborting the run); post hook failures
// are recorded but do not return an error.
func (s *Syncer) runHooks(ctx context.Context, stage string, hooks []HookSpec) error {
	for index, hook := range hooks {
		item := hook.Name
		if item == "" {
			item = fmt.Sprintf("%s[%d]", stage, index)
		} else {
			item = fmt.Sprintf("%s: %s", stage, item)
		}

		if s.DryRun {
			s.record("hooks", item, ActionPlanned, hook.Run, nil)
			continue
		}

		cmd := exec.CommandContext(ctx, "sh", "-c", hook.Run)
		cmd.Dir = s.Resolver.BaseDir
		cmd.Env = os.Environ()

		output, err := cmd.CombinedOutput()
		if err != nil {
			s.record("hooks", item, ActionFailed, hook.Run,
				fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output))))
			if stage == "pre" {
				return fmt.Errorf("pre hook failed: %s", hook.Run)
			}
			continue
		}

		s.record("hooks", item, ActionExecuted, hook.Run, nil)
		if trimmed := strings.TrimSpace(string(output)); trimmed != "" {
			for _, line := range strings.Split(trimmed, "\n") {
				fmt.Printf("      %s\n", line)
			}
		}
	}
	return nil
}
