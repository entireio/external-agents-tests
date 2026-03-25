package hooks_test

import (
	"strings"
	"testing"
	"time"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
)

func TestHooks_InstallAndUninstall(t *testing.T) {
	harness.RequireCapability(t, "hooks", harness.AgentInfo.Capabilities.Hooks)
	r := harness.NewTestRunner(t)
	ctx := harness.TestCtx(t)

	// Install.
	res := r.Run(ctx, nil, "install-hooks")
	harness.RequireSuccess(t, res)
	var installed protocol.HooksInstalledResponse
	harness.RequireUnmarshal(t, res.Stdout, &installed)
	if installed.HooksInstalled < 0 {
		t.Errorf("hooks_installed must be non-negative, got %d", installed.HooksInstalled)
	}

	// Verify status reflects what was installed.
	res = r.Run(ctx, nil, "are-hooks-installed")
	harness.RequireSuccess(t, res)
	var status protocol.AreHooksInstalledResponse
	harness.RequireUnmarshal(t, res.Stdout, &status)
	if installed.HooksInstalled > 0 && !status.Installed {
		t.Error("expected installed=true after install-hooks reported hooks installed")
	}

	// Uninstall.
	res = r.Run(ctx, nil, "uninstall-hooks")
	harness.RequireSuccess(t, res)

	// Verify uninstalled.
	res = r.Run(ctx, nil, "are-hooks-installed")
	harness.RequireSuccess(t, res)
	var after protocol.AreHooksInstalledResponse
	harness.RequireUnmarshal(t, res.Stdout, &after)
	if after.Installed {
		t.Error("expected installed=false after uninstall-hooks")
	}
}

func TestHooks_InstallIdempotent(t *testing.T) {
	harness.RequireCapability(t, "hooks", harness.AgentInfo.Capabilities.Hooks)
	r := harness.NewTestRunner(t)
	ctx := harness.TestCtx(t)

	res := r.Run(ctx, nil, "install-hooks")
	harness.RequireSuccess(t, res)

	// Second install should succeed with 0 new hooks.
	res = r.Run(ctx, nil, "install-hooks")
	harness.RequireSuccess(t, res)
	var installed protocol.HooksInstalledResponse
	harness.RequireUnmarshal(t, res.Stdout, &installed)
	if installed.HooksInstalled != 0 {
		t.Errorf("idempotent install: hooks_installed = %d, want 0", installed.HooksInstalled)
	}
}

func TestHooks_InstallFlagVariants(t *testing.T) {
	harness.RequireCapability(t, "hooks", harness.AgentInfo.Capabilities.Hooks)
	r := harness.NewTestRunner(t)
	ctx := harness.TestCtx(t)

	res := r.Run(ctx, nil, "install-hooks", "--force")
	harness.RequireSuccess(t, res)

	res = r.Run(ctx, nil, "uninstall-hooks")
	harness.RequireSuccess(t, res)

	res = r.Run(ctx, nil, "install-hooks", "--local-dev")
	harness.RequireSuccess(t, res)
}

func TestHooks_Parse(t *testing.T) {
	harness.RequireCapability(t, "hooks", harness.AgentInfo.Capabilities.Hooks)
	if len(harness.AgentInfo.HookNames) == 0 {
		t.Skip("agent declares hooks capability but no hook_names")
	}

	r := harness.NewTestRunner(t)
	ctx := harness.TestCtx(t)

	for _, hookName := range harness.AgentInfo.HookNames {
		t.Run(hookName, func(t *testing.T) {
			input := harness.HookInputWithPrompt(hookName, "test-session-hook", "test prompt")
			res := r.Run(ctx, harness.MustMarshal(t, input), "parse-hook", "--hook", hookName)
			harness.RequireSuccess(t, res)

			stdout := strings.TrimSpace(string(res.Stdout))
			if stdout == "null" || stdout == "" {
				return // null means no lifecycle significance, which is valid.
			}

			var event protocol.EventJSON
			harness.RequireUnmarshal(t, res.Stdout, &event)
			if event.Type < 1 {
				t.Errorf("event type must be >= 1, got %d", event.Type)
			}
			if event.SessionID == "" {
				t.Error("non-null hook event must include session_id")
			}
			if event.Timestamp != "" {
				if _, err := time.Parse(time.RFC3339, event.Timestamp); err != nil {
					t.Errorf("timestamp must be RFC3339, got %q: %v", event.Timestamp, err)
				}
			}
		})
	}
}

func TestHooks_ParseEmptyInput(t *testing.T) {
	harness.RequireCapability(t, "hooks", harness.AgentInfo.Capabilities.Hooks)
	if len(harness.AgentInfo.HookNames) == 0 {
		t.Skip("agent declares hooks capability but no hook_names")
	}

	r := harness.NewTestRunner(t)
	ctx := harness.TestCtx(t)

	for _, hookName := range harness.AgentInfo.HookNames {
		t.Run(hookName, func(t *testing.T) {
			res := r.Run(ctx, nil, "parse-hook", "--hook", hookName)
			harness.RequireSuccess(t, res)

			stdout := strings.TrimSpace(string(res.Stdout))
			if stdout == "null" || stdout == "" {
				return
			}

			var event protocol.EventJSON
			harness.RequireUnmarshal(t, res.Stdout, &event)
			if event.Type < 1 {
				t.Errorf("event type must be >= 1, got %d", event.Type)
			}
		})
	}
}
