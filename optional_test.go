package compliancetest

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
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

func TestHooks_InstallFlagVariants(t *testing.T) {
	requireCapability(t, "hooks", agentInfo.Capabilities.Hooks)
	r := newTestRunner(t)
	ctx := testCtx(t)

	res := r.Run(ctx, nil, "install-hooks", "--force")
	requireSuccess(t, res)

	res = r.Run(ctx, nil, "uninstall-hooks")
	requireSuccess(t, res)

	res = r.Run(ctx, nil, "install-hooks", "--local-dev")
	requireSuccess(t, res)
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
	requireCapability(t, "hooks", agentInfo.Capabilities.Hooks)
	if len(agentInfo.HookNames) == 0 {
		t.Skip("agent declares hooks capability but no hook_names")
	}

	r := newTestRunner(t)
	ctx := testCtx(t)

	for _, hookName := range agentInfo.HookNames {
		t.Run(hookName, func(t *testing.T) {
			res := r.Run(ctx, nil, "parse-hook", "--hook", hookName)
			requireSuccess(t, res)

			stdout := strings.TrimSpace(string(res.Stdout))
			if stdout == "null" || stdout == "" {
				return
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

func TestTranscriptAnalyzer_SemanticFixture(t *testing.T) {
	requireCapability(t, "transcript_analyzer", agentInfo.Capabilities.TranscriptAnalyzer)
	if complianceFixtures == nil || complianceFixtures.TranscriptAnalyzer == nil {
		t.Skip("no transcript_analyzer fixture configured (set AGENT_FIXTURES)")
	}

	fixture := *complianceFixtures.TranscriptAnalyzer
	path, err := resolveFixturePath(fixture.SessionRef)
	if err != nil {
		t.Fatalf("resolve transcript fixture path: %v", err)
	}

	r := newTestRunner(t)
	ctx := testCtx(t)

	if fixture.Position != nil {
		res := r.Run(ctx, nil, "get-transcript-position", "--path", path)
		requireSuccess(t, res)

		var resp TranscriptPositionResponse
		requireUnmarshal(t, res.Stdout, &resp)
		if resp.Position != *fixture.Position {
			t.Errorf("position: got %d, want %d", resp.Position, *fixture.Position)
		}
	}

	if fixture.ModifiedFiles != nil {
		res := r.Run(ctx, nil, "extract-modified-files", "--path", path, "--offset", intArg(fixture.Offset))
		requireSuccess(t, res)

		var resp ExtractFilesResponse
		requireUnmarshal(t, res.Stdout, &resp)
		if !sameStrings(resp.Files, fixture.ModifiedFiles) {
			t.Errorf("modified_files: got %v, want %v", resp.Files, fixture.ModifiedFiles)
		}
	}

	if fixture.Prompts != nil {
		res := r.Run(ctx, nil, "extract-prompts", "--session-ref", path, "--offset", intArg(fixture.Offset))
		requireSuccess(t, res)

		var resp ExtractPromptsResponse
		requireUnmarshal(t, res.Stdout, &resp)
		if !slices.Equal(resp.Prompts, fixture.Prompts) {
			t.Errorf("prompts: got %v, want %v", resp.Prompts, fixture.Prompts)
		}
	}

	if fixture.HasSummary != nil {
		res := r.Run(ctx, nil, "extract-summary", "--session-ref", path)
		requireSuccess(t, res)

		var resp ExtractSummaryResponse
		requireUnmarshal(t, res.Stdout, &resp)
		if resp.HasSummary != *fixture.HasSummary {
			t.Errorf("has_summary: got %v, want %v", resp.HasSummary, *fixture.HasSummary)
		}
		if resp.Summary != fixture.Summary {
			t.Errorf("summary: got %q, want %q", resp.Summary, fixture.Summary)
		}
	}
}

func TestTranscriptPreparer_PrepareTranscript(t *testing.T) {
	requireCapability(t, "transcript_preparer", agentInfo.Capabilities.TranscriptPreparer)
	r := newTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "prepare-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(testCtx(t), nil, "prepare-transcript", "--session-ref", refPath)
	requireSuccess(t, res)
}

func TestTokenCalculator_CalculateTokens(t *testing.T) {
	requireCapability(t, "token_calculator", agentInfo.Capabilities.TokenCalculator)
	r := newTestRunner(t)

	res := r.Run(testCtx(t), []byte(`{}`), "calculate-tokens", "--offset", "0")
	requireSuccess(t, res)

	var resp TokenUsageResponse
	requireUnmarshal(t, res.Stdout, &resp)
	requireNonNegativeTokenUsage(t, &resp)
}

func TestTextGenerator_GenerateText(t *testing.T) {
	requireCapability(t, "text_generator", agentInfo.Capabilities.TextGenerator)
	r := newTestRunner(t)

	res := r.Run(testCtx(t), []byte("Reply with a short sentence."), "generate-text", "--model", "compliance-test")
	requireSuccess(t, res)

	var resp GenerateTextResponse
	requireUnmarshal(t, res.Stdout, &resp)
	if strings.TrimSpace(resp.Text) == "" {
		t.Error("generate-text returned empty text")
	}
}

func TestHookResponseWriter_WriteHookResponse(t *testing.T) {
	requireCapability(t, "hook_response_writer", agentInfo.Capabilities.HookResponseWriter)
	r := newTestRunner(t)

	res := r.Run(testCtx(t), nil, "write-hook-response", "--message", "compliance test response")
	requireSuccess(t, res)
}

func TestSubagentAwareExtractor_ExtractAllModifiedFiles(t *testing.T) {
	requireCapability(t, "subagent_aware_extractor", agentInfo.Capabilities.SubagentAwareExtractor)
	r := newTestRunner(t)
	subagentsDir := filepath.Join(r.RepoRoot(), "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("creating subagents dir: %v", err)
	}

	res := r.Run(testCtx(t), []byte(`{}`), "extract-all-modified-files",
		"--offset", "0",
		"--subagents-dir", subagentsDir)
	requireSuccess(t, res)

	var resp ExtractFilesResponse
	requireUnmarshal(t, res.Stdout, &resp)
}

func TestSubagentAwareExtractor_CalculateTotalTokens(t *testing.T) {
	requireCapability(t, "subagent_aware_extractor", agentInfo.Capabilities.SubagentAwareExtractor)
	r := newTestRunner(t)
	subagentsDir := filepath.Join(r.RepoRoot(), "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("creating subagents dir: %v", err)
	}

	res := r.Run(testCtx(t), []byte(`{}`), "calculate-total-tokens",
		"--offset", "0",
		"--subagents-dir", subagentsDir)
	requireSuccess(t, res)

	var resp TokenUsageResponse
	requireUnmarshal(t, res.Stdout, &resp)
	requireNonNegativeTokenUsage(t, &resp)
}

func TestDetect_WithFixtures(t *testing.T) {
	if complianceFixtures == nil || complianceFixtures.Detect == nil {
		t.Skip("no detect fixture configured (set AGENT_FIXTURES)")
	}

	fixture := *complianceFixtures.Detect
	tests := []struct {
		name     string
		repoPath string
		want     bool
	}{
		{name: "present", repoPath: fixture.PresentRepo, want: true},
		{name: "absent", repoPath: fixture.AbsentRepo, want: false},
	}

	for _, tt := range tests {
		if tt.repoPath == "" {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			path, err := resolveFixturePath(tt.repoPath)
			if err != nil {
				t.Fatalf("resolve fixture path: %v", err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat fixture path %s: %v", path, err)
			}
			if !info.IsDir() {
				t.Fatalf("fixture path must be a directory: %s", path)
			}

			r := NewRunnerWithEnvRoot(binaryPath, path, t.TempDir())
			res := r.Run(testCtx(t), nil, "detect")
			requireSuccess(t, res)

			var resp DetectResponse
			requireUnmarshal(t, res.Stdout, &resp)
			if resp.Present != tt.want {
				t.Errorf("present: got %v, want %v", resp.Present, tt.want)
			}
		})
	}
}

func intArg(v int) string {
	return strconv.Itoa(v)
}

func sameStrings(got, want []string) bool {
	gotCopy := append([]string(nil), got...)
	wantCopy := append([]string(nil), want...)
	sort.Strings(gotCopy)
	sort.Strings(wantCopy)
	return slices.Equal(gotCopy, wantCopy)
}

func requireNonNegativeTokenUsage(t *testing.T, usage *TokenUsageResponse) {
	t.Helper()
	if usage == nil {
		t.Fatal("token usage response must not be nil")
	}
	fields := map[string]int{
		"input_tokens":          usage.InputTokens,
		"cache_creation_tokens": usage.CacheCreationTokens,
		"cache_read_tokens":     usage.CacheReadTokens,
		"output_tokens":         usage.OutputTokens,
		"api_call_count":        usage.APICallCount,
	}
	for name, value := range fields {
		if value < 0 {
			t.Errorf("%s must be non-negative, got %d", name, value)
		}
	}
	if usage.SubagentTokens != nil {
		requireNonNegativeTokenUsage(t, usage.SubagentTokens)
	}
}
