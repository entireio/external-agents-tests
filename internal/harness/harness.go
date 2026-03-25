package harness

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/entireio/external-agents-tests/internal/protocol"
	"github.com/entireio/external-agents-tests/internal/runner"
)

// Shared state, initialized once per test binary via Init().
var (
	BinaryPath string
	AgentInfo  *protocol.InfoResponse
	Fixtures   *ComplianceFixtures
)

// ComplianceFixtures holds optional fixture data for semantic tests.
type ComplianceFixtures struct {
	Detect             *DetectFixtures            `json:"detect,omitempty"`
	TranscriptAnalyzer *TranscriptAnalyzerFixture `json:"transcript_analyzer,omitempty"`
}

// DetectFixtures holds paths for detect fixture tests.
type DetectFixtures struct {
	PresentRepo string `json:"present_repo,omitempty"`
	AbsentRepo  string `json:"absent_repo,omitempty"`
}

// TranscriptAnalyzerFixture holds expected values for transcript analysis tests.
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

// Init resolves the agent binary, runs info to discover capabilities,
// and optionally loads fixtures. Call from each test package's TestMain
// before m.Run().
func Init() {
	BinaryPath = os.Getenv("AGENT_BINARY")
	if BinaryPath == "" {
		fmt.Fprintln(os.Stderr, "AGENT_BINARY environment variable is required")
		os.Exit(1)
	}

	fi, err := os.Stat(BinaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "cannot stat binary %s: %v\n", BinaryPath, err)
		os.Exit(1)
	}
	if fi.IsDir() {
		fmt.Fprintf(os.Stderr, "%s is a directory, not a binary\n", BinaryPath)
		os.Exit(1)
	}

	if fixturesPath := os.Getenv("AGENT_FIXTURES"); fixturesPath != "" {
		path, err := ResolveFixturePath(fixturesPath)
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
		Fixtures = &parsed
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

	r := runner.NewRunnerWithEnvRoot(BinaryPath, infoRepoRoot, infoEnvRoot)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	res := r.Run(ctx, nil, "info")
	cancel()

	if res.ExitCode != 0 {
		fmt.Fprintf(os.Stderr, "info subcommand failed (exit %d):\n%s\n", res.ExitCode, res.Stderr)
		os.Exit(1)
	}

	var parsed protocol.InfoResponse
	if err := json.Unmarshal(res.Stdout, &parsed); err != nil {
		fmt.Fprintf(os.Stderr, "parsing info response: %v\nstdout: %s\n", err, res.Stdout)
		os.Exit(1)
	}
	AgentInfo = &parsed

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
}

// --- test helpers ---

// TestCtx returns a context with a 30-second timeout, cancelled on test cleanup.
func TestCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// NewTestRunner creates a runner with temporary repo and env directories.
func NewTestRunner(t *testing.T) *runner.Runner {
	t.Helper()
	return runner.NewRunnerWithEnvRoot(BinaryPath, t.TempDir(), t.TempDir())
}

// RequireSuccess fails the test if the result has a non-zero exit code.
func RequireSuccess(t *testing.T, res *runner.Result) {
	t.Helper()
	if res.ExitCode != 0 {
		t.Fatalf("command failed (exit %d):\nstderr: %s\nstdout: %s",
			res.ExitCode, res.Stderr, res.Stdout)
	}
}

// RequireFailure fails the test if the result has a zero exit code.
func RequireFailure(t *testing.T, res *runner.Result) {
	t.Helper()
	if res.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code, got 0:\nstdout: %s", res.Stdout)
	}
}

// RequireUnmarshal unmarshals JSON data into v, failing the test on error.
func RequireUnmarshal(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, data)
	}
}

// RequireCapability skips the test if the agent does not declare the named capability.
func RequireCapability(t *testing.T, name string, has bool) {
	t.Helper()
	if !has {
		t.Skipf("agent does not declare %q capability", name)
	}
}

// MustMarshal marshals v to JSON, failing the test on error.
func MustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	return data
}

// HookInput returns a HookInputJSON with the given type and session ID.
func HookInput(hookType, sessionID string) protocol.HookInputJSON {
	return protocol.HookInputJSON{
		HookType:  hookType,
		SessionID: sessionID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// HookInputWithRef returns a HookInputJSON with a session ref.
func HookInputWithRef(hookType, sessionID, ref string) protocol.HookInputJSON {
	h := HookInput(hookType, sessionID)
	h.SessionRef = ref
	return h
}

// HookInputWithPrompt returns a HookInputJSON with a user prompt.
func HookInputWithPrompt(hookType, sessionID, prompt string) protocol.HookInputJSON {
	h := HookInput(hookType, sessionID)
	h.UserPrompt = prompt
	return h
}

// ResolveFixturePath resolves a potentially relative fixture path.
func ResolveFixturePath(path string) (string, error) {
	if filepath.IsAbs(path) {
		return path, nil
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(wd, path), nil
}

// IntArg converts an int to its string representation for CLI arguments.
func IntArg(v int) string {
	return strconv.Itoa(v)
}

// SameStrings checks if two string slices contain the same elements (order-independent).
func SameStrings(got, want []string) bool {
	gotCopy := append([]string(nil), got...)
	wantCopy := append([]string(nil), want...)
	sort.Strings(gotCopy)
	sort.Strings(wantCopy)
	return slices.Equal(gotCopy, wantCopy)
}

// RunAndUnmarshal runs a subcommand, requires success, and unmarshals stdout into v.
func RunAndUnmarshal(t *testing.T, r *runner.Runner, ctx context.Context, v any, input []byte, args ...string) {
	t.Helper()
	res := r.Run(ctx, input, args...)
	RequireSuccess(t, res)
	RequireUnmarshal(t, res.Stdout, v)
}

// WriteFixture writes content to a file in dir and returns the full path.
func WriteFixture(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, content, 0o644); err != nil {
		t.Fatalf("writing fixture %s: %v", name, err)
	}
	return p
}

// RequireNonNegativeTokenUsage fails if any token usage field is negative.
func RequireNonNegativeTokenUsage(t *testing.T, usage *protocol.TokenUsageResponse) {
	t.Helper()
	if usage == nil {
		t.Fatal("token usage response must not be nil")
	}
	fields := map[string]int{
		"input_tokens":          usage.InputTokens,
		"cache_creation_tokens": usage.CacheCreationTokens,
		"cache_read_tokens":     usage.CacheReadTokens,
		"output_tokens":         usage.OutputTokens,
		"api_call_count":        usage.APICallCount,
	}
	for name, value := range fields {
		if value < 0 {
			t.Errorf("%s must be non-negative, got %d", name, value)
		}
	}
	if usage.SubagentTokens != nil {
		RequireNonNegativeTokenUsage(t, usage.SubagentTokens)
	}
}
