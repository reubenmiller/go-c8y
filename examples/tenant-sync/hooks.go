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

		cmd := exec.CommandContext(ctx, "sh", "-c", hook.Run) // NOSONAR
		cmd.Dir = s.Resolver.BaseDir
		cmd.Env = s.hookEnv()

		output, err := cmd.CombinedOutput()
		if err != nil {
			s.record("hooks", item, ActionFailed, hook.Run,
				fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output))))
			if stage == "pre" || strings.HasSuffix(stage, ".pre") {
				return fmt.Errorf("%s hook failed: %s", stage, hook.Run)
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

// hookEnv returns the environment for hook commands. When the active target
// uses its own credentials (multi-tenant runs), the C8Y_* session variables
// are overridden with the target tenant's credentials so hooks (e.g.
// go-c8y-cli calls) act on the same tenant as the sync sections.
func (s *Syncer) hookEnv() []string {
	env := os.Environ()
	if s.activeTarget == nil || s.activeTarget.Auth == nil {
		return env
	}
	auth := s.activeTarget.Auth
	// Later entries win for duplicate keys (os/exec uses the last value).
	// C8Y_TOKEN is cleared because a token from the base session would take
	// precedence over the basic credentials in most tools.
	return append(env,
		"C8Y_TENANT="+auth.Tenant,
		"C8Y_USERNAME="+auth.Username,
		"C8Y_USER="+auth.Username,
		"C8Y_PASSWORD="+auth.Password,
		"C8Y_TOKEN=",
	)
}
