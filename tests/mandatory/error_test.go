package mandatory_test

import (
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
)

func TestGetSessionID_InvalidJSONFails(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), []byte(`{"session_id":`), "get-session-id")
	harness.RequireFailure(t, res)
}

func TestReadSession_InvalidJSONFails(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), []byte(`{"session_id":`), "read-session")
	harness.RequireFailure(t, res)
}

func TestWriteSession_InvalidJSONFails(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), []byte(`{"session_id":`), "write-session")
	harness.RequireFailure(t, res)
}

func TestChunkTranscript_InvalidMaxSizeFails(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), []byte("test"), "chunk-transcript", "--max-size", "0")
	harness.RequireFailure(t, res)
}

func TestReassembleTranscript_InvalidJSONFails(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), []byte(`{"chunks":`), "reassemble-transcript")
	harness.RequireFailure(t, res)
}

func TestUnknownSubcommand(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), nil, "this-subcommand-does-not-exist")
	harness.RequireFailure(t, res)
}

func TestNoSubcommand(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), nil)
	harness.RequireFailure(t, res)
}
