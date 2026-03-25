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

	var resp protocol.GenerateTextResponse
	harness.RunAndUnmarshal(t, r, harness.TestCtx(t), &resp, []byte("Reply with a short sentence."), "generate-text", "--model", "compliance-test")
	if strings.TrimSpace(resp.Text) == "" {
		t.Error("generate-text returned empty text")
	}
}
