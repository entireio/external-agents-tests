package transcript_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
)

func TestTranscriptAnalyzer_Position(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "test-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(harness.TestCtx(t), nil, "get-transcript-position", "--path", refPath)
	harness.RequireSuccess(t, res)

	var resp protocol.TranscriptPositionResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	if resp.Position < 0 {
		t.Errorf("position must be non-negative, got %d", resp.Position)
	}
}

func TestTranscriptAnalyzer_PositionMissingFile(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)

	res := r.Run(harness.TestCtx(t), nil, "get-transcript-position",
		"--path", filepath.Join(r.RepoRoot(), "nonexistent.json"))
	harness.RequireSuccess(t, res)

	var resp protocol.TranscriptPositionResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	if resp.Position != 0 {
		t.Errorf("position for missing file: got %d, want 0", resp.Position)
	}
}

func TestTranscriptAnalyzer_ExtractModifiedFiles(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "test-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(harness.TestCtx(t), nil, "extract-modified-files", "--path", refPath, "--offset", "0")
	harness.RequireSuccess(t, res)

	var resp protocol.ExtractFilesResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	if resp.CurrentPosition < 0 {
		t.Errorf("current_position must be non-negative, got %d", resp.CurrentPosition)
	}
}

func TestTranscriptAnalyzer_ExtractPrompts(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "test-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(harness.TestCtx(t), nil, "extract-prompts", "--session-ref", refPath, "--offset", "0")
	harness.RequireSuccess(t, res)

	var resp protocol.ExtractPromptsResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	// Empty prompts array is valid for an empty/minimal transcript.
}

func TestTranscriptAnalyzer_ExtractSummary(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "test-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(harness.TestCtx(t), nil, "extract-summary", "--session-ref", refPath)
	harness.RequireSuccess(t, res)

	var resp protocol.ExtractSummaryResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	// has_summary=false is expected for an empty/minimal transcript.
}

func TestTranscriptAnalyzer_SemanticFixture(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	if harness.Fixtures == nil || harness.Fixtures.TranscriptAnalyzer == nil {
		t.Skip("no transcript_analyzer fixture configured (set AGENT_FIXTURES)")
	}

	fixture := *harness.Fixtures.TranscriptAnalyzer
	path, err := harness.ResolveFixturePath(fixture.SessionRef)
	if err != nil {
		t.Fatalf("resolve transcript fixture path: %v", err)
	}

	r := harness.NewTestRunner(t)
	ctx := harness.TestCtx(t)

	if fixture.Position != nil {
		res := r.Run(ctx, nil, "get-transcript-position", "--path", path)
		harness.RequireSuccess(t, res)

		var resp protocol.TranscriptPositionResponse
		harness.RequireUnmarshal(t, res.Stdout, &resp)
		if resp.Position != *fixture.Position {
			t.Errorf("position: got %d, want %d", resp.Position, *fixture.Position)
		}
	}

	if fixture.ModifiedFiles != nil {
		res := r.Run(ctx, nil, "extract-modified-files", "--path", path, "--offset", harness.IntArg(fixture.Offset))
		harness.RequireSuccess(t, res)

		var resp protocol.ExtractFilesResponse
		harness.RequireUnmarshal(t, res.Stdout, &resp)
		if !harness.SameStrings(resp.Files, fixture.ModifiedFiles) {
			t.Errorf("modified_files: got %v, want %v", resp.Files, fixture.ModifiedFiles)
		}
	}

	if fixture.Prompts != nil {
		res := r.Run(ctx, nil, "extract-prompts", "--session-ref", path, "--offset", harness.IntArg(fixture.Offset))
		harness.RequireSuccess(t, res)

		var resp protocol.ExtractPromptsResponse
		harness.RequireUnmarshal(t, res.Stdout, &resp)
		if !slices.Equal(resp.Prompts, fixture.Prompts) {
			t.Errorf("prompts: got %v, want %v", resp.Prompts, fixture.Prompts)
		}
	}

	if fixture.HasSummary != nil {
		res := r.Run(ctx, nil, "extract-summary", "--session-ref", path)
		harness.RequireSuccess(t, res)

		var resp protocol.ExtractSummaryResponse
		harness.RequireUnmarshal(t, res.Stdout, &resp)
		if resp.HasSummary != *fixture.HasSummary {
			t.Errorf("has_summary: got %v, want %v", resp.HasSummary, *fixture.HasSummary)
		}
		if resp.Summary != fixture.Summary {
			t.Errorf("summary: got %q, want %q", resp.Summary, fixture.Summary)
		}
	}
}
