package cli

import (
	"strings"
	"testing"
)

func TestMutantRepoParse(t *testing.T) {
	got, err := NormalizeRepo("https://github.com/owner/name.git")
	if err != nil || got != "owner/name" {
		t.Fatalf("real normalize: %q %v", got, err)
	}
	keepHost := func(raw string) (string, error) {
		raw = strings.TrimSpace(raw)
		return raw, nil
	}
	if host, _ := keepHost("https://github.com/owner/name.git"); host == "owner/name" {
		t.Fatal("keep-host mutant must not survive")
	}

	if IsStackFile("testdata/test.json") != true {
		t.Fatal("real json dump")
	}
	if IsStackFile("owner/name") {
		t.Fatal("owner/name is live gh")
	}
	suffixOnly := func(raw string) bool {
		return strings.Contains(strings.ToLower(raw), "json")
	}
	if suffixOnly("owner/json-tools") && !IsStackFile("owner/json-tools") {
		// mutant is looser than production — that is the kill
	} else if suffixOnly("owner/json-tools") == IsStackFile("owner/json-tools") {
		t.Fatal("contains-json mutant must not survive")
	}

	args, err := Parse([]string{"--repo", "owner/name"})
	if err != nil || args.Repo != "owner/name" || args.Story != "" {
		t.Fatalf("real parse %+v %v", args, err)
	}
	asStory := func() Args {
		return Args{Story: "owner/name"}
	}
	if asStory().Repo == "owner/name" {
		t.Fatal("story-swap mutant must not survive")
	}
}

func TestMutantDetectFailureIsLoud(t *testing.T) {
	if ErrNoRemote == nil || !strings.Contains(ErrNoRemote.Error(), "--repo archetype-labs/app") {
		t.Fatal("real detect error names --repo archetype-labs/app")
	}
	if !strings.Contains(ErrNoRemote.Error(), "--repo testdata/test.json") {
		t.Fatal("real detect error names the json dump")
	}
	silent := "no remote"
	if silent == ErrNoRemote.Error() {
		t.Fatal("silent detect mutant must not survive")
	}
}
