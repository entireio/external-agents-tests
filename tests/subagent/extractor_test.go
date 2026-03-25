package subagent_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
)

func TestSubagentAwareExtractor_ExtractAllModifiedFiles(t *testing.T) {
	harness.RequireCapability(t, "subagent_aware_extractor", harness.AgentInfo.Capabilities.SubagentAwareExtractor)
	r := harness.NewTestRunner(t)
	subagentsDir := filepath.Join(r.RepoRoot(), "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("creating subagents dir: %v", err)
	}

	res := r.Run(harness.TestCtx(t), []byte(`{}`), "extract-all-modified-files",
		"--offset", "0",
		"--subagents-dir", subagentsDir)
	harness.RequireSuccess(t, res)

	var resp protocol.ExtractFilesResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
}

func TestSubagentAwareExtractor_CalculateTotalTokens(t *testing.T) {
	harness.RequireCapability(t, "subagent_aware_extractor", harness.AgentInfo.Capabilities.SubagentAwareExtractor)
	r := harness.NewTestRunner(t)
	subagentsDir := filepath.Join(r.RepoRoot(), "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("creating subagents dir: %v", err)
	}

	res := r.Run(harness.TestCtx(t), []byte(`{}`), "calculate-total-tokens",
		"--offset", "0",
		"--subagents-dir", subagentsDir)
	harness.RequireSuccess(t, res)

	var resp protocol.TokenUsageResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	harness.RequireNonNegativeTokenUsage(t, &resp)
}
