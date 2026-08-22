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

func TestParseStoryIgnoresRepo(t *testing.T) {
	args, err := Parse([]string{"-story", "mixed", "--repo", "gsimone/leva-2"})
	if err != nil {
		t.Fatal(err)
	}
	if args.Story != "mixed" {
		t.Fatalf("story %q", args.Story)
	}
	if args.Repo != "" {
		t.Fatalf("story must ignore live repo, got %q", args.Repo)
	}
}

func TestParseNeitherIsFixturePath(t *testing.T) {
	args, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if args.Repo != "" || args.Story != "" {
		t.Fatalf("no --repo means fixtures, got %+v", args)
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
	_, err := Parse([]string{"-h"})
	if err != flag.ErrHelp {
		t.Fatalf("help: %v", err)
	}
}
