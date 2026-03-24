# external-agents-tests

Go compliance tests and a composite GitHub Action for validating an Entire external agent binary.

## What this repo does

This project runs protocol checks against a prebuilt external agent binary by invoking its CLI subcommands and validating the JSON responses.

It covers:

- required protocol behavior such as `info`, `detect`, session helpers, transcript chunking, and session read/write
- optional capability tests for hooks and transcript analysis, gated by the agent's declared capabilities

## Run locally

Build your external agent binary first, then run:

```bash
AGENT_BINARY=./bin/entire-agent-your-name go test -v -count=1 -timeout=5m ./...
```

## Use as a GitHub Action

This repo also exposes a composite action through [`action.yml`](action.yml). It expects a path to an already-built executable.

```yaml
- uses: entireio/external-agents-tests@main
  with:
    binary-path: path/to/entire-agent-your-name
    go-version: stable
```

## Notes

- `AGENT_BINARY` is required for the test suite.
- The test runner sets `ENTIRE_PROTOCOL_VERSION=1` and `ENTIRE_REPO_ROOT` before invoking the binary.
