package tui

import (
	"context"
	"encoding/base64"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// copyText writes s to the clipboard. Tests replace this.
var copyText = defaultCopyText

func defaultCopyText(s string) {
	writeOSC52(s)
	_ = writeSystemClipboard(s)
}

func writeOSC52(s string) {
	seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(s)) + "\x07"
	_, _ = os.Stderr.WriteString(seq)
}

func writeSystemClipboard(s string) error {
	name, args := clipboardCommand()
	if name == "" {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 400*time.Millisecond)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = strings.NewReader(s)
	return cmd.Run()
}

func clipboardCommand() (string, []string) {
	switch runtime.GOOS {
	case "darwin":
		return "pbcopy", nil
	case "windows":
		return "clip", nil
	}
	for _, cand := range []struct {
		name string
		args []string
	}{
		{"wl-copy", nil},
		{"xclip", []string{"-selection", "clipboard"}},
		{"xsel", []string{"--clipboard", "--input"}},
	} {
		if _, err := exec.LookPath(cand.name); err == nil {
			return cand.name, cand.args
		}
	}
	return "", nil
}
