package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/tui"
)

func TestDoctorIsStdoutOnlyAndExits(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	opened := false
	old := startTUI
	startTUI = func(tui.Model) error {
		opened = true
		t.Fatal("doctor must not open the TUI / alt screen / splash")
		return nil
	}
	t.Cleanup(func() { startTUI = old })

	var stdout, stderr strings.Builder
	code := run([]string{"--doctor"}, &stdout, &stderr)
	if opened {
		t.Fatal("startTUI ran")
	}
	if code != 2 {
		t.Fatalf("missing describe exits 2, got %d stderr %q", code, stderr.String())
	}
	if stderr.String() != "" {
		t.Fatalf("doctor is stdout only, stderr %q", stderr.String())
	}
	out := stdout.String()
	wantJSON := filepath.Join(dir, "dango.json")
	for _, needle := range []string{
		"cwd: " + dir,
		"looked: " + wantJSON,
		"wrote: " + wantJSON,
		"won: " + wantJSON,
		"describe: none",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q:\n%s", needle, out)
		}
	}
	if strings.Contains(out, "pane-hook-ok") {
		t.Fatalf("must not seed echo pane-hook-ok:\n%s", out)
	}
	for _, leak := range []string{"DANGO", "fetching", "STACKS", "\x1b[?1049", "●-●-●"} {
		if strings.Contains(out, leak) {
			t.Fatalf("TUI/splash leaked %q:\n%s", leak, out)
		}
	}
	raw, err := os.ReadFile(wantJSON)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "pane-hook-ok") || strings.Contains(string(raw), "echo") {
		t.Fatalf("wrote echo seed %q", raw)
	}
	if strings.TrimSpace(string(raw)) != `{}` {
		t.Fatalf("wrote %q", raw)
	}
}

func TestDoctorLeavesExistingFileAndSkipsTUI(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := filepath.Join(dir, "dango.json")
	original := `{"describe":"echo already-there"}` + "\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	old := startTUI
	startTUI = func(tui.Model) error {
		t.Fatal("doctor must not open the TUI")
		return nil
	}
	t.Cleanup(func() { startTUI = old })

	var stdout, stderr strings.Builder
	code := run([]string{"--doctor"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	if stderr.String() != "" {
		t.Fatalf("stderr %q", stderr.String())
	}
	out := stdout.String()
	if strings.Contains(out, "wrote:") {
		t.Fatalf("must not overwrite:\n%s", out)
	}
	if !strings.Contains(out, "won: "+path) {
		t.Fatalf("winning path:\n%s", out)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("file changed: %q", got)
	}
}
