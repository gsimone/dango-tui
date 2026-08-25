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

// Args is the flag-driven launch config. No --repo detects the cwd git
// remote and fetches live gh. Detect failure is an error, not examples.
// --repo archetype-labs/app is live gh. --repo path.json is a stack dump.
type Args struct {
	Frame       string
	Story       string // test/dev hook only; not advertised
	Repo        string
	Provider    summary.Provider
	Describe    string
	DescribeDir string // directory of the config file that set Describe
	Doctor      bool
}

// Provider is the --provider flag. The summarizer owns the type.
type Provider = summary.Provider

func ParseProvider(raw string) Provider {
	return summary.ParseProvider(raw)
}

// Usage is the advertised help. -story is a hidden test/dev hook and
// must not appear here.
func Usage() string {
	return "Usage: dango\n" +
		"       dango --repo archetype-labs/app [--provider name@model]\n" +
		"       dango --repo testdata/test.json\n" +
		"       dango --doctor\n" +
		"\n" +
		"No --repo: detect the GitHub remote from cwd and fetch via gh.\n" +
		"No GitHub remote: pass --repo archetype-labs/app or --repo testdata/test.json.\n" +
		"--repo archetype-labs/app fetches via gh. --repo path.json is a stack dump, not live gh.\n" +
		"--doctor prints cwd / looked / won / describe on stdout and exits. No TUI.\n" +
		"If cwd has no dango.json / dango.yml / dango.yaml it writes cwd/dango.json without a describe key.\n" +
		"dango.json / dango.yml / dango.yaml may set provider and describe.\n" +
		"Missing config file = no generated title. Missing describe = empty inspector pane.\n" +
		"--provider overrides the title hook. No picker.\n"
}

func Parse(args []string) (Args, error) {
	return parse(args, io.Discard)
}

func parse(args []string, usage io.Writer) (Args, error) {
	fs := flag.NewFlagSet("dango", flag.ContinueOnError)
	fs.SetOutput(usage)
	frame := fs.String("frame", "", "print one frame (WxH, e.g. 80x24) and exit")
	story := fs.String("story", "", "")
	repo := fs.String("repo", "", "archetype-labs/app (live gh) or a JSON file of authored stacks")
	provider := fs.String("provider", "", "stack title summarizer (e.g. name@model); optional, does not block fetch")
	doctor := fs.Bool("doctor", false, "print config lookup and write cwd/dango.json if missing")
	fs.Usage = func() {
		fmt.Fprint(usage, Usage())
	}
	if err := fs.Parse(args); err != nil {
		return Args{}, err
	}

	out := Args{
		Frame:    strings.TrimSpace(*frame),
		Story:    strings.TrimSpace(*story),
		Provider: ParseProvider(*provider),
		Doctor:   *doctor,
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
		return "", fmt.Errorf("pass --repo archetype-labs/app")
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
