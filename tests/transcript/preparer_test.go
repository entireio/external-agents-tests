package transcript_test

import (
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
)

func TestTranscriptPreparer_PrepareTranscript(t *testing.T) {
	harness.RequireCapability(t, "transcript_preparer", harness.AgentInfo.Capabilities.TranscriptPreparer)
	r := harness.NewTestRunner(t)
	refPath := harness.WriteFixture(t, r.RepoRoot(), "prepare-transcript.json", []byte(`{}`))

	res := r.Run(harness.TestCtx(t), nil, "prepare-transcript", "--session-ref", refPath)
	harness.RequireSuccess(t, res)
}
