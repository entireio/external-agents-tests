package main

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed go.mod internal tests
var testSources embed.FS

const usage = `Usage: external-agents-tests verify <path-to-binary> [--fixtures <path>]

Runs the Entire external agent protocol compliance tests against the given binary.

Arguments:
  <path-to-binary>     Path to the external agent binary to test

Options:
  --fixtures <path>    Path to a JSON fixtures file for semantic detect/transcript checks
  -h, --help           Show this help message`

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Println(usage)
		os.Exit(0)
	}

	if args[0] != "verify" {
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s\n", args[0], usage)
		os.Exit(1)
	}

	args = args[1:]
	if len(args) == 0 {
		fmt.Fprintf(os.Stderr, "error: missing required argument <path-to-binary>\n\n%s\n", usage)
		os.Exit(1)
	}

	var binaryPath, fixturesPath string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--fixtures":
			if i+1 >= len(args) {
				fmt.Fprintf(os.Stderr, "error: --fixtures requires a value\n")
				os.Exit(1)
			}
			i++
			fixturesPath = args[i]
		case "-h", "--help":
			fmt.Println(usage)
			os.Exit(0)
		default:
			if binaryPath != "" {
				fmt.Fprintf(os.Stderr, "error: unexpected argument: %s\n\n%s\n", args[i], usage)
				os.Exit(1)
			}
			binaryPath = args[i]
		}
	}

	if binaryPath == "" {
		fmt.Fprintf(os.Stderr, "error: missing required argument <path-to-binary>\n\n%s\n", usage)
		os.Exit(1)
	}

	// Resolve binary path to absolute.
	binaryPath, err := filepath.Abs(binaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: resolving binary path: %v\n", err)
		os.Exit(1)
	}

	fi, err := os.Stat(binaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: cannot stat binary %s: %v\n", binaryPath, err)
		os.Exit(1)
	}
	if fi.IsDir() {
		fmt.Fprintf(os.Stderr, "error: %s is a directory, not a binary\n", binaryPath)
		os.Exit(1)
	}
	if runtime.GOOS != "windows" && fi.Mode()&0o111 == 0 {
		fmt.Fprintf(os.Stderr, "error: %s is not executable\n", binaryPath)
		os.Exit(1)
	}

	if name := filepath.Base(binaryPath); !strings.HasPrefix(name, "entire-agent-") {
		fmt.Fprintf(os.Stderr, "error: binary name %q must start with \"entire-agent-\"\n", name)
		os.Exit(1)
	}

	// Resolve fixtures path to absolute if provided.
	if fixturesPath != "" {
		fixturesPath, err = filepath.Abs(fixturesPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: resolving fixtures path: %v\n", err)
			os.Exit(1)
		}
	}

	// Find the test source directory: the directory containing this binary,
	// falling back to the directory containing go.mod relative to the executable,
	// and finally extracting the embedded test sources.
	testDir, embedded, err := findTestDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	if embedded {
		defer os.RemoveAll(testDir)
	}

	// Run go test.
	cmd := exec.Command("go", "test", "-v", "-count=1", "-timeout=5m", "./...")
	cmd.Dir = testDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"AGENT_BINARY="+binaryPath,
		"AGENT_FIXTURES="+fixturesPath,
	)

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// findTestDir locates the project root containing go.mod.
// It checks the executable's directory first, then CWD, and finally
// extracts the embedded test sources to a temp directory.
// The bool return indicates whether a temp directory was created (caller should clean up).
func findTestDir() (string, bool, error) {
	// Try executable's directory (for running from source tree).
	exe, err := os.Executable()
	if err == nil {
		exe, err = filepath.EvalSymlinks(exe)
		if err == nil {
			dir := filepath.Dir(exe)
			if hasGoMod(dir) {
				return dir, false, nil
			}
		}
	}

	// Fall back to CWD.
	wd, err := os.Getwd()
	if err != nil {
		return "", false, fmt.Errorf("cannot determine working directory: %v", err)
	}
	if hasGoMod(wd) {
		return wd, false, nil
	}

	// Extract embedded test sources to a temp directory.
	dir, err := extractEmbeddedSources()
	if err != nil {
		return "", false, err
	}
	return dir, true, nil
}

func hasGoMod(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

// extractEmbeddedSources writes the embedded test source tree to a temp
// directory and returns the path.
func extractEmbeddedSources() (string, error) {
	dir, err := os.MkdirTemp("", "external-agents-tests-*")
	if err != nil {
		return "", fmt.Errorf("creating temp directory: %v", err)
	}

	err = fs.WalkDir(testSources, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		target := filepath.Join(dir, path)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := testSources.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		os.RemoveAll(dir)
		return "", fmt.Errorf("extracting embedded test sources: %v", err)
	}

	return dir, nil
}
