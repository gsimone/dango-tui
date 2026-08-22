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

func TestChromeHexLocks(t *testing.T) {
	want := map[string]string{
		"surface":       "#14120f",
		"surfaceRaised": "#242018",
		"text":          "#efeae2",
		"stick":         "#8f8678",
	}
	for name, hex := range want {
		if got := domain.Color(name); got != hex {
			t.Fatalf("%s: got %s want %s", name, got, hex)
		}
	}
	if _, ok := domain.ChromeHex["ready"]; ok {
		t.Fatal("ready must not be chrome-locked")
	}
	if domain.Color("ready") == domain.Color("stick") {
		t.Fatal("status color collided with connector")
	}
}

func TestPostcardIsNotListGround(t *testing.T) {
	if _, ok := domain.ChromeHex["postcard"]; ok {
		t.Fatal("postcard must not be chrome-locked to the list field")
	}
	if domain.Color("postcard") == domain.Color("surface") {
		t.Fatal("postcard fill must not match the list field")
	}
	if domain.Color("postcard") == domain.Color("surfaceRaised") {
		t.Fatal("postcard fill must not match the selected-row lift")
	}
	if domain.Color("postcardInk") == domain.Color("text") {
		t.Fatal("postcard ink should sit on paper, not cream-on-field type")
	}
	paper := domain.OKLCHTokens["postcard"]
	field := domain.OKLCHTokens["surface"]
	if paper[0] <= field[0]+0.4 {
		t.Fatalf("postcard should be a lifted paper surface, lightness %v vs field %v", paper[0], field[0])
	}
}
