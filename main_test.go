package compliancetest

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var (
	binaryPath         string
	agentInfo          *InfoResponse
	complianceFixtures *ComplianceFixtures
)

type ComplianceFixtures struct {
	Detect             *DetectFixtures            `json:"detect,omitempty"`
	TranscriptAnalyzer *TranscriptAnalyzerFixture `json:"transcript_analyzer,omitempty"`
}

type DetectFixtures struct {
	PresentRepo string `json:"present_repo,omitempty"`
	AbsentRepo  string `json:"absent_repo,omitempty"`
}

type TranscriptAnalyzerFixture struct {
	SessionRef     string   `json:"session_ref"`
	Offset         int      `json:"offset,omitempty"`
	Position       *int     `json:"position,omitempty"`
	ModifiedFiles  []string `json:"modified_files,omitempty"`
	Prompts        []string `json:"prompts,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	HasSummary     *bool    `json:"has_summary,omitempty"`
	MissingSession string   `json:"missing_session,omitempty"`
}

func TestMain(m *testing.M) {
	binaryPath = os.Getenv("AGENT_BINARY")
	if binaryPath == "" {
		fmt.Fprintln(os.Stderr, "AGENT_BINARY environment variable is required")
		os.Exit(1)
	}

	fi, err := os.Stat(binaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot stat binary %s: %v\n", binaryPath, err)
		os.Exit(1)
	}
	if fi.IsDir() {
		fmt.Fprintf(os.Stderr, "%s is a directory, not a binary\n", binaryPath)
		os.Exit(1)
	}

	if fixturesPath := os.Getenv("AGENT_FIXTURES"); fixturesPath != "" {
		path, err := resolveFixturePath(fixturesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve AGENT_FIXTURES: %v\n", err)
			os.Exit(1)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read fixtures file %s: %v\n", path, err)
			os.Exit(1)
		}
		var parsed ComplianceFixtures
		if err := json.Unmarshal(data, &parsed); err != nil {
			fmt.Fprintf(os.Stderr, "parse fixtures file %s: %v\n", path, err)
			os.Exit(1)
		}
		complianceFixtures = &parsed
	}

	// Fetch agent info to discover capabilities.
	infoRepoRoot, err := os.MkdirTemp("", "external-agent-info-repo-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create info repo root: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(infoRepoRoot)

	infoEnvRoot, err := os.MkdirTemp("", "external-agent-info-env-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create info env root: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(infoEnvRoot)

	r := NewRunnerWithEnvRoot(binaryPath, infoRepoRoot, infoEnvRoot)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	res := r.Run(ctx, nil, "info")
	cancel()

	if res.ExitCode != 0 {
		fmt.Fprintf(os.Stderr, "info subcommand failed (exit %d):\n%s\n", res.ExitCode, res.Stderr)
		os.Exit(1)
	}

	var parsed InfoResponse
	if err := json.Unmarshal(res.Stdout, &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "parsing info response: %v\nstdout: %s\n", err, res.Stdout)
		os.Exit(1)
	}
	agentInfo = &parsed

	fmt.Printf("Agent: %s (%s) protocol_version=%d\n", parsed.Name, parsed.Type, parsed.ProtocolVersion)
	fmt.Printf("Capabilities: hooks=%v transcript_analyzer=%v transcript_preparer=%v "+
		"token_calculator=%v text_generator=%v hook_response_writer=%v subagent_aware_extractor=%v\n",
		parsed.Capabilities.Hooks,
		parsed.Capabilities.TranscriptAnalyzer,
		parsed.Capabilities.TranscriptPreparer,
		parsed.Capabilities.TokenCalculator,
		parsed.Capabilities.TextGenerator,
		parsed.Capabilities.HookResponseWriter,
		parsed.Capabilities.SubagentAwareExtractor,
	)

	os.Exit(m.Run())
}

// --- test helpers ---

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func newTestRunner(t *testing.T) *Runner {
	t.Helper()
	return NewRunnerWithEnvRoot(binaryPath, t.TempDir(), t.TempDir())
}

func requireSuccess(t *testing.T, res *Result) {
	t.Helper()
	if res.ExitCode != 0 {
		t.Fatalf("command failed (exit %d):\nstderr: %s\nstdout: %s",
			res.ExitCode, res.Stderr, res.Stdout)
	}
}

func requireFailure(t *testing.T, res *Result) {
	t.Helper()
	if res.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code, got 0:\nstdout: %s", res.Stdout)
	}
}

func requireUnmarshal(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, data)
	}
}

func requireCapability(t *testing.T, name string, has bool) {
	t.Helper()
	if !has {
		t.Skipf("agent does not declare %q capability", name)
	}
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return data
}

func hookInput(hookType, sessionID string) HookInputJSON {
	return HookInputJSON{
		HookType:  hookType,
		SessionID: sessionID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func hookInputWithRef(hookType, sessionID, ref string) HookInputJSON {
	h := hookInput(hookType, sessionID)
	h.SessionRef = ref
	return h
}

func hookInputWithPrompt(hookType, sessionID, prompt string) HookInputJSON {
	h := hookInput(hookType, sessionID)
	h.UserPrompt = prompt
	return h
}

func resolveFixturePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, path), nil
}
