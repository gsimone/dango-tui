package summary

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gsimone/dango-tui/internal/domain"
)

const describeTimeout = 8 * time.Second

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
	argv := describeArgv(job.Describe)
	if len(argv) == 0 {
		return "", errNoDescribe
	}
	stdin := describePayload(job.Stack)
	if len(stdin) == 0 || len(layerTitleList(job.Stack)) == 0 {
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
	if testing.Testing() {
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
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stderr = io.Discard
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(out)), nil
}
