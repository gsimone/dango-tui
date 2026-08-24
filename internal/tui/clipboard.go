package tui

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// copyText writes s to the clipboard. Tests replace this.
var copyText = defaultCopyText

var oscStdout io.Writer = os.Stdout
var oscStderr io.Writer = os.Stderr

func defaultCopyText(s string) error {
	oscErr := writeOSC52(s)
	sysErr := writeSystemClipboard(s)
	if oscErr != nil && sysErr != nil {
		return fmt.Errorf("clipboard: %v; %v", oscErr, sysErr)
	}
	return nil
}

func writeOSC52(s string) error {
	seq := "\x1b]52;c;" + base64.StdEncoding.EncodeToString([]byte(s)) + "\x07"
	_, outErr := oscStdout.Write([]byte(seq))
	_, errErr := oscStderr.Write([]byte(seq))
	if outErr != nil && errErr != nil {
		return outErr
	}
	return nil
}

func clipboardTimeout() time.Duration {
	if runtime.GOOS == "darwin" {
		return 2 * time.Second
	}
	return 400 * time.Millisecond
}

func writeSystemClipboard(s string) error {
	name, args := clipboardCommand()
	if name == "" {
		return fmt.Errorf("no system clipboard")
	}
	ctx, cancel := context.WithTimeout(context.Background(), clipboardTimeout())
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
