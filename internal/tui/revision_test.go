package tui

import (
	"runtime/debug"
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

func TestVersionSHAFromPseudoVersion(t *testing.T) {
	if got := versionSHA("v0.0.0-20260824134000-dcdcfd6f0806"); got != "dcdcfd6f0806" {
		t.Fatalf("pseudo-version suffix: %q", got)
	}
	if got := shortSHA(versionSHA("v0.0.0-20260824134000-dcdcfd6f0806")); got != "dcdcfd6" {
		t.Fatalf("short splash sha: %q", got)
	}
	if got := versionSHA("(devel)"); got != "" {
		t.Fatalf("devel: %q", got)
	}
	if got := versionSHA("v1.2.3"); got != "" {
		t.Fatalf("semver is not a sha: %q", got)
	}
}

func TestReadRevisionPrefersVCSThenModule(t *testing.T) {
	got := readRevision()
	info, ok := debug.ReadBuildInfo()
	if !ok {
		if got != "" {
			t.Fatalf("no build info, got %q", got)
		}
		return
	}
	want := ""
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && setting.Value != "" {
			want = setting.Value
			break
		}
	}
	if want == "" {
		want = versionSHA(info.Main.Version)
	}
	if got != want {
		t.Fatalf("revision %q, want %q (main %q)", got, want, info.Main.Version)
	}
}

func TestSplashSHAIsShort(t *testing.T) {
	withVCS(t, "dcdcfd6f0806bbcf032bd3cc43d2f77cb0452743")
	m := New(Options{
		Repo:   "archetype-labs/app",
		Width:  80,
		Height: 24,
		Fetch:  func(string) ([]domain.Stack, error) { return nil, nil },
	})
	if got := m.splashSHA(); got != "dcdcfd6" {
		t.Fatalf("splash sha %q", got)
	}
	frame := stripANSI(m.View())
	if !strings.Contains(frame, "dcdcfd6") {
		t.Fatalf("splash must show short sha:\n%s", frame)
	}
	if strings.Contains(frame, "dcdcfd6f0806bbcf") {
		t.Fatalf("splash must not dump the full sha:\n%s", frame)
	}
}
