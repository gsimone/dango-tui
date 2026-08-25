package cli

import (
	"os"
	"strings"
	"testing"
)

func TestNightlyAndDistPublishDescribeAsset(t *testing.T) {
	nightly, err := os.ReadFile(repoFile(t, ".github", "workflows", "nightly.yml"))
	if err != nil {
		t.Fatal(err)
	}
	mk, err := os.ReadFile(repoFile(t, "Makefile"))
	if err != nil {
		t.Fatal(err)
	}
	readme, err := os.ReadFile(repoFile(t, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(nightly)
	if !strings.Contains(src, "dist/dango-describe") || !strings.Contains(src, "- dango-describe") {
		t.Fatal("nightly must publish the single dango-describe asset")
	}
	if strings.Contains(src, "dango-describe-linux") || strings.Contains(src, "dango-describe-darwin") {
		t.Fatal("dango-describe is one file, not per-platform")
	}
	if !strings.Contains(string(mk), "dist/dango-describe") || !strings.Contains(string(mk), "scripts/dango-describe") {
		t.Fatal("make dist must copy scripts/dango-describe")
	}
	if !strings.Contains(string(readme), `"describe": "dango-describe"`) {
		t.Fatal("README product example is the PATH command")
	}
}
