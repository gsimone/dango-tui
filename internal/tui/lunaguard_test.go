package tui_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	_ = os.Setenv("DANGO_OPENAI_API_KEY", "")
	_ = os.Setenv("OPENAI_API_KEY", "")
	_ = os.Setenv("DANGO_API_KEY", "")
	os.Exit(m.Run())
}
