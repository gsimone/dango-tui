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

// Config is dango.json / dango.yml / dango.yaml. Missing file means no generated title.
type Config struct {
	Provider string `json:"provider"`
}

var configNames = []string{"dango.json", "dango.yml", "dango.yaml"}

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

func configRoots(dir string) []string {
	seen := map[string]bool{}
	var roots []string
	add := func(path string) {
		if path == "" {
			return
		}
		if seen[path] {
			return
		}
		seen[path] = true
		roots = append(roots, path)
	}
	if found, err := GitRoot(dir); err == nil && found != "" {
		add(found)
	}
	add(dir)
	return roots
}

func parseDangoYAML(raw []byte) (Config, error) {
	var cfg Config
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "provider" {
			continue
		}
		val = strings.TrimSpace(val)
		if i := strings.Index(val, " #"); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}
		val = strings.Trim(val, `"'`)
		cfg.Provider = val
	}
	return cfg, nil
}

func parseDangoConfig(name string, raw []byte) (Config, error) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".yml", ".yaml":
		cfg, err := parseDangoYAML(raw)
		if err != nil {
			return Config{}, fmt.Errorf("%s: %w", name, err)
		}
		return cfg, nil
	default:
		var cfg Config
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return Config{}, fmt.Errorf("%s: %w", name, err)
		}
		return cfg, nil
	}
}

// ReadDangoConfig loads provider config from dango.json, dango.yml, or
// dango.yaml. Looks in the git root, then cwd. Missing file is empty.
func ReadDangoConfig(dir string) (Config, error) {
	for _, root := range configRoots(dir) {
		for _, name := range configNames {
			raw, err := os.ReadFile(filepath.Join(root, name))
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return Config{}, err
			}
			cfg, err := parseDangoConfig(name, raw)
			if err != nil {
				return Config{}, err
			}
			cfg.Provider = strings.TrimSpace(cfg.Provider)
			return cfg, nil
		}
	}
	return Config{}, nil
}

// ReadDangoJSON loads provider config. Prefer ReadDangoConfig.
func ReadDangoJSON(dir string) (Config, error) {
	return ReadDangoConfig(dir)
}

// Resolve fills repo from the cwd git remote when --repo is omitted, and
// provider from dango.json / dango.yml / dango.yaml when --provider is
// omitted. --repo (owner/name or a stack file) and --provider win.
// Story is a test/dev hook and skips detect. Detect failure leaves Repo
// empty so the TUI loads authored examples.
func Resolve(args Args, dir string) Args {
	if args.Story != "" {
		return args
	}
	if args.Repo == "" {
		if repo, err := DetectRepo(dir); err == nil {
			args.Repo = repo
		}
	}
	if args.Provider.Empty() {
		if cfg, err := ReadDangoConfig(dir); err == nil && cfg.Provider != "" {
			args.Provider = ParseProvider(cfg.Provider)
		}
	}
	return args
}
