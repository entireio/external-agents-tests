package mandatory_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
)

func TestBinaryName(t *testing.T) {
	name := filepath.Base(harness.BinaryPath)
	if !strings.HasPrefix(name, "entire-agent-") {
		t.Errorf("binary name %q must start with \"entire-agent-\"", name)
	}
}

func TestInfo(t *testing.T) {
	r := harness.NewTestRunner(t)
	var info protocol.InfoResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &info, nil, "info")

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
	var resp protocol.DetectResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "detect")
	// present can be true or false depending on the environment.
}

func TestGetSessionID(t *testing.T) {
	r := harness.NewTestRunner(t)
	input := harness.HookInput("agent-spawn", "test-session-abc")
	var resp protocol.SessionIDResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, harness.MustMarshal(t, input), "get-session-id")
	if resp.SessionID == "" {
		t.Error("session_id must not be empty")
	}
}

func TestGetSessionID_EmptyInput(t *testing.T) {
	r := harness.NewTestRunner(t)
	input := harness.HookInput("agent-spawn", "")
	var resp protocol.SessionIDResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, harness.MustMarshal(t, input), "get-session-id")
	// Agent may echo the empty ID or generate one; both are valid.
}

func TestGetSessionDir(t *testing.T) {
	r := harness.NewTestRunner(t)
	var resp protocol.SessionDirResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "get-session-dir", "--repo-path", r.RepoRoot())
	if resp.SessionDir == "" {
		t.Error("session_dir must not be empty")
	}
}

func TestResolveSessionFile(t *testing.T) {
	r := harness.NewTestRunner(t)
	var resp protocol.SessionFileResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "resolve-session-file",
		"--session-dir", r.RepoRoot(),
		"--session-id", "test-session-123")
	if resp.SessionFile == "" {
		t.Error("session_file must not be empty")
	}
	if !strings.Contains(resp.SessionFile, "test-session-123") {
		t.Errorf("session_file %q should contain the session-id %q", resp.SessionFile, "test-session-123")
	}
}

func TestFormatResumeCommand(t *testing.T) {
	r := harness.NewTestRunner(t)
	var resp protocol.ResumeCommandResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "format-resume-command", "--session-id", "test-session-123")
	if resp.Command == "" {
		t.Error("command must not be empty")
	}
}

func TestChunkAndReassembleTranscript(t *testing.T) {
	r := harness.NewTestRunner(t)
	ctx := harness.TestCtx(t)

	// Create content larger than chunk size to force multiple chunks.
	original := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog.\n", 100))

	var chunks protocol.ChunkResponse
	harness.RunAndUnmarshal(t, r, ctx, &chunks, original, "chunk-transcript", "--max-size", "512")
	if len(chunks.Chunks) < 2 {
		t.Errorf("expected multiple chunks for %d bytes with max-size 512, got %d chunks",
			len(original), len(chunks.Chunks))
	}

	// Reassemble and verify round-trip.
	res := r.Run(ctx, harness.MustMarshal(t, chunks), "reassemble-transcript")
	harness.RequireSuccess(t, res)

	if string(res.Stdout) != string(original) {
		t.Errorf("round-trip mismatch: original %d bytes, reassembled %d bytes",
			len(original), len(res.Stdout))
	}
}

func TestChunkTranscript_SingleChunk(t *testing.T) {
	r := harness.NewTestRunner(t)
	var chunks protocol.ChunkResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &chunks, []byte("small content"), "chunk-transcript", "--max-size", "1048576")
	if len(chunks.Chunks) != 1 {
		t.Errorf("expected 1 chunk for small content, got %d", len(chunks.Chunks))
	}
}
