package transcript_test

import (
	"slices"
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
)

func TestTranscriptAnalyzer_Position(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)
	refPath := harness.WriteFixture(t, r.RepoRoot(), "test-transcript.json", []byte(`{}`))

	var resp protocol.TranscriptPositionResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "get-transcript-position", "--path", refPath)
	if resp.Position < 0 {
		t.Errorf("position must be non-negative, got %d", resp.Position)
	}
}

func TestTranscriptAnalyzer_PositionMissingFile(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)

	var resp protocol.TranscriptPositionResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "get-transcript-position",
		"--path", r.RepoRoot()+"/nonexistent.json")
	if resp.Position != 0 {
		t.Errorf("position for missing file: got %d, want 0", resp.Position)
	}
}

func TestTranscriptAnalyzer_ExtractModifiedFiles(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)
	refPath := harness.WriteFixture(t, r.RepoRoot(), "test-transcript.json", []byte(`{}`))

	var resp protocol.ExtractFilesResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "extract-modified-files", "--path", refPath, "--offset", "0")
	if resp.CurrentPosition < 0 {
		t.Errorf("current_position must be non-negative, got %d", resp.CurrentPosition)
	}
}

func TestTranscriptAnalyzer_ExtractModifiedFiles_WithOffset(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)
	refPath := harness.WriteFixture(t, r.RepoRoot(), "test-transcript.json", []byte(`{}`))

	// A non-zero offset past the end of an empty transcript should still succeed.
	var resp protocol.ExtractFilesResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "extract-modified-files", "--path", refPath, "--offset", "999")
	if len(resp.Files) != 0 {
		t.Errorf("expected no files when offset exceeds transcript length, got %v", resp.Files)
	}
}

func TestTranscriptAnalyzer_ExtractPrompts(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)
	refPath := harness.WriteFixture(t, r.RepoRoot(), "test-transcript.json", []byte(`{}`))

	var resp protocol.ExtractPromptsResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "extract-prompts", "--session-ref", refPath, "--offset", "0")
	// Empty prompts array is valid for an empty/minimal transcript.
}

func TestTranscriptAnalyzer_ExtractPrompts_WithOffset(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)
	refPath := harness.WriteFixture(t, r.RepoRoot(), "test-transcript.json", []byte(`{}`))

	// A non-zero offset past the end of an empty transcript should still succeed.
	var resp protocol.ExtractPromptsResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "extract-prompts", "--session-ref", refPath, "--offset", "999")
	if len(resp.Prompts) != 0 {
		t.Errorf("expected no prompts when offset exceeds transcript length, got %v", resp.Prompts)
	}
}

func TestTranscriptAnalyzer_ExtractSummary(t *testing.T) {
	harness.RequireCapability(t, "transcript_analyzer", harness.AgentInfo.Capabilities.TranscriptAnalyzer)
	r := harness.NewTestRunner(t)
	refPath := harness.WriteFixture(t, r.RepoRoot(), "test-transcript.json", []byte(`{}`))

	var resp protocol.ExtractSummaryResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "extract-summary", "--session-ref", refPath)
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
		var resp protocol.TranscriptPositionResponse
		harness.RunAndUnmarshal(t, r, ctx, &resp, nil, "get-transcript-position", "--path", path)
		if resp.Position != *fixture.Position {
			t.Errorf("position: got %d, want %d", resp.Position, *fixture.Position)
		}
	}

	if fixture.ModifiedFiles != nil {
		var resp protocol.ExtractFilesResponse
		harness.RunAndUnmarshal(t, r, ctx, &resp, nil, "extract-modified-files", "--path", path, "--offset", harness.IntArg(fixture.Offset))
		if !harness.SameStrings(resp.Files, fixture.ModifiedFiles) {
			t.Errorf("modified_files: got %v, want %v", resp.Files, fixture.ModifiedFiles)
		}
	}

	if fixture.Prompts != nil {
		var resp protocol.ExtractPromptsResponse
		harness.RunAndUnmarshal(t, r, ctx, &resp, nil, "extract-prompts", "--session-ref", path, "--offset", harness.IntArg(fixture.Offset))
		if !slices.Equal(resp.Prompts, fixture.Prompts) {
			t.Errorf("prompts: got %v, want %v", resp.Prompts, fixture.Prompts)
		}
	}

	if fixture.HasSummary != nil {
		var resp protocol.ExtractSummaryResponse
		harness.RunAndUnmarshal(t, r, ctx, &resp, nil, "extract-summary", "--session-ref", path)
		if resp.HasSummary != *fixture.HasSummary {
			t.Errorf("has_summary: got %v, want %v", resp.HasSummary, *fixture.HasSummary)
		}
		if resp.Summary != fixture.Summary {
			t.Errorf("summary: got %q, want %q", resp.Summary, fixture.Summary)
		}
	}
}
