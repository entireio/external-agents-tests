package compliancetest

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
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
}

// NewRunner returns a runner that invokes binaryPath with repoRoot as ENTIRE_REPO_ROOT.
func NewRunner(binaryPath, repoRoot string) *Runner {
	return &Runner{binaryPath: binaryPath, repoRoot: repoRoot}
}

// RepoRoot returns the directory set as ENTIRE_REPO_ROOT for this runner.
func (r *Runner) RepoRoot() string {
	return r.repoRoot
}

// Run executes the agent binary with the given arguments.
// If stdin is non-nil it is piped to the process.
func (r *Runner) Run(ctx context.Context, stdin []byte, args ...string) *Result {
	cmd := exec.CommandContext(ctx, r.binaryPath, args...)
	cmd.Env = append(os.Environ(),
		"ENTIRE_PROTOCOL_VERSION=1",
		"ENTIRE_REPO_ROOT="+r.repoRoot,
	)
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
