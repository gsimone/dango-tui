package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitInitWithOrigin(t *testing.T, dir, origin string) {
	t.Helper()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	cmd = exec.Command("git", "remote", "add", "origin", origin)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote: %v %s", err, out)
	}
}

func TestParseRemote(t *testing.T) {
	cases := map[string]string{
		"https://github.com/gsimone/leva-2.git": "gsimone/leva-2",
		"git@github.com:gsimone/leva-2.git":     "gsimone/leva-2",
		"ssh://git@github.com/gsimone/leva-2":   "gsimone/leva-2",
	}
	for raw, want := range cases {
		got, err := ParseRemote(raw)
		if err != nil || got != want {
			t.Fatalf("%s: got %q err %v, want %s", raw, got, err, want)
		}
	}
}

func TestDetectRepoFromOrigin(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	got, err := DetectRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "gsimone/leva-2" {
		t.Fatalf("got %q", got)
	}
}

func TestReadDangoConfigMissingIsEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg, err := ReadDangoConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "" {
		t.Fatalf("missing file must not invent a provider: %+v", cfg)
	}
}

func TestReadDangoJSONProvider(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "git@github.com:gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(dir, "dango.json"), []byte(`{"provider":"codex@luna.medium"}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadDangoConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "codex@luna.medium" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestReadDangoYAMLProvider(t *testing.T) {
	for _, name := range []string{"dango.yml", "dango.yaml"} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, name), []byte("provider: codex@luna.medium\n"), 0644); err != nil {
			t.Fatal(err)
		}
		cfg, err := ReadDangoConfig(dir)
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Provider != "codex@luna.medium" {
			t.Fatalf("%s: %+v", name, cfg)
		}
	}
}

func TestReadDangoJSONWinsOverYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dango.json"), []byte(`{"provider":"from-json"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "dango.yml"), []byte("provider: from-yml\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadDangoConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "from-json" {
		t.Fatalf("json wins, got %+v", cfg)
	}
}

func TestReadDangoConfigPrefersRepoOverCwd(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(dir, "dango.yml"), []byte("provider: from-repo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "pkg")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "dango.yaml"), []byte("provider: from-cwd\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadDangoConfig(child)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "from-repo" {
		t.Fatalf("repo file wins over cwd, got %+v", cfg)
	}
}

func TestResolveEmptyRepoStaysExamples(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(dir, "dango.json"), []byte(`{"provider":"codex@luna.medium"}`), 0644); err != nil {
		t.Fatal(err)
	}
	got := Resolve(Args{}, dir)
	if got.Repo != "" {
		t.Fatalf("no --repo stays examples, got repo %q", got.Repo)
	}
	if got.Provider.Raw != "codex@luna.medium" {
		t.Fatalf("provider %+v", got.Provider)
	}
}

func TestResolveFlagsOverrideDetectAndYAML(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(dir, "dango.yml"), []byte("provider: codex@luna.medium\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := Resolve(Args{Repo: "other/repo", Provider: ParseProvider("flag@model")}, dir)
	if got.Repo != "other/repo" {
		t.Fatalf("repo %q", got.Repo)
	}
	if got.Provider.Raw != "flag@model" {
		t.Fatalf("provider %+v", got.Provider)
	}
}

func TestResolveYAMLProviderWithoutRepo(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(dir, "dango.yaml"), []byte("# title hook\nprovider: \"codex@luna.medium\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := Resolve(Args{}, dir)
	if got.Repo != "" {
		t.Fatalf("no --repo stays examples, got repo %q", got.Repo)
	}
	if got.Provider.Raw != "codex@luna.medium" {
		t.Fatalf("provider %+v", got.Provider)
	}
}

func TestResolveMissingJSONHasNoProvider(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	got := Resolve(Args{}, dir)
	if got.Repo != "" {
		t.Fatalf("no --repo stays examples, got repo %q", got.Repo)
	}
	if got.Provider.Raw != "" {
		t.Fatalf("missing dango.json must not invent a provider: %+v", got.Provider)
	}
}

func TestResolveStoryHookStaysFixtures(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	got := Resolve(Args{Story: "mixed"}, dir)
	if got.Repo != "" || got.Story != "mixed" {
		t.Fatalf("story must ignore detect: %+v", got)
	}
}

func TestResolveRepoFileWinsOverDetect(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(dir, "dango.yml"), []byte("provider: from-yml\n"), 0644); err != nil {
		t.Fatal(err)
	}
	got := Resolve(Args{Repo: "testdata/test.json"}, dir)
	if got.Repo != "testdata/test.json" {
		t.Fatalf("--repo file must win, got %q", got.Repo)
	}
	if got.Provider.Raw != "from-yml" {
		t.Fatalf("yml still sets provider: %+v", got.Provider)
	}
}
