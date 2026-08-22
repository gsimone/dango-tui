package cli

import (
	"flag"
	"strings"
	"testing"
)

func TestParseRepoAndProvider(t *testing.T) {
	args, err := Parse([]string{"--repo", "gsimone/leva-2", "--provider", "codex@luna.medium"})
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != "gsimone/leva-2" {
		t.Fatalf("repo %q", args.Repo)
	}
	if args.Provider.Raw != "codex@luna.medium" || args.Provider.Name != "codex" || args.Provider.Model != "luna.medium" {
		t.Fatalf("provider %+v", args.Provider)
	}
}

func TestParseAcceptsShortRepoFlag(t *testing.T) {
	args, err := Parse([]string{"-repo", "gsimone/leva-2"})
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != "gsimone/leva-2" {
		t.Fatalf("repo %q", args.Repo)
	}
}

func TestParseNormalizesGitHubURL(t *testing.T) {
	args, err := Parse([]string{"--repo", "https://github.com/gsimone/leva-2.git"})
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != "gsimone/leva-2" {
		t.Fatalf("repo %q", args.Repo)
	}
}

func TestParseStoryHookStaysHidden(t *testing.T) {
	args, err := Parse([]string{"-story", "mixed", "--repo", "gsimone/leva-2"})
	if err != nil {
		t.Fatal(err)
	}
	if args.Story != "mixed" {
		t.Fatalf("hidden hook still works, got %q", args.Story)
	}
	if args.Repo != "" {
		t.Fatalf("story hook ignores --repo, got %q", args.Repo)
	}
}

func TestParseRepoJSONFile(t *testing.T) {
	args, err := Parse([]string{"--repo", "testdata/test.json"})
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != "testdata/test.json" {
		t.Fatalf("file path must stay a path, got %q", args.Repo)
	}
	args, err = Parse([]string{"-repo", "test.json"})
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != "test.json" {
		t.Fatalf("bare json: %q", args.Repo)
	}
}

func TestParseNeitherIsExamples(t *testing.T) {
	args, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != "" || args.Story != "" {
		t.Fatalf("no --repo is examples, got %+v", args)
	}
}

func TestParseRepoDoesNotRequireProvider(t *testing.T) {
	args, err := Parse([]string{"--repo", "owner/name"})
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != "owner/name" {
		t.Fatalf("repo %q", args.Repo)
	}
	if args.Provider.Raw != "" {
		t.Fatalf("provider must stay optional, got %+v", args.Provider)
	}
}

func TestParseRejectsBareName(t *testing.T) {
	_, err := Parse([]string{"--repo", "leva-2"})
	if err == nil || !strings.Contains(err.Error(), "owner/name") {
		t.Fatalf("want owner/name error, got %v", err)
	}
}

func TestParseProviderWithoutAt(t *testing.T) {
	p := ParseProvider("codex")
	if p.Name != "codex" || p.Model != "" || p.Raw != "codex" {
		t.Fatalf("%+v", p)
	}
}

func TestParseHelp(t *testing.T) {
	var buf strings.Builder
	_, err := parse([]string{"-h"}, &buf)
	if err != flag.ErrHelp {
		t.Fatalf("help: %v", err)
	}
	got := buf.String()
	if strings.Contains(got, "-story") || strings.Contains(got, "dango -story") {
		t.Fatalf("must not advertise -story:\n%s", got)
	}
	if !strings.Contains(got, "No --repo: authored example stacks") {
		t.Fatalf("advertised path:\n%s", got)
	}
	if !strings.Contains(got, "--repo owner/name") {
		t.Fatalf("live path:\n%s", got)
	}
}

func TestUsageDoesNotAdvertiseStory(t *testing.T) {
	got := Usage()
	if strings.Contains(got, "-story") || strings.Contains(got, "story") {
		t.Fatalf("usage must not sell -story:\n%s", got)
	}
	if strings.Contains(got, "detect") {
		t.Fatalf("no --repo is examples, not git detect:\n%s", got)
	}
}
