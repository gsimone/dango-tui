package tui

import (
	"runtime/debug"
	"strings"
)

// vcsRevision is the VCS commit baked into this binary (go install @sha).
// Tests replace it.
var vcsRevision = readVCSRevision()

func readVCSRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			return strings.TrimSpace(setting.Value)
		}
	}
	return ""
}

func (m Model) splashSHA() string {
	return strings.TrimSpace(vcsRevision)
}
