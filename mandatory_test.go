package compliancetest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInfo(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), nil, "info")
	requireSuccess(t, res)

	var info InfoResponse
	requireUnmarshal(t, res.Stdout, &info)

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
}

func TestDetect(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), nil, "detect")
	requireSuccess(t, res)

	var resp DetectResponse
	requireUnmarshal(t, res.Stdout, &resp)
	// present can be true or false depending on the environment.
}

func TestGetSessionID(t *testing.T) {
	r := newTestRunner(t)
	input := hookInput("agent-spawn", "test-session-abc")
	res := r.Run(testCtx(t), mustMarshal(t, input), "get-session-id")
	requireSuccess(t, res)

	var resp SessionIDResponse
	requireUnmarshal(t, res.Stdout, &resp)
	if resp.SessionID == "" {
		t.Error("session_id must not be empty")
	}
}

func TestGetSessionID_EmptyInput(t *testing.T) {
	r := newTestRunner(t)
	input := hookInput("agent-spawn", "")
	res := r.Run(testCtx(t), mustMarshal(t, input), "get-session-id")
	requireSuccess(t, res)

	var resp SessionIDResponse
	requireUnmarshal(t, res.Stdout, &resp)
	// Agent may echo the empty ID or generate one; both are valid.
}

func TestGetSessionDir(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), nil, "get-session-dir", "--repo-path", r.RepoRoot())
	requireSuccess(t, res)

	var resp SessionDirResponse
	requireUnmarshal(t, res.Stdout, &resp)
	if resp.SessionDir == "" {
		t.Error("session_dir must not be empty")
	}
}

func TestResolveSessionFile(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), nil, "resolve-session-file",
		"--session-dir", r.RepoRoot(),
		"--session-id", "test-session-123")
	requireSuccess(t, res)

	var resp SessionFileResponse
	requireUnmarshal(t, res.Stdout, &resp)
	if resp.SessionFile == "" {
		t.Error("session_file must not be empty")
	}
}

func TestFormatResumeCommand(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), nil, "format-resume-command", "--session-id", "test-session-123")
	requireSuccess(t, res)

	var resp ResumeCommandResponse
	requireUnmarshal(t, res.Stdout, &resp)
	if resp.Command == "" {
		t.Error("command must not be empty")
	}
}

func TestChunkAndReassembleTranscript(t *testing.T) {
	r := newTestRunner(t)
	ctx := testCtx(t)

	// Create content larger than chunk size to force multiple chunks.
	original := []byte(strings.Repeat("The quick brown fox jumps over the lazy dog.\n", 100))

	res := r.Run(ctx, original, "chunk-transcript", "--max-size", "512")
	requireSuccess(t, res)

	var chunks ChunkResponse
	requireUnmarshal(t, res.Stdout, &chunks)
	if len(chunks.Chunks) < 2 {
		t.Errorf("expected multiple chunks for %d bytes with max-size 512, got %d chunks",
			len(original), len(chunks.Chunks))
	}

	// Reassemble and verify round-trip.
	res = r.Run(ctx, mustMarshal(t, chunks), "reassemble-transcript")
	requireSuccess(t, res)

	if string(res.Stdout) != string(original) {
		t.Errorf("round-trip mismatch: original %d bytes, reassembled %d bytes",
			len(original), len(res.Stdout))
	}
}

func TestChunkTranscript_SingleChunk(t *testing.T) {
	r := newTestRunner(t)

	small := []byte("small content")
	res := r.Run(testCtx(t), small, "chunk-transcript", "--max-size", "1048576")
	requireSuccess(t, res)

	var chunks ChunkResponse
	requireUnmarshal(t, res.Stdout, &chunks)
	if len(chunks.Chunks) != 1 {
		t.Errorf("expected 1 chunk for small content, got %d", len(chunks.Chunks))
	}
}

func TestWriteAndReadSession(t *testing.T) {
	r := newTestRunner(t)
	ctx := testCtx(t)

	// Ask the agent where it stores sessions.
	res := r.Run(ctx, nil, "get-session-dir", "--repo-path", r.RepoRoot())
	requireSuccess(t, res)
	var dirResp SessionDirResponse
	requireUnmarshal(t, res.Stdout, &dirResp)

	sessionID := "test-roundtrip-session"
	res = r.Run(ctx, nil, "resolve-session-file",
		"--session-dir", dirResp.SessionDir,
		"--session-id", sessionID)
	requireSuccess(t, res)
	var fileResp SessionFileResponse
	requireUnmarshal(t, res.Stdout, &fileResp)

	// Ensure parent directories exist.
	if err := os.MkdirAll(filepath.Dir(fileResp.SessionFile), 0o755); err != nil {
		t.Fatalf("creating session dir: %v", err)
	}

	// Write session.
	session := AgentSessionJSON{
		SessionID:     sessionID,
		AgentName:     agentInfo.Name,
		RepoPath:      r.RepoRoot(),
		SessionRef:    fileResp.SessionFile,
		StartTime:     time.Now().UTC().Format(time.RFC3339),
		NativeData:    []byte(`{"test": true}`),
		ModifiedFiles: []string{"file1.go", "file2.go"},
		NewFiles:      []string{"file3.go"},
		DeletedFiles:  []string{},
	}
	res = r.Run(ctx, mustMarshal(t, session), "write-session")
	requireSuccess(t, res)

	// Read it back.
	input := hookInputWithRef("agent-spawn", sessionID, fileResp.SessionFile)
	res = r.Run(ctx, mustMarshal(t, input), "read-session")
	requireSuccess(t, res)

	var readBack AgentSessionJSON
	requireUnmarshal(t, res.Stdout, &readBack)

	if readBack.SessionID != sessionID {
		t.Errorf("session_id: got %q, want %q", readBack.SessionID, sessionID)
	}
	if readBack.AgentName != session.AgentName {
		t.Errorf("agent_name: got %q, want %q", readBack.AgentName, session.AgentName)
	}
	if readBack.RepoPath != session.RepoPath {
		t.Errorf("repo_path: got %q, want %q", readBack.RepoPath, session.RepoPath)
	}
	if readBack.SessionRef != session.SessionRef {
		t.Errorf("session_ref: got %q, want %q", readBack.SessionRef, session.SessionRef)
	}
	if string(readBack.NativeData) != string(session.NativeData) {
		t.Errorf("native_data: got %q, want %q", readBack.NativeData, session.NativeData)
	}
	// Agents may derive file lists from native transcript data instead of
	// persisting the caller-provided values verbatim. We only require that the
	// protocol fields are present and initialized on readback.
	if readBack.ModifiedFiles == nil {
		t.Error("modified_files must be initialized")
	}
	if readBack.NewFiles == nil {
		t.Error("new_files must be initialized")
	}
	if readBack.DeletedFiles == nil {
		t.Error("deleted_files must be initialized")
	}
}

func TestReadTranscript(t *testing.T) {
	r := newTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "test-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{"test": "data"}`), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	res := r.Run(testCtx(t), nil, "read-transcript", "--session-ref", refPath)
	// The agent may fail if it expects a specific transcript format.
	// We only verify it handles the request without crashing.
	if res.ExitCode != 0 {
		t.Logf("read-transcript returned exit %d (may require agent-specific format): %s",
			res.ExitCode, res.Stderr)
		return
	}
	if len(res.Stdout) == 0 {
		t.Error("read-transcript succeeded but returned empty output")
	}
}

func TestGetSessionID_InvalidJSONFails(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), []byte(`{"session_id":`), "get-session-id")
	requireFailure(t, res)
}

func TestReadSession_InvalidJSONFails(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), []byte(`{"session_id":`), "read-session")
	requireFailure(t, res)
}

func TestWriteSession_InvalidJSONFails(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), []byte(`{"session_id":`), "write-session")
	requireFailure(t, res)
}

func TestChunkTranscript_InvalidMaxSizeFails(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), []byte("test"), "chunk-transcript", "--max-size", "0")
	requireFailure(t, res)
}

func TestReassembleTranscript_InvalidJSONFails(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), []byte(`{"chunks":`), "reassemble-transcript")
	requireFailure(t, res)
}

func TestUnknownSubcommand(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), nil, "this-subcommand-does-not-exist")
	requireFailure(t, res)
}

func TestNoSubcommand(t *testing.T) {
	r := newTestRunner(t)
	res := r.Run(testCtx(t), nil)
	requireFailure(t, res)
}
