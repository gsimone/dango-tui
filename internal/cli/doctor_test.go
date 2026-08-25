package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDoctorProbeTimeoutAllowsLunaExec(t *testing.T) {
	if doctorProbeTimeout != 45*time.Second {
		t.Fatalf("doctorProbeTimeout=%s; Luna exec needs ~45s, not 8s echo", doctorProbeTimeout)
	}
}

func TestDoctorDescribeJSONIsNotEchoPaneHookOK(t *testing.T) {
	if strings.Contains(doctorDescribeJSON, "pane-hook-ok") || strings.Contains(doctorDescribeJSON, "echo") {
		t.Fatalf("doctor must not seed echo pane-hook-ok, got %q", doctorDescribeJSON)
	}
	if strings.Contains(doctorDescribeJSON, "describe") {
		t.Fatalf("doctor write has no describe key, got %q", doctorDescribeJSON)
	}
}

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
	err := Doctor(dir, &buf)
	if err == nil {
		t.Fatal("empty describe must exit 2")
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
	if !strings.Contains(got, "describe: none") {
		t.Fatalf("missing describe stays none:\n%s", got)
	}
	if strings.Contains(got, "pane-hook-ok") || strings.Contains(got, `{"describe":"echo`) {
		t.Fatalf("must not seed echo pane-hook-ok:\n%s", got)
	}
	raw, err := os.ReadFile(wantPath)
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

func TestDoctorBareDangoDescribeStaysPATHCommand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dango.json")
	if err := os.WriteFile(path, []byte(`{"describe":"dango-describe"}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	_ = Doctor(dir, &buf)
	got := buf.String()
	if !strings.Contains(got, "describe: dango-describe") {
		t.Fatalf("bare name stays on PATH:\n%s", got)
	}
	if strings.Contains(got, "describe: "+filepath.Join(dir, "dango-describe")) {
		t.Fatalf("must not join dango-describe to the config dir:\n%s", got)
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
	err := Doctor(child, &buf)
	if err == nil {
		t.Fatal("cwd write with no describe must exit 2")
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
	if !strings.Contains(got, "describe: none") {
		t.Fatalf("cwd write has no describe:\n%s", got)
	}
	if strings.Contains(got, "pane-hook-ok") {
		t.Fatalf("must not seed echo pane-hook-ok:\n%s", got)
	}
}
