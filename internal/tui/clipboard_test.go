package tui

import (
	"bytes"
	"encoding/base64"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWriteOSC52GoesToStdoutAndStderr(t *testing.T) {
	var out, errBuf bytes.Buffer
	oldOut, oldErr := oscStdout, oscStderr
	oscStdout, oscStderr = &out, &errBuf
	t.Cleanup(func() { oscStdout, oscStderr = oldOut, oldErr })

	if err := writeOSC52("gh pr list --repo archetype-labs/app"); err != nil {
		t.Fatal(err)
	}
	payload := base64.StdEncoding.EncodeToString([]byte("gh pr list --repo archetype-labs/app"))
	want := "\x1b]52;c;" + payload + "\x07"
	if out.String() != want {
		t.Fatalf("stdout OSC52 %q", out.String())
	}
	if errBuf.String() != want {
		t.Fatalf("stderr OSC52 %q", errBuf.String())
	}
}

func TestDarwinClipboardTimeoutIsTwoSeconds(t *testing.T) {
	got := clipboardTimeout()
	if runtime.GOOS == "darwin" {
		if got != 2*time.Second {
			t.Fatalf("darwin pbcopy timeout %s, want 2s", got)
		}
		name, args := clipboardCommand()
		if name != "pbcopy" || len(args) != 0 {
			t.Fatalf("darwin clipboard %s %v", name, args)
		}
		return
	}
	if got <= 0 {
		t.Fatalf("timeout %s", got)
	}
}

func TestDefaultCopyTextWritesOSC52EvenWithoutSystemClipboard(t *testing.T) {
	if runtime.GOOS == "darwin" || runtime.GOOS == "windows" {
		t.Skip("system clipboard exists")
	}
	var out bytes.Buffer
	oldOut, oldErr := oscStdout, oscStderr
	oscStdout, oscStderr = &out, &out
	t.Cleanup(func() { oscStdout, oscStderr = oldOut, oldErr })

	if err := defaultCopyText("argv"); err != nil {
		t.Fatalf("stdout OSC52 is enough: %v", err)
	}
	if !strings.Contains(out.String(), "\x1b]52;c;") {
		t.Fatalf("missing OSC52: %q", out.String())
	}
}
