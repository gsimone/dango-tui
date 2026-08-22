package domain_test

import (
	"regexp"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

func TestOKLCHNeutralEndpoints(t *testing.T) {
	if got := domain.OKLCHToSRGB(domain.OKLCH{0, 0, 0}); got != [3]int{0, 0, 0} {
		t.Fatalf("black: %v", got)
	}
	if got := domain.OKLCHToSRGB(domain.OKLCH{1, 0, 0}); got != [3]int{255, 255, 255} {
		t.Fatalf("white: %v", got)
	}
	hex := regexp.MustCompile(`^#[0-9a-f]{6}$`)
	if !hex.MatchString(domain.OKLCHToHex(domain.OKLCHTokens["ready"])) {
		t.Fatalf("ready hex %s", domain.OKLCHToHex(domain.OKLCHTokens["ready"]))
	}
	for name, value := range domain.TerminalColors {
		if !hex.MatchString(value) {
			t.Fatalf("%s is not a terminal hex color: %s", name, value)
		}
	}
}

func TestOKLCHTokenRelationships(t *testing.T) {
	draft := domain.OKLCHTokens["draft"]
	open := domain.OKLCHTokens["open"]
	if abs(draft[0]-open[0]) >= 0.08 {
		t.Fatalf("draft/open lightness too far: %v vs %v", draft, open)
	}
	if abs(draft[1]-open[1]) >= 0.02 {
		t.Fatalf("draft/open chroma too far: %v vs %v", draft, open)
	}
	if abs(draft[2]-open[2]) >= 1 {
		t.Fatalf("draft/open hue too far: %v vs %v", draft, open)
	}
	if abs(domain.OKLCHTokens["ciFailure"][2]-domain.OKLCHTokens["reviewBlocked"][2]) <= 10 {
		t.Fatal("danger hues should stay visibly distinct")
	}
	if domain.OKLCHTokens["queued"][2] <= 70 {
		t.Fatal("queued should stay in the warm yellow range")
	}
	if domain.OKLCHTokens["ready"][1] <= 0.1 {
		t.Fatal("ready should stay chromatic")
	}
	if domain.OKLCHTokens["merged"][2] <= 280 {
		t.Fatal("merged should stay in the magenta range")
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
