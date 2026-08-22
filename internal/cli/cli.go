package cli

import (
	"flag"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/summary"
)

// IsStackFile reports whether --repo is a JSON stack dump, not owner/name.
func IsStackFile(raw string) bool {
	return data.IsStackFile(raw)
}

// Args is the flag-driven launch config. --repo is owner/name (live gh) or a
// JSON stack file. -story loads authored fixtures and ignores live fetch.
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
	story := fs.String("story", "", "authored fixture: mixed, freight, pair (chaos is stress only)")
	repo := fs.String("repo", "", "owner/name (live gh) or a JSON file of authored stacks")
	provider := fs.String("provider", "", "stack title summarizer (e.g. codex@luna.medium); optional, does not block fetch")
	fs.Usage = func() {
		fmt.Fprintln(usage, "Usage: dango")
		fmt.Fprintln(usage, "       dango --repo owner/name [--provider name@model]")
		fmt.Fprintln(usage, "       dango --repo testdata/test.json")
		fmt.Fprintln(usage, "       dango -story mixed")
		fmt.Fprintln(usage, "       dango -story freight")
		fmt.Fprintln(usage, "       dango -story pair")
		fmt.Fprintln(usage, "")
		fmt.Fprintln(usage, "With no flags, owner/name comes from git remote of cwd.")
		fmt.Fprintln(usage, "dango.json / dango.yml / dango.yaml sets the title provider. Missing file = no generated title.")
		fmt.Fprintln(usage, "--repo is live gh or a stack dump. -story loads fixtures and ignores fetch.")
		fmt.Fprintln(usage, "--provider overrides the config file. No picker.")
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
	rawRepo := strings.TrimSpace(*repo)
	if rawRepo == "" {
		return out, nil
	}
	if IsStackFile(rawRepo) {
		out.Repo = rawRepo
		return out, nil
	}
	normalized, err := NormalizeRepo(rawRepo)
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
