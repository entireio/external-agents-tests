package mandatory_test

import (
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
)

func TestInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		args  []string
	}{
		{"GetSessionID_InvalidJSON", []byte(`{"session_id":`), []string{"get-session-id"}},
		{"ReadSession_InvalidJSON", []byte(`{"session_id":`), []string{"read-session"}},
		{"WriteSession_InvalidJSON", []byte(`{"session_id":`), []string{"write-session"}},
		{"ChunkTranscript_InvalidMaxSize", []byte("test"), []string{"chunk-transcript", "--max-size", "0"}},
		{"ReassembleTranscript_InvalidJSON", []byte(`{"chunks":`), []string{"reassemble-transcript"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := harness.NewTestRunner(t)
			res := r.Run(harness.TestCtx(t), tt.input, tt.args...)
			harness.RequireFailure(t, res)
		})
	}
}

func TestUnknownSubcommand(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), nil, "this-subcommand-does-not-exist")
	harness.RequireFailure(t, res)
	if len(res.Stderr) == 0 {
		t.Error("unknown subcommand should produce diagnostic output on stderr")
	}
}

func TestNoSubcommand(t *testing.T) {
	r := harness.NewTestRunner(t)
	res := r.Run(harness.TestCtx(t), nil)
	harness.RequireFailure(t, res)
}
