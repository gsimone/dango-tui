package summary

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gsimone/dango-tui/internal/domain"
)

const (
	lunaModelDefault = "gpt-5.6"
	lunaURLDefault   = "https://api.openai.com/v1/chat/completions"
	lunaTimeout      = 8 * time.Second
)

var errNoLuna = errors.New("dango: no luna key")

// describeRemote is the network hook. Production is lunaDescribe.
// Tests replace it. Empty or error → Run uses Describe().
var describeRemote = lunaDescribe

type lunaPoster func(endpoint, key string, body []byte) ([]byte, error)

func lunaKey() string {
	for _, name := range []string{"DANGO_OPENAI_API_KEY", "OPENAI_API_KEY", "DANGO_API_KEY"} {
		if v := strings.TrimSpace(os.Getenv(name)); v != "" {
			return v
		}
	}
	return ""
}

func lunaModel() string {
	if v := strings.TrimSpace(os.Getenv("DANGO_LUNA_MODEL")); v != "" {
		return v
	}
	return lunaModelDefault
}

func lunaURL() string {
	if v := strings.TrimSpace(os.Getenv("DANGO_LUNA_URL")); v != "" {
		return v
	}
	return lunaURLDefault
}

func lunaDescribe(stack domain.Stack) (string, error) {
	key := lunaKey()
	if key == "" {
		return "", errNoLuna
	}
	return describeWith(postJSON, key, lunaModel(), lunaURL(), stack)
}

func describeWith(post lunaPoster, key, model, endpoint string, stack domain.Stack) (string, error) {
	if post == nil || strings.TrimSpace(key) == "" {
		return "", errNoLuna
	}
	if strings.TrimSpace(model) == "" {
		model = lunaModelDefault
	}
	if strings.TrimSpace(endpoint) == "" {
		endpoint = lunaURLDefault
	}
	prompt := lunaPrompt(stack)
	if prompt == "" {
		return "", errNoLuna
	}
	reqBody, err := json.Marshal(map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": lunaSystem},
			{"role": "user", "content": prompt},
		},
		"max_completion_tokens": 80,
	})
	if err != nil {
		return "", err
	}
	raw, err := post(endpoint, key, reqBody)
	if err != nil || len(bytes.TrimSpace(raw)) == 0 {
		return "", err
	}
	text, err := parseLuna(raw)
	if err != nil {
		return "", err
	}
	return cleanLuna(text, stack), nil
}

const lunaSystem = "Write one short inspector blurb (at most two sentences) for this GitHub pull-request stack. Plain prose. No heading. Do not start with Covers. Do not mention CURSOR_AGENT. Do not paste a PR body."

func lunaPrompt(stack domain.Stack) string {
	var b strings.Builder
	b.WriteString("Layers:\n")
	n := 0
	for _, pr := range stack.PRs {
		t := strings.TrimSpace(pr.Title)
		if t == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(t)
		b.WriteByte('\n')
		n++
	}
	if n == 0 {
		return ""
	}
	return b.String()
}

type lunaChatResp struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func parseLuna(raw []byte) (string, error) {
	var resp lunaChatResp
	if err := json.Unmarshal(raw, &resp); err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("dango: empty luna choices")
	}
	return resp.Choices[0].Message.Content, nil
}

func cleanLuna(raw string, stack domain.Stack) string {
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
	return s
}

func postJSON(endpoint, key string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: lunaTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("dango: luna http " + resp.Status)
	}
	return buf.Bytes(), nil
}
