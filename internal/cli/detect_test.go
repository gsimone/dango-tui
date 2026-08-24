package cli

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func TestDetectRepoFirstRemoteWhenNoOrigin(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	cmd = exec.Command("git", "remote", "add", "upstream", "https://github.com/other/repo.git")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git remote: %v %s", err, out)
	}
	got, err := DetectRepo(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got != "other/repo" {
		t.Fatalf("first remote: %q", got)
	}
}

func mustResolve(t *testing.T, args Args, dir string) Args {
	t.Helper()
	got, err := Resolve(args, dir)
	if err != nil {
		t.Fatal(err)
	}
	return got
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
	if cfg.Provider != "" || cfg.Describe != "" {
		t.Fatalf("missing file must not invent provider/describe: %+v", cfg)
	}
}

func TestReadDangoJSONDescribe(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dango.json"), []byte(`{"describe":"bin/describe-stack"}`), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadDangoConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Describe != "bin/describe-stack" || cfg.Provider != "" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestReadDangoYAMLDescribe(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dango.yml"), []byte("describe: bin/describe-stack\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadDangoConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Describe != "bin/describe-stack" {
		t.Fatalf("got %+v", cfg)
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

func TestReadDangoConfigPrefersCwdOverGitRoot(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(dir, "dango.yml"), []byte("provider: from-root\n"), 0644); err != nil {
		t.Fatal(err)
	}
	child := filepath.Join(dir, "pkg")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(child, "dango.yaml"), []byte("provider: from-cwd\ndescribe: bin/describe-stack\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := ReadDangoConfig(child)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Provider != "from-cwd" || cfg.Describe != "bin/describe-stack" {
		t.Fatalf("launch dir wins over git root, got %+v", cfg)
	}
}

func TestResolveDetectsRepoAndJSON(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(dir, "dango.json"), []byte(`{"provider":"codex@luna.medium"}`), 0644); err != nil {
		t.Fatal(err)
	}
	got := mustResolve(t, Args{}, dir)
	if got.Repo != "gsimone/leva-2" {
		t.Fatalf("repo %q", got.Repo)
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
	got := mustResolve(t, Args{Repo: "other/repo", Provider: ParseProvider("flag@model")}, dir)
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
	got := mustResolve(t, Args{}, dir)
	if got.Repo != "gsimone/leva-2" {
		t.Fatalf("repo %q", got.Repo)
	}
	if got.Provider.Raw != "codex@luna.medium" {
		t.Fatalf("provider %+v", got.Provider)
	}
}

func TestResolveMissingJSONHasNoProvider(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	got := mustResolve(t, Args{}, dir)
	if got.Repo != "gsimone/leva-2" {
		t.Fatalf("repo %q", got.Repo)
	}
	if got.Provider.Raw != "" {
		t.Fatalf("missing dango.json must not invent a provider: %+v", got.Provider)
	}
	if got.Describe != "" {
		t.Fatalf("missing dango.json must not invent describe: %q", got.Describe)
	}
}

func TestResolveDescribeFromConfigFileOnly(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	if err := os.WriteFile(filepath.Join(dir, "dango.json"), []byte(`{"describe":"bin/from-json"}`), 0644); err != nil {
		t.Fatal(err)
	}
	fromFile := mustResolve(t, Args{}, dir)
	if fromFile.Describe != "bin/from-json" {
		t.Fatalf("json describe: %q", fromFile.Describe)
	}
	ignored := mustResolve(t, Args{Describe: "bin/from-flag"}, dir)
	if ignored.Describe != "bin/from-json" {
		t.Fatalf("describe comes only from the config file: %q", ignored.Describe)
	}
}

func TestResolveDetectFailureIsLoudError(t *testing.T) {
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}

	got, err := Resolve(Args{}, dir)
	if err == nil {
		t.Fatal("detect miss must die, not fall back to examples")
	}
	if got.Repo != "" {
		t.Fatalf("failed resolve must not invent a repo, got %q", got.Repo)
	}
	msg := err.Error()
	if !strings.Contains(msg, "--repo archetype-labs/app") || !strings.Contains(msg, "--repo testdata/test.json") {
		t.Fatalf("error must name both --repo forms: %v", err)
	}
	if strings.Contains(msg, "example") || strings.Contains(msg, "fixture") {
		t.Fatalf("must not offer silent examples: %v", err)
	}
}

func TestResolveStoryHookStaysFixtures(t *testing.T) {
	dir := t.TempDir()
	gitInitWithOrigin(t, dir, "https://github.com/gsimone/leva-2.git")
	got := mustResolve(t, Args{Story: "mixed"}, dir)
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
	got := mustResolve(t, Args{Repo: "testdata/test.json"}, dir)
	if got.Repo != "testdata/test.json" {
		t.Fatalf("--repo file must win, got %q", got.Repo)
	}
	if got.Provider.Raw != "from-yml" {
		t.Fatalf("yml still sets provider: %+v", got.Provider)
	}
}
