package optional_test

import (
	"os"
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
	"github.com/entireio/external-agents-tests/internal/runner"
)

func TestDetect_WithFixtures(t *testing.T) {
	if harness.Fixtures == nil || harness.Fixtures.Detect == nil {
		t.Skip("no detect fixture configured (set AGENT_FIXTURES)")
	}

	fixture := *harness.Fixtures.Detect
	tests := []struct {
		name     string
		repoPath string
		want     bool
	}{
		{name: "present", repoPath: fixture.PresentRepo, want: true},
		{name: "absent", repoPath: fixture.AbsentRepo, want: false},
	}

	for _, tt := range tests {
		if tt.repoPath == "" {
			continue
		}
		t.Run(tt.name, func(t *testing.T) {
			path, err := harness.ResolveFixturePath(tt.repoPath)
			if err != nil {
				t.Fatalf("resolve fixture path: %v", err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatalf("stat fixture path %s: %v", path, err)
			}
			if !info.IsDir() {
				t.Fatalf("fixture path must be a directory: %s", path)
			}

			r := runner.NewRunnerWithEnvRoot(harness.BinaryPath, path, t.TempDir())
			var resp protocol.DetectResponse
			harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, nil, "detect")
			if resp.Present != tt.want {
				t.Errorf("present: got %v, want %v", resp.Present, tt.want)
			}
		})
	}
}
