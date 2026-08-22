package cli

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/gsimone/dango-tui/internal/summary"
)

// Args is the flag-driven launch config. --repo fetches via gh. No repo or -story is fixtures.
type Args struct {
	Frame    string
	Story    string
	Repo     string
	Provider summary.Provider
}

// Provider is the --provider flag. The summarizer owns the type.
type Provider = summary.Provider

func ParseProvider(raw string) Provider {
	return summary.ParseProvider(raw)
}

func Parse(args []string) (Args, error) {
	return parse(args, io.Discard)
}

func parse(args []string, usage io.Writer) (Args, error) {
	fs := flag.NewFlagSet("dango", flag.ContinueOnError)
	fs.SetOutput(usage)
	frame := fs.String("frame", "", "print one frame (WxH, e.g. 80x24) and exit")
	story := fs.String("story", "", "fixture story id; ignores live fetch")
	repo := fs.String("repo", "", "GitHub repo as owner/name; fetch live PRs via gh")
	provider := fs.String("provider", "", "summarizer provider (e.g. codex@luna.medium); optional, does not block fetch")
	fs.Usage = func() {
		fmt.Fprintln(usage, "Usage: dango --repo owner/name [--provider name@model]")
		fmt.Fprintln(usage, "       dango [-story mixed]")
		fmt.Fprintln(usage, "")
		fmt.Fprintln(usage, "Live fetch requires --repo owner/name. No baked-in repo.")
		fmt.Fprintln(usage, "--provider is optional and is not required to fetch.")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return Args{}, err
	}

	out := Args{
		Frame:    strings.TrimSpace(*frame),
		Story:    strings.TrimSpace(*story),
		Provider: ParseProvider(*provider),
	}
	if out.Story != "" {
		return out, nil
	}
	if strings.TrimSpace(*repo) == "" {
		return out, nil
	}
	normalized, err := NormalizeRepo(*repo)
	if err != nil {
		return Args{}, err
	}
	out.Repo = normalized
	return out, nil
}

func NormalizeRepo(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("pass --repo owner/name")
	}
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Path == "" {
			return "", fmt.Errorf("repo must look like owner/name")
		}
		raw = strings.TrimPrefix(u.Path, "/")
	}
	raw = strings.TrimSuffix(raw, ".git")
	raw = strings.TrimPrefix(raw, "github.com/")
	parts := strings.Split(raw, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("repo must look like owner/name")
	}
	if strings.Contains(parts[0], " ") || strings.Contains(parts[1], " ") {
		return "", fmt.Errorf("repo must look like owner/name")
	}
	return parts[0] + "/" + parts[1], nil
}
