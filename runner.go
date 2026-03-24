package compliancetest

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Result holds the output of an external agent binary invocation.
type Result struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
}

// Runner invokes an external agent binary with the protocol-required environment.
type Runner struct {
	binaryPath string
	repoRoot   string
	envRoot    string
}

// NewRunner returns a runner that invokes binaryPath with repoRoot as ENTIRE_REPO_ROOT.
func NewRunner(binaryPath, repoRoot string) *Runner {
	envRoot, err := os.MkdirTemp("", "external-agent-test-env-*")
	if err != nil {
		envRoot = os.TempDir()
	}
	return &Runner{binaryPath: binaryPath, repoRoot: repoRoot, envRoot: envRoot}
}

// NewRunnerWithEnvRoot returns a runner with an explicit environment root.
func NewRunnerWithEnvRoot(binaryPath, repoRoot, envRoot string) *Runner {
	return &Runner{binaryPath: binaryPath, repoRoot: repoRoot, envRoot: envRoot}
}

// RepoRoot returns the directory set as ENTIRE_REPO_ROOT for this runner.
func (r *Runner) RepoRoot() string {
	return r.repoRoot
}

// Run executes the agent binary with the given arguments.
// If stdin is non-nil it is piped to the process.
func (r *Runner) Run(ctx context.Context, stdin []byte, args ...string) *Result {
	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	cmd.Env = r.env()
	if info, err := os.Stat(r.repoRoot); err == nil && info.IsDir() {
		cmd.Dir = r.repoRoot
	}
	if stdin != nil {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return &Result{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: exitCode,
	}
}

func (r *Runner) env() []string {
	names := []string{
		"ENTIRE_PROTOCOL_VERSION",
		"ENTIRE_REPO_ROOT",
		"HOME",
		"USERPROFILE",
		"APPDATA",
		"LOCALAPPDATA",
		"XDG_CONFIG_HOME",
		"XDG_CACHE_HOME",
		"XDG_DATA_HOME",
		"TMPDIR",
		"TMP",
		"TEMP",
	}
	env := filterEnv(os.Environ(), names...)

	home := filepath.Join(r.envRoot, "home")
	config := filepath.Join(r.envRoot, "config")
	cache := filepath.Join(r.envRoot, "cache")
	data := filepath.Join(r.envRoot, "data")
	temp := filepath.Join(r.envRoot, "tmp")
	dirs := []string{home, config, cache, data, temp}

	if runtime.GOOS == "windows" {
		dirs = append(dirs,
			filepath.Join(r.envRoot, "appdata"),
			filepath.Join(r.envRoot, "localappdata"),
		)
	}
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		_ = os.MkdirAll(dir, 0o755)
	}

	env = append(env,
		"ENTIRE_PROTOCOL_VERSION=1",
		"ENTIRE_REPO_ROOT="+r.repoRoot,
		"HOME="+home,
		"XDG_CONFIG_HOME="+config,
		"XDG_CACHE_HOME="+cache,
		"XDG_DATA_HOME="+data,
		"TMPDIR="+temp,
		"TMP="+temp,
		"TEMP="+temp,
	)

	if runtime.GOOS == "windows" {
		env = append(env,
			"USERPROFILE="+home,
			"APPDATA="+filepath.Join(r.envRoot, "appdata"),
			"LOCALAPPDATA="+filepath.Join(r.envRoot, "localappdata"),
		)
	}

	return env
}

func filterEnv(env []string, names ...string) []string {
	out := make([]string, 0, len(env))
	for _, entry := range env {
		skip := false
		for _, name := range names {
			if len(entry) > len(name) && entry[:len(name)+1] == name+"=" {
				skip = true
				break
			}
		}
		if !skip {
			out = append(out, entry)
		}
	}
	return out
}
