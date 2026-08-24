package tui

import (
	"runtime/debug"
	"testing"
)

func TestReadVCSRevisionUsesBuildInfoSetting(t *testing.T) {
	got := readVCSRevision()
	info, ok := debug.ReadBuildInfo()
	if !ok {
		if got != "" {
			t.Fatalf("no build info, got %q", got)
		}
		return
	}
	want := ""
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			want = setting.Value
			break
		}
	}
	if got != want {
		t.Fatalf("vcs.revision %q, want %q", got, want)
	}
}
