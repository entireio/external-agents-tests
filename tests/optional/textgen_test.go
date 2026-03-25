package optional_test

import (
	"strings"
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
	"github.com/entireio/external-agents-tests/internal/protocol"
)

func TestTextGenerator_GenerateText(t *testing.T) {
	harness.RequireCapability(t, "text_generator", harness.AgentInfo.Capabilities.TextGenerator)
	r := harness.NewTestRunner(t)

	res := r.Run(harness.TestCtx(t), []byte("Reply with a short sentence."), "generate-text", "--model", "compliance-test")
	harness.RequireSuccess(t, res)

	var resp protocol.GenerateTextResponse
	harness.RequireUnmarshal(t, res.Stdout, &resp)
	if strings.TrimSpace(resp.Text) == "" {
		t.Error("generate-text returned empty text")
	}
}
