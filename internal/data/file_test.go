package data_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/data"
	"github.com/gsimone/dango-tui/internal/domain"
)

func testdataJSON(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		p := filepath.Join(dir, "testdata", "test.json")
		if _, err := os.Stat(p); err == nil {
			return p
		}
		next := filepath.Dir(dir)
		if next == dir {
			t.Fatal("testdata/test.json not found")
		}
		dir = next
	}
}

func TestIsStackFile(t *testing.T) {
	if !data.IsStackFile("testdata/test.json") || !data.IsStackFile("test.json") {
		t.Fatal("json paths are stack files")
	}
	if data.IsStackFile("gsimone/leva-2") || data.IsStackFile("owner/name") || data.IsStackFile("") {
		t.Fatal("owner/name is not a stack file")
	}
}

func TestLoadStacksTestdata(t *testing.T) {
	repo, stacks, err := data.LoadStacks(testdataJSON(t))
	if err != nil {
		t.Fatal(err)
	}
	if repo != "example/stacks" {
		t.Fatalf("repo %q", repo)
	}
	if len(stacks) < 3 {
		t.Fatalf("want a few authored stacks, got %d", len(stacks))
	}
	if stacks[0].Name != "auth cleanup" || len(stacks[0].PRs) != 3 {
		t.Fatalf("auth stack: %+v", stacks[0])
	}
	if stacks[0].PRs[0].Title != "Split auth scope from session checks" {
		t.Fatalf("named PR: %+v", stacks[0].PRs[0])
	}
	if len(stacks[0].PRs[0].Labels) != 2 || stacks[0].PRs[0].Labels[0].Name != "bug" || stacks[0].PRs[0].Labels[0].Color != "#d73a4a" {
		t.Fatalf("file labels: %+v", stacks[0].PRs[0].Labels)
	}
	if stacks[0].PRs[0].Author != "gianni" || stacks[0].PRs[0].AuthorColor != domain.LoginColor("gianni") {
		t.Fatalf("file author: %+v", stacks[0].PRs[0])
	}
	if domain.GetDisplayState(stacks[0].PRs[2]) != domain.StateCIFailure {
		t.Fatalf("mixed CI on auth head: %s", domain.GetDisplayState(stacks[0].PRs[2]))
	}
	freight := stacks[len(stacks)-1]
	if freight.Name != "schema cutover" || len(freight.PRs) < 8 {
		t.Fatalf("freight-quality train: %+v", freight)
	}
	if freight.PRs[0].Title == "Freight layer 1" {
		t.Fatal("authored names, not random filler")
	}
}

func TestLoadStacksRejectsProviderConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dango.json")
	if err := os.WriteFile(path, []byte(`{"provider":"codex@luna.medium"}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, _, err := data.LoadStacks(path)
	if err == nil || !strings.Contains(err.Error(), "provider config") {
		t.Fatalf("dango.json is not a stack dump: %v", err)
	}
}

func TestLoadStacksMissingFile(t *testing.T) {
	_, _, err := data.LoadStacks(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("missing file must error")
	}
}
