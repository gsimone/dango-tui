package tui

import (
	"runtime/debug"
	"strings"
)

// revisionOverride is an optional -ldflags default:
//
//	-X github.com/gsimone/dango-tui/internal/tui.revisionOverride=<sha>
//
// go install @commit often has no vcs.revision; splashSHA still
// reads module Main.Version (pseudo-version suffix).
var revisionOverride string

// vcsRevision is the baked commit shown on the splash. Tests replace it.
var vcsRevision = readRevision()

func readRevision() string {
	if v := strings.TrimSpace(revisionOverride); v != "" {
		return v
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && strings.TrimSpace(setting.Value) != "" {
			return setting.Value
		}
	}
	return versionSHA(info.Main.Version)
}

func versionSHA(ver string) string {
	ver = strings.TrimSpace(ver)
	if ver == "" || ver == "(devel)" {
		return ""
	}
	i := strings.LastIndex(ver, "-")
	if i < 0 || i+1 >= len(ver) {
		return ""
	}
	suf := ver[i+1:]
	if !isHexSHA(suf) {
		return ""
	}
	return suf
}

func isHexSHA(s string) bool {
	if len(s) < 7 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

func shortSHA(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 7 {
		return s[:7]
	}
	return s
}

func (m Model) splashSHA() string {
	return shortSHA(vcsRevision)
}
