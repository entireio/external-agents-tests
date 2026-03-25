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

	var resp protocol.ExtractFilesResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, []byte(`{}`), "extract-all-modified-files",
		"--offset", "0",
		"--subagents-dir", subagentsDir)
}

func TestSubagentAwareExtractor_CalculateTotalTokens(t *testing.T) {
	harness.RequireCapability(t, "subagent_aware_extractor", harness.AgentInfo.Capabilities.SubagentAwareExtractor)
	r := harness.NewTestRunner(t)
	subagentsDir := filepath.Join(r.RepoRoot(), "subagents")
	if err := os.MkdirAll(subagentsDir, 0o755); err != nil {
		t.Fatalf("creating subagents dir: %v", err)
	}

	var resp protocol.TokenUsageResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, []byte(`{}`), "calculate-total-tokens",
		"--offset", "0",
		"--subagents-dir", subagentsDir)
	harness.RequireNonNegativeTokenUsage(t, &resp)
}
