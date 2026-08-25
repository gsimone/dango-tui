package summary

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gsimone/dango-tui/internal/domain"
)

const describeTimeout = 45 * time.Second

var errNoDescribe = errors.New("dango: no describe command")

type describeRunner func(ctx context.Context, argv []string, stdin []byte) (string, error)

// runDescribe is the script runner. Production is execDescribe. Tests inject a fake.
var runDescribe describeRunner = execDescribe

type describeInput struct {
	Titles []string `json:"titles"`
}

func describeArgv(raw string) []string {
	return strings.Fields(strings.TrimSpace(raw))
}

// resolveDescribeArgv joins a relative argv[0] to configDir. echo stays on PATH.
func resolveDescribeArgv(argv []string, configDir string) []string {
	if len(argv) == 0 {
		return argv
	}
	out := append([]string(nil), argv...)
	name := out[0]
	if filepath.IsAbs(name) || strings.TrimSpace(configDir) == "" {
		return out
	}
	if strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		out[0] = filepath.Join(configDir, name)
		return out
	}
	cand := filepath.Join(configDir, name)
	if st, err := os.Stat(cand); err == nil && !st.IsDir() {
		out[0] = cand
	}
	return out
}

func describePayload(stack domain.Stack) []byte {
	in := describeInput{Titles: layerTitleList(stack)}
	raw, err := json.Marshal(in)
	if err != nil {
		return nil
	}
	return raw
}

func layerTitleList(stack domain.Stack) []string {
	var out []string
	for _, pr := range stack.PRs {
		t := strings.TrimSpace(pr.Title)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}

func describeScript(job Job) (string, error) {
	argv := resolveDescribeArgv(describeArgv(job.Describe), job.DescribeDir)
	if len(argv) == 0 {
		return "", errNoDescribe
	}
	stdin := describePayload(job.Stack)
	if len(stdin) == 0 {
		return "", errNoDescribe
	}
	ctx, cancel := context.WithTimeout(context.Background(), describeTimeout)
	defer cancel()
	out, err := runDescribe(ctx, argv, stdin)
	if err != nil {
		return "", err
	}
	return cleanDescribe(out, job.Stack), nil
}

func cleanDescribe(raw string, stack domain.Stack) string {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, "\"'`")
	if s == "" || strings.Contains(s, "CURSOR_AGENT") || strings.HasPrefix(s, "Covers ") {
		return ""
	}
	var kept []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		kept = append(kept, line)
		if len(kept) == 2 {
			break
		}
	}
	s = strings.Join(kept, " ")
	if sameFold(s, ghName(stack)) {
		return ""
	}
	if clause := clauseFromLayers(stack); clause != "" && sameFold(s, clause) {
		return ""
	}
	if d := Describe(stack); d != "" && sameFold(s, d) {
		return ""
	}
	return s
}

func execDescribe(ctx context.Context, argv []string, stdin []byte) (string, error) {
	if testing.Testing() && !allowTestDescribe(argv) {
		return "", errNoDescribe
	}
	if len(argv) == 0 {
		return "", errNoDescribe
	}
	name := argv[0]
	if strings.ContainsAny(name, `/\`) {
		if _, err := os.Stat(name); err != nil {
			return "", errNoDescribe
		}
	} else if _, err := exec.LookPath(name); err != nil {
		return "", errNoDescribe
	}
	cmd := exec.CommandContext(ctx, name, argv[1:]...)
	if filepath.IsAbs(name) {
		cmd.Dir = filepath.Dir(name)
	}
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stderr = io.Discard
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(out)), nil
}

func allowTestDescribe(argv []string) bool {
	if len(argv) == 0 {
		return false
	}
	if argv[0] == "echo" {
		return true
	}
	// Testdata fixtures only. Never a PATH lookup or the in-repo example script.
	slash := filepath.ToSlash(argv[0])
	return strings.Contains(slash, "/testdata/") || strings.HasPrefix(slash, "testdata/")
}
