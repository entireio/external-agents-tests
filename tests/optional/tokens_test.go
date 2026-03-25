package optional_test

import (
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
)

func TestTokenCalculator_CalculateTokens(t *testing.T) {
	harness.RequireCapability(t, "token_calculator", harness.AgentInfo.Capabilities.TokenCalculator)
	r := harness.NewTestRunner(t)

	var resp protocol.TokenUsageResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, []byte(`{}`), "calculate-tokens", "--offset", "0")
	harness.RequireNonNegativeTokenUsage(t, &resp)
}
