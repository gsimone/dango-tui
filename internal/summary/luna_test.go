package summary

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

func sampleStack() domain.Stack {
	return domain.Stack{
		ID: "s",
		PRs: []domain.PullRequest{
			{Number: 182, Title: "LEV-182: Bound hosts to the session"},
			{Number: 183, Title: "Pin each host to the worker"},
		},
	}
}

func TestRunPrefersLunaDescription(t *testing.T) {
	stack := sampleStack()
	old := describeRemote
	describeRemote = func(domain.Stack) (string, error) {
		return "luna bound hosts to the worker so undo cannot wedge", nil
	}
	t.Cleanup(func() { describeRemote = old })

	res := Run(Job{ID: "s", Stack: stack})
	if res.Title != "" {
		t.Fatalf("no provider must not retitle: %+v", res)
	}
	if res.Description != "luna bound hosts to the worker so undo cannot wedge" {
		t.Fatalf("luna description: %q", res.Description)
	}
	if res.Description == Describe(stack) {
		t.Fatal("product sentence is luna, not Describe()")
	}
}

func TestRunFallsBackToDescribe(t *testing.T) {
	stack := sampleStack()
	old := describeRemote
	describeRemote = func(domain.Stack) (string, error) {
		return "", errors.New("no key")
	}
	t.Cleanup(func() { describeRemote = old })

	res := Run(Job{ID: "s", Stack: stack})
	if res.Title != "" {
		t.Fatalf("fallback must not retitle: %+v", res)
	}
	if res.Description != Describe(stack) {
		t.Fatalf("Describe() only on failure: got %q want %q", res.Description, Describe(stack))
	}

	describeRemote = func(domain.Stack) (string, error) { return "", nil }
	empty := Run(Job{ID: "s", Stack: stack})
	if empty.Description != Describe(stack) {
		t.Fatalf("empty luna uses Describe(): %q", empty.Description)
	}

	describeRemote = func(domain.Stack) (string, error) { return "   ", nil }
	blank := Run(Job{ID: "s", Stack: stack})
	if blank.Description != Describe(stack) {
		t.Fatalf("whitespace luna uses Describe(): %q", blank.Description)
	}
}

func TestRunProviderTitleKeepsLunaDescription(t *testing.T) {
	stack := sampleStack()
	old := describeRemote
	describeRemote = func(domain.Stack) (string, error) {
		return "luna bound hosts to the worker so undo cannot wedge", nil
	}
	t.Cleanup(func() { describeRemote = old })

	res := Run(Job{Provider: ParseProvider("codex@luna.medium"), ID: "s", Stack: stack})
	if res.Title == "" || res.Title == stack.PRs[0].Title {
		t.Fatalf("provider may still swap a title: %+v", res)
	}
	if res.Description != "luna bound hosts to the worker so undo cannot wedge" {
		t.Fatalf("luna still writes the pane: %q", res.Description)
	}
	if res.Description == Describe(stack) {
		t.Fatal("Describe() is fallback only")
	}
}

func TestLunaKeyOrder(t *testing.T) {
	t.Setenv("DANGO_OPENAI_API_KEY", "dango-key")
	t.Setenv("OPENAI_API_KEY", "openai-key")
	if got := lunaKey(); got != "dango-key" {
		t.Fatalf("prefer DANGO_OPENAI_API_KEY, got %q", got)
	}
	t.Setenv("DANGO_OPENAI_API_KEY", "")
	if got := lunaKey(); got != "openai-key" {
		t.Fatalf("OPENAI_API_KEY, got %q", got)
	}
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("DANGO_API_KEY", "legacy")
	if got := lunaKey(); got != "legacy" {
		t.Fatalf("DANGO_API_KEY, got %q", got)
	}
}

func TestLunaDescribeSkipsWithoutKey(t *testing.T) {
	t.Setenv("DANGO_OPENAI_API_KEY", "")
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("DANGO_API_KEY", "")
	got, err := lunaDescribe(sampleStack())
	if got != "" || !errors.Is(err, errNoLuna) {
		t.Fatalf("no key must skip network: %q %v", got, err)
	}
}

func TestDescribeWithParsesChatAndStripsJunk(t *testing.T) {
	stack := sampleStack()
	var sawURL, sawKey string
	var sawBody []byte
	post := func(endpoint, key string, body []byte) ([]byte, error) {
		sawURL, sawKey, sawBody = endpoint, key, append([]byte(nil), body...)
		return []byte(`{"choices":[{"message":{"content":"Hosts stay pinned so undo cannot widen scope."}}]}`), nil
	}
	got, err := describeWith(post, "sk-test", "gpt-5.6", "https://api.openai.com/v1/chat/completions", stack)
	if err != nil {
		t.Fatal(err)
	}
	if got != "Hosts stay pinned so undo cannot widen scope." {
		t.Fatalf("parsed %q", got)
	}
	if sawURL != lunaURLDefault || sawKey != "sk-test" {
		t.Fatalf("post %s key %s", sawURL, sawKey)
	}
	var payload map[string]any
	if err := json.Unmarshal(sawBody, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["model"] != "gpt-5.6" {
		t.Fatalf("model %v", payload["model"])
	}
	var payloadMsgs []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	rawMsgs, _ := json.Marshal(payload["messages"])
	if err := json.Unmarshal(rawMsgs, &payloadMsgs); err != nil {
		t.Fatal(err)
	}
	user := ""
	for _, msg := range payloadMsgs {
		if msg.Role == "user" {
			user = msg.Content
		}
	}
	if strings.Contains(user, "CURSOR_AGENT") || strings.Contains(user, "Pin each bound host to the worker") {
		t.Fatalf("must not send body dump: %s", user)
	}
	if !strings.Contains(user, "LEV-182") || !strings.Contains(user, "Pin each host to the worker") {
		t.Fatalf("prompt uses layer titles: %s", user)
	}

	junk, err := describeWith(func(string, string, []byte) ([]byte, error) {
		return []byte(`{"choices":[{"message":{"content":"Covers the bound hosts"}}]}`), nil
	}, "sk-test", "gpt-5.6", lunaURLDefault, stack)
	if err != nil {
		t.Fatal(err)
	}
	if junk != "" {
		t.Fatalf("Covers wrapper must be dropped: %q", junk)
	}

	fail, err := describeWith(func(string, string, []byte) ([]byte, error) {
		return nil, errors.New("502")
	}, "sk-test", "gpt-5.6", lunaURLDefault, stack)
	if fail != "" || err == nil {
		t.Fatalf("http error must fail soft: %q %v", fail, err)
	}

	echo, err := describeWith(func(string, string, []byte) ([]byte, error) {
		return []byte(`{"choices":[{"message":{"content":"LEV-182: Bound hosts to the session"}}]}`), nil
	}, "sk-test", "gpt-5.6", lunaURLDefault, stack)
	if err != nil {
		t.Fatal(err)
	}
	if echo != "" {
		t.Fatalf("gh title echo must be dropped: %q", echo)
	}
}

func TestLunaPromptOmitsBody(t *testing.T) {
	stack := sampleStack()
	stack.PRs[0].Body = "<!-- CURSOR_AGENT_PR_BODY_BEGIN -->secret body"
	got := lunaPrompt(stack)
	if strings.Contains(got, "CURSOR_AGENT") || strings.Contains(got, "secret body") {
		t.Fatalf("prompt leaked body: %q", got)
	}
}
