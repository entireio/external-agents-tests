package optional_test

import (
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
)

func TestHookResponseWriter_WriteHookResponse(t *testing.T) {
	harness.RequireCapability(t, "hook_response_writer", harness.AgentInfo.Capabilities.HookResponseWriter)
	r := harness.NewTestRunner(t)

	res := r.Run(harness.TestCtx(t), nil, "write-hook-response", "--message", "compliance test response")
	harness.RequireSuccess(t, res)
}
