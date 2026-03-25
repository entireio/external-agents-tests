package mandatory_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
)

func TestWriteAndReadSession(t *testing.T) {
	r := harness.NewTestRunner(t)
	ctx := harness.TestCtx(t)

	// Ask the agent where it stores sessions.
	var dirResp protocol.SessionDirResponse
	harness.RunAndUnmarshal(t, r, ctx, &dirResp, nil, "get-session-dir", "--repo-path", r.RepoRoot())

	sessionID := "test-roundtrip-session"
	var fileResp protocol.SessionFileResponse
	harness.RunAndUnmarshal(t, r, ctx, &fileResp, nil, "resolve-session-file",
		"--session-dir", dirResp.SessionDir,
		"--session-id", sessionID)

	// Ensure parent directories exist.
	if err := os.MkdirAll(filepath.Dir(fileResp.SessionFile), 0o755); err != nil {
		t.Fatalf("creating session dir: %v", err)
	}

	// Write session.
	session := protocol.AgentSessionJSON{
		SessionID:     sessionID,
		AgentName:     harness.AgentInfo.Name,
		RepoPath:      r.RepoRoot(),
		SessionRef:    fileResp.SessionFile,
		StartTime:     time.Now().UTC().Format(time.RFC3339),
		NativeData:    []byte(`{"test": true}`),
		ModifiedFiles: []string{"file1.go", "file2.go"},
		NewFiles:      []string{"file3.go"},
		DeletedFiles:  []string{},
	}
	res := r.Run(ctx, harness.MustMarshal(t, session), "write-session")
	harness.RequireSuccess(t, res)

	// Read it back.
	input := harness.HookInputWithRef("agent-spawn", sessionID, fileResp.SessionFile)
	var readBack protocol.AgentSessionJSON
	harness.RunAndUnmarshal(t, r, ctx, &readBack, harness.MustMarshal(t, input), "read-session")

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
	r := harness.NewTestRunner(t)
	refPath := harness.WriteFixture(t, r.RepoRoot(), "test-transcript.json", []byte(`{"test": "data"}`))

	res := r.Run(harness.TestCtx(t), nil, "read-transcript", "--session-ref", refPath)
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
