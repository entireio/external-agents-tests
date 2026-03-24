# external-agents-tests

Go compliance tests and a composite GitHub Action for validating an Entire external agent binary.

## What this repo does

This project runs protocol checks against a prebuilt external agent binary by invoking its CLI subcommands and validating the JSON responses.

It covers:

- required protocol behavior such as `info`, `detect`, session helpers, transcript chunking, and session read/write
- protocol error paths for malformed input and invalid arguments
- optional capability tests for hooks, transcript analysis, transcript preparation, token calculation, text generation, hook response writing, and subagent-aware extraction, gated by the agent's declared capabilities
- optional fixture-driven semantic checks for `detect` and transcript analysis

## Run locally

Build your external agent binary first, then run:

```bash
AGENT_BINARY=./bin/entire-agent-your-name go test -v -count=1 -timeout=5m ./...
```

Each test run uses an isolated `HOME`/XDG environment and sets the process working directory to `ENTIRE_REPO_ROOT`, so hook installation and config writes do not leak into the developer machine state.

## Optional semantic fixtures

If you want stronger black-box validation against real sample data, provide a fixtures file:

```bash
AGENT_BINARY=./bin/entire-agent-your-name \
AGENT_FIXTURES=./fixtures/compliance.json \
go test -v -count=1 -timeout=5m ./...
```

Example fixture schema:

```json
{
  "detect": {
    "present_repo": "../your-agent/testdata/detect-present",
    "absent_repo": "../your-agent/testdata/detect-absent"
  },
  "transcript_analyzer": {
    "session_ref": "../your-agent/testdata/transcript.json",
    "offset": 0,
    "position": 4,
    "modified_files": ["src/main.go", "src/main_test.go"],
    "prompts": ["add tests", "fix lint"],
    "summary": "Done.",
    "has_summary": true
  }
}
```

## Use as a GitHub Action

This repo also exposes a composite action through [`action.yml`](action.yml). It expects a path to an already-built executable.

```yaml
- uses: entireio/external-agents-tests@main
  with:
    binary-path: path/to/entire-agent-your-name
    go-version: stable
    fixtures-path: path/to/compliance.json
```

## Notes

- `AGENT_BINARY` is required for the test suite.
- `AGENT_FIXTURES` is optional. If set, additional semantic tests run.
- The test runner sets `ENTIRE_PROTOCOL_VERSION=1` and `ENTIRE_REPO_ROOT` before invoking the binary.
