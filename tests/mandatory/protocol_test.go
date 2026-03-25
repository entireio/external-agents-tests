package mandatory_test

import (
	"strings"
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
)

func TestInfo(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), nil, "info")
	harness.RequireSuccess(t, res)

	var info protocol.InfoResponse
	harness.RequireUnmarshal(t, res.Stdout, &info)

	if info.ProtocolVersion != 1 {
		t.Errorf("protocol_version: got %d, want 1", info.ProtocolVersion)
	}
	if info.Name == "" {
		t.Error("name must not be empty")
	}
	if info.Type == "" {
		t.Error("type must not be empty")
	}
	if info.Description == "" {
		t.Error("description must not be empty")
	}
	if info.Capabilities.Hooks && len(info.HookNames) == 0 {
		t.Error("agent declares hooks capability but hook_names is empty")
	}
	if !info.Capabilities.Hooks && len(info.HookNames) > 0 {
		t.Errorf("agent does not declare hooks capability but hook_names has %d entries", len(info.HookNames))
	}
}

func TestDetect(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), nil, "detect")
	harness.RequireSuccess(t, res)

	var resp protocol.DetectResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	// present can be true or false depending on the environment.
}

func TestGetSessionID(t *testing.T) {
	r := harness.NewTestRunner(t)
	input := harness.HookInput("agent-spawn", "test-session-abc")
	res := r.Run(harness.TestCtx(t), harness.MustMarshal(t, input), "get-session-id")
	harness.RequireSuccess(t, res)

	var resp protocol.SessionIDResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	if resp.SessionID == "" {
		t.Error("session_id must not be empty")
	}
}

func TestGetSessionID_EmptyInput(t *testing.T) {
	r := harness.NewTestRunner(t)
	input := harness.HookInput("agent-spawn", "")
	res := r.Run(harness.TestCtx(t), harness.MustMarshal(t, input), "get-session-id")
	harness.RequireSuccess(t, res)

	var resp protocol.SessionIDResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	// Agent may echo the empty ID or generate one; both are valid.
}

func TestGetSessionDir(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), nil, "get-session-dir", "--repo-path", r.RepoRoot())
	harness.RequireSuccess(t, res)

	var resp protocol.SessionDirResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	if resp.SessionDir == "" {
		t.Error("session_dir must not be empty")
	}
}

func TestResolveSessionFile(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), nil, "resolve-session-file",
		"--session-dir", r.RepoRoot(),
		"--session-id", "test-session-123")
	harness.RequireSuccess(t, res)

	var resp protocol.SessionFileResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	if resp.SessionFile == "" {
		t.Error("session_file must not be empty")
	}
	if !strings.Contains(resp.SessionFile, "test-session-123") {
		t.Errorf("session_file %q should contain the session-id %q", resp.SessionFile, "test-session-123")
	}
}

func TestFormatResumeCommand(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), nil, "format-resume-command", "--session-id", "test-session-123")
	harness.RequireSuccess(t, res)

	var resp protocol.ResumeCommandResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	if resp.Command == "" {
		t.Error("command must not be empty")
	}
}

func TestChunkAndReassembleTranscript(t *testing.T) {
	r := harness.NewTestRunner(t)
	ctx := harness.TestCtx(t)

	// Create content larger than chunk size to force multiple chunks.
	original := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog.\n", 100))

	res := r.Run(ctx, original, "chunk-transcript", "--max-size", "512")
	harness.RequireSuccess(t, res)

	var chunks protocol.ChunkResponse
	harness.RequireUnmarshal(t, res.Stdout, &chunks)
	if len(chunks.Chunks) < 2 {
		t.Errorf("expected multiple chunks for %d bytes with max-size 512, got %d chunks",
			len(original), len(chunks.Chunks))
	}

	// Reassemble and verify round-trip.
	res = r.Run(ctx, harness.MustMarshal(t, chunks), "reassemble-transcript")
	harness.RequireSuccess(t, res)

	if string(res.Stdout) != string(original) {
		t.Errorf("round-trip mismatch: original %d bytes, reassembled %d bytes",
			len(original), len(res.Stdout))
	}
}

func TestChunkTranscript_SingleChunk(t *testing.T) {
	r := harness.NewTestRunner(t)

	small := []byte("small content")
	res := r.Run(harness.TestCtx(t), small, "chunk-transcript", "--max-size", "1048576")
	harness.RequireSuccess(t, res)

	var chunks protocol.ChunkResponse
	harness.RequireUnmarshal(t, res.Stdout, &chunks)
	if len(chunks.Chunks) != 1 {
		t.Errorf("expected 1 chunk for small content, got %d", len(chunks.Chunks))
	}
}
