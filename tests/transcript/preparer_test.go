package transcript_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
)

func TestTranscriptPreparer_PrepareTranscript(t *testing.T) {
	harness.RequireCapability(t, "transcript_preparer", harness.AgentInfo.Capabilities.TranscriptPreparer)
	r := harness.NewTestRunner(t)

	refPath := filepath.Join(r.RepoRoot(), "prepare-transcript.json")
	if err := os.WriteFile(refPath, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("writing fixture: %v", err)
	}

	res := r.Run(harness.TestCtx(t), nil, "prepare-transcript", "--session-ref", refPath)
	harness.RequireSuccess(t, res)
}
