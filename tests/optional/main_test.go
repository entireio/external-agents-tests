package optional_test

import (
	"os"
	"testing"

	"github.com/entireio/external-agents-tests/internal/harness"
)

func TestMain(m *testing.M) {
	harness.Init()
	os.Exit(m.Run())
}
