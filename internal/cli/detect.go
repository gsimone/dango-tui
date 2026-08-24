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
	Describe string `json:"describe"`
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
	add(dir)
	if found, err := GitRoot(dir); err == nil && found != "" {
		add(found)
	}
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
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if i := strings.Index(val, " #"); i >= 0 {
			val = strings.TrimSpace(val[:i])
		}
		val = strings.Trim(val, `"'`)
		switch key {
		case "provider":
			cfg.Provider = val
		case "describe":
			cfg.Describe = val
		}
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

// ReadDangoConfig loads provider / describe from dango.json, dango.yml,
// or dango.yaml. Looks in dir (launch cwd), then the git root of dir.
// --repo owner/name is not a remote file. Missing file is empty.
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
			cfg.Describe = strings.TrimSpace(cfg.Describe)
			return cfg, nil
		}
	}
	return Config{}, nil
}

// ReadDangoJSON loads provider config. Prefer ReadDangoConfig.
func ReadDangoJSON(dir string) (Config, error) {
	return ReadDangoConfig(dir)
}

// ErrNoRemote is the loud failure when no --repo is set and cwd has no
// GitHub owner/name remote. It names the two ways out.
var ErrNoRemote = fmt.Errorf("no GitHub remote in this directory. Pass --repo archetype-labs/app or --repo testdata/test.json")

// Resolve fills repo from the cwd git remote when --repo is omitted, and
// provider / describe from dango.json / dango.yml / dango.yaml in dir
// (then that dir's git root). --repo owner/name is live gh, not a
// remote config file. --provider overrides the title hook. describe
// comes only from the config file. Story is a test/dev hook and skips
// detect. Detect failure is ErrNoRemote — never a silent examples fallback.
func Resolve(args Args, dir string) (Args, error) {
	if args.Story != "" {
		return args, nil
	}
	if args.Repo == "" {
		repo, err := DetectRepo(dir)
		if err != nil || repo == "" {
			return args, ErrNoRemote
		}
		args.Repo = repo
	}
	args.Describe = ""
	cfg, err := ReadDangoConfig(dir)
	if err == nil {
		if args.Provider.Empty() && cfg.Provider != "" {
			args.Provider = ParseProvider(cfg.Provider)
		}
		args.Describe = cfg.Describe
	}
	return args, nil
}
