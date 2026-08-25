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

// LaunchDir is the real process cwd. Resolve uses this, not ".".
func LaunchDir() string {
	cwd, err := os.Getwd()
	if err != nil || strings.TrimSpace(cwd) == "" {
		return "."
	}
	return cwd
}

// ResolveLaunch is what `dango --repo` does: detect/config from Getwd.
func ResolveLaunch(args Args) (Args, error) {
	return Resolve(args, LaunchDir())
}

func loadConfigFile(path, name string) (Config, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, false, nil
		}
		return Config{}, false, err
	}
	cfg, err := parseDangoConfig(name, raw)
	if err != nil {
		return Config{}, false, err
	}
	cfg.Provider = strings.TrimSpace(cfg.Provider)
	cfg.Describe = strings.TrimSpace(cfg.Describe)
	return cfg, true, nil
}

// ReadDangoConfig loads provider / describe from dango.json, dango.yml,
// or dango.yaml. A dango.json in dir (launch cwd) wins and stops the
// walk — even when that file has no describe key. Only a missing cwd
// dango.json may look at cwd yml/yaml, then the git root. --repo
// owner/name is not a remote file. Missing file is empty.
func ReadDangoConfig(dir string) (Config, error) {
	cfg, _, err := ReadDangoConfigAt(dir)
	return cfg, err
}

func configSearchPaths(dir string) []string {
	dir = filepath.Clean(dir)
	var paths []string
	for _, name := range configNames {
		paths = append(paths, filepath.Join(dir, name))
	}
	root, err := GitRoot(dir)
	if err != nil || root == "" {
		return paths
	}
	if filepath.Clean(root) == dir {
		return paths
	}
	for _, name := range configNames {
		paths = append(paths, filepath.Join(root, name))
	}
	return paths
}

func cwdHasConfigFile(dir string) bool {
	dir = filepath.Clean(dir)
	for _, name := range configNames {
		st, err := os.Stat(filepath.Join(dir, name))
		if err == nil && !st.IsDir() {
			return true
		}
	}
	return false
}

// ReadDangoConfigAt is ReadDangoConfig plus the file path that won.
func ReadDangoConfigAt(dir string) (Config, string, error) {
	dir = filepath.Clean(dir)
	path := filepath.Join(dir, "dango.json")
	if cfg, ok, err := loadConfigFile(path, "dango.json"); ok || err != nil {
		return cfg, path, err
	}
	for _, name := range []string{"dango.yml", "dango.yaml"} {
		path = filepath.Join(dir, name)
		if cfg, ok, err := loadConfigFile(path, name); ok || err != nil {
			return cfg, path, err
		}
	}
	root, err := GitRoot(dir)
	if err != nil || root == "" {
		return Config{}, "", nil
	}
	if filepath.Clean(root) == dir {
		return Config{}, "", nil
	}
	for _, name := range configNames {
		path = filepath.Join(root, name)
		if cfg, ok, err := loadConfigFile(path, name); ok || err != nil {
			return cfg, path, err
		}
	}
	return Config{}, "", nil
}

// resolveDescribe rewrites a relative argv[0] against the config file dir.
// Bare PATH commands such as echo stay as-is unless that name exists there.
func resolveDescribe(raw, configPath string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.TrimSpace(configPath) == "" {
		return raw
	}
	dir := filepath.Dir(configPath)
	fields := strings.Fields(raw)
	if len(fields) == 0 {
		return raw
	}
	name := fields[0]
	if filepath.IsAbs(name) {
		return raw
	}
	if strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		fields[0] = filepath.Join(dir, name)
		return strings.Join(fields, " ")
	}
	cand := filepath.Join(dir, name)
	if st, err := os.Stat(cand); err == nil && !st.IsDir() {
		fields[0] = cand
		return strings.Join(fields, " ")
	}
	return raw
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
// (then that dir's git root). A cwd dango.json stops the walk.
// --repo owner/name is live gh, not a remote config file, and does not
// drop the launch-dir config. --provider overrides the title hook.
// describe comes only from the config file. Story is a test/dev hook
// and skips detect. Detect failure is ErrNoRemote — never a silent
// examples fallback.
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
	args.DescribeDir = ""
	cfg, path, err := ReadDangoConfigAt(dir)
	if err == nil {
		if args.Provider.Empty() && cfg.Provider != "" {
			args.Provider = ParseProvider(cfg.Provider)
		}
		args.Describe = resolveDescribe(cfg.Describe, path)
		if args.Describe != "" && path != "" {
			args.DescribeDir = filepath.Dir(path)
		}
	}
	return args, nil
}
