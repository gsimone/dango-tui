package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDoctorNeedsNoRepo(t *testing.T) {
	args, err := Parse([]string{"--doctor"})
	if err != nil {
		t.Fatal(err)
	}
	if !args.Doctor {
		t.Fatal("doctor flag")
	}
	if args.Repo != "" {
		t.Fatalf("doctor must not invent a repo, got %q", args.Repo)
	}
}

func TestDoctorWritesMissingCwdJSON(t *testing.T) {
	dir := t.TempDir()
	var buf strings.Builder
	if err := Doctor(dir, &buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	wantPath := filepath.Join(dir, "dango.json")
	if !strings.Contains(got, "cwd: "+dir) {
		t.Fatalf("cwd:\n%s", got)
	}
	if !strings.Contains(got, "looked: "+wantPath) {
		t.Fatalf("looked at cwd json:\n%s", got)
	}
	if !strings.Contains(got, "wrote: "+wantPath) {
		t.Fatalf("must write missing cwd file:\n%s", got)
	}
	if !strings.Contains(got, "won: "+wantPath) {
		t.Fatalf("winning path:\n%s", got)
	}
	if !strings.Contains(got, "describe: echo pane-hook-ok") {
		t.Fatalf("describe argv:\n%s", got)
	}
	if !strings.Contains(got, "stdout: pane-hook-ok") {
		t.Fatalf("echo probe:\n%s", got)
	}
	raw, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(raw)) != `{"describe":"echo pane-hook-ok"}` {
		t.Fatalf("wrote %q", raw)
	}
}

func TestDoctorLeavesExistingFileAlone(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dango.json")
	original := `{"describe":"echo already-there"}` + "\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := Doctor(dir, &buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, "wrote:") {
		t.Fatalf("must not overwrite:\n%s", got)
	}
	if !strings.Contains(got, "won: "+path) {
		t.Fatalf("winning path:\n%s", got)
	}
	if !strings.Contains(got, "describe: echo already-there") {
		t.Fatalf("existing describe:\n%s", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != original {
		t.Fatalf("file changed: %q", raw)
	}
}

func TestDoctorDoesNotOverwriteEmptyJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dango.json")
	if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err := Doctor(dir, &buf)
	if err == nil {
		t.Fatal("empty describe must exit 2")
	}
	got := buf.String()
	if strings.Contains(got, "wrote:") {
		t.Fatalf("must not overwrite existing cwd file:\n%s", got)
	}
	if !strings.Contains(got, "won: "+path) {
		t.Fatalf("winning path:\n%s", got)
	}
	if !strings.Contains(got, "describe: none") {
		t.Fatalf("describe none:\n%s", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{}` {
		t.Fatalf("overwrote: %q", raw)
	}
}

func TestDoctorCwdJSONWinsOverGitRoot(t *testing.T) {
	root := t.TempDir()
	gitInitWithOrigin(t, root, "https://github.com/gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(root, "dango.json"), []byte(`{"describe":"echo from-root"}`), 0644); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(root, "pkg")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}

	var buf strings.Builder
	if err := Doctor(child, &buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	wrote := filepath.Join(child, "dango.json")
	if !strings.Contains(got, "looked: "+filepath.Join(root, "dango.json")) {
		t.Fatalf("must list git root path:\n%s", got)
	}
	if !strings.Contains(got, "wrote: "+wrote) {
		t.Fatalf("cwd missing file must be written:\n%s", got)
	}
	if !strings.Contains(got, "won: "+wrote) {
		t.Fatalf("cwd wins:\n%s", got)
	}
	if strings.Contains(got, "won: "+filepath.Join(root, "dango.json")) {
		t.Fatalf("git root must not win:\n%s", got)
	}
	if !strings.Contains(got, "describe: echo pane-hook-ok") {
		t.Fatalf("describe:\n%s", got)
	}
}
