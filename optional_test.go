package compliancetest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- hooks capability ---

func TestHooks_InstallAndUninstall(t *testing.T) {
	requireCapability(t, "hooks", agentInfo.Capabilities.Hooks)
	r := newTestRunner(t)
	ctx := testCtx(t)

	// Install.
	res := r.Run(ctx, nil, "install-hooks")
	requireSuccess(t, res)
	var installed HooksInstalledResponse
	requireUnmarshal(t, res.Stdout, &installed)
	if installed.HooksInstalled < 0 {
		t.Errorf("hooks_installed must be non-negative, got %d", installed.HooksInstalled)
	}

	// Verify status reflects what was installed.
	res = r.Run(ctx, nil, "are-hooks-installed")
	requireSuccess(t, res)
	var status AreHooksInstalledResponse
	requireUnmarshal(t, res.Stdout, &status)
	if installed.HooksInstalled > 0 && !status.Installed {
		t.Error("expected installed=true after install-hooks reported hooks installed")
	}

	// Uninstall.
	res = r.Run(ctx, nil, "uninstall-hooks")
	requireSuccess(t, res)

	// Verify uninstalled.
	res = r.Run(ctx, nil, "are-hooks-installed")
	requireSuccess(t, res)
	var after AreHooksInstalledResponse
	requireUnmarshal(t, res.Stdout, &after)
	if after.Installed {
		t.Error("expected installed=false after uninstall-hooks")
	}
}

func TestHooks_InstallIdempotent(t *testing.T) {
	requireCapability(t, "hooks", agentInfo.Capabilities.Hooks)
	r := newTestRunner(t)
	ctx := testCtx(t)

	res := r.Run(ctx, nil, "install-hooks")
	requireSuccess(t, res)

	// Second install should succeed with 0 new hooks.
	res = r.Run(ctx, nil, "install-hooks")
	requireSuccess(t, res)
	var installed HooksInstalledResponse
	requireUnmarshal(t, res.Stdout, &installed)
	if installed.HooksInstalled != 0 {
		t.Errorf("idempotent install: hooks_installed = %d, want 0", installed.HooksInstalled)
	}
}

func TestHooks_Parse(t *testing.T) {
	requireCapability(t, "hooks", agentInfo.Capabilities.Hooks)
	if len(agentInfo.HookNames) == 0 {
		t.Skip("agent declares hooks capability but no hook_names")
	}

	r := newTestRunner(t)
	ctx := testCtx(t)

	for _, hookName := range agentInfo.HookNames {
		t.Run(hookName, func(t *testing.T) {
			input := hookInputWithPrompt(hookName, "test-session-hook", "test prompt")
			res := r.Run(ctx, mustMarshal(t, input), "parse-hook", "--hook", hookName)
			requireSuccess(t, res)

			stdout := strings.TrimSpace(string(res.Stdout))
			if stdout == "null" || stdout == "" {
				return // null means no lifecycle significance, which is valid.
			}

			var event EventJSON
			requireUnmarshal(t, res.Stdout, &event)
			if event.Type < 1 {
				t.Errorf("event type must be >= 1, got %d", event.Type)
			}
		})
	}
}

// --- transcript_analyzer capability ---

func TestTranscriptAnalyzer_Position(t *testing.T) {
	requireCapability(t, "transcript_analyzer", agentInfo.Capabilities.TranscriptAnalyzer)
	r := newTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "test-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(testCtx(t), nil, "get-transcript-position", "--path", refPath)
	requireSuccess(t, res)

	var resp TranscriptPositionResponse
	requireUnmarshal(t, res.Stdout, &resp)
	if resp.Position < 0 {
		t.Errorf("position must be non-negative, got %d", resp.Position)
	}
}

func TestTranscriptAnalyzer_PositionMissingFile(t *testing.T) {
	requireCapability(t, "transcript_analyzer", agentInfo.Capabilities.TranscriptAnalyzer)
	r := newTestRunner(t)

	res := r.Run(testCtx(t), nil, "get-transcript-position",
		"--path", filepath.Join(r.RepoRoot(), "nonexistent.json"))
	requireSuccess(t, res)

	var resp TranscriptPositionResponse
	requireUnmarshal(t, res.Stdout, &resp)
	if resp.Position != 0 {
		t.Errorf("position for missing file: got %d, want 0", resp.Position)
	}
}

func TestTranscriptAnalyzer_ExtractModifiedFiles(t *testing.T) {
	requireCapability(t, "transcript_analyzer", agentInfo.Capabilities.TranscriptAnalyzer)
	r := newTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "test-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(testCtx(t), nil, "extract-modified-files", "--path", refPath, "--offset", "0")
	requireSuccess(t, res)

	var resp ExtractFilesResponse
	requireUnmarshal(t, res.Stdout, &resp)
	if resp.CurrentPosition < 0 {
		t.Errorf("current_position must be non-negative, got %d", resp.CurrentPosition)
	}
}

func TestTranscriptAnalyzer_ExtractPrompts(t *testing.T) {
	requireCapability(t, "transcript_analyzer", agentInfo.Capabilities.TranscriptAnalyzer)
	r := newTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "test-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(testCtx(t), nil, "extract-prompts", "--session-ref", refPath, "--offset", "0")
	requireSuccess(t, res)

	var resp ExtractPromptsResponse
	requireUnmarshal(t, res.Stdout, &resp)
	// Empty prompts array is valid for an empty/minimal transcript.
}

func TestTranscriptAnalyzer_ExtractSummary(t *testing.T) {
	requireCapability(t, "transcript_analyzer", agentInfo.Capabilities.TranscriptAnalyzer)
	r := newTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "test-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(testCtx(t), nil, "extract-summary", "--session-ref", refPath)
	requireSuccess(t, res)

	var resp ExtractSummaryResponse
	requireUnmarshal(t, res.Stdout, &resp)
	// has_summary=false is expected for an empty/minimal transcript.
}
