package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Config is dango.json in a repo. Missing file means no generated title.
type Config struct {
	Provider string `json:"provider"`
}

func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return bytes.TrimSpace(out), nil
}

func GitRoot(dir string) (string, error) {
	out, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// DetectRepo reads owner/name from the cwd git remote. No baked-in default.
func DetectRepo(dir string) (string, error) {
	raw, err := runGit(dir, "remote", "get-url", "origin")
	if err != nil {
		listed, listErr := runGit(dir, "remote", "-v")
		if listErr != nil {
			return "", err
		}
		raw = firstRemoteURL(listed)
	}
	return ParseRemote(string(raw))
}

func firstRemoteURL(listed []byte) []byte {
	for _, line := range strings.Split(string(listed), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			return []byte(fields[1])
		}
	}
	return nil
}

func ParseRemote(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("no git remote")
	}
	raw = strings.TrimSuffix(raw, ".git")
	if strings.HasPrefix(raw, "git@") {
		_, path, ok := strings.Cut(raw, ":")
		if !ok || path == "" {
			return "", fmt.Errorf("repo must look like owner/name")
		}
		return NormalizeRepo(path)
	}
	return NormalizeRepo(raw)
}

// ReadDangoJSON loads provider config from the repo root. Missing file is empty.
func ReadDangoJSON(dir string) (Config, error) {
	root := dir
	if found, err := GitRoot(dir); err == nil && found != "" {
		root = found
	}
	raw, err := os.ReadFile(filepath.Join(root, "dango.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, fmt.Errorf("dango.json: %w", err)
	}
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	return cfg, nil
}

// Resolve fills repo from git remote and provider from dango.json when flags
// are omitted. --repo and --provider win. -story stays fixtures.
func Resolve(args Args, dir string) Args {
	if args.Story != "" {
		return args
	}
	if args.Repo == "" {
		if repo, err := DetectRepo(dir); err == nil {
			args.Repo = repo
		}
	}
	if args.Provider.Raw == "" && args.Provider.Name == "" {
		if cfg, err := ReadDangoJSON(dir); err == nil && cfg.Provider != "" {
			args.Provider = ParseProvider(cfg.Provider)
		}
	}
	return args
}
