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
		"paper":         "#f2ebe0",
		"text":          "#f2ebe0",
		"meta":          "#9a8f82",
		"muted":         "#9a8f82",
		"stick":         "#8f8678",
	}
	for name, hex := range want {
		if got := domain.Color(name); got != hex {
			t.Fatalf("%s: got %s want %s", name, got, hex)
		}
	}
	if domain.Color("paper") == domain.Color("meta") {
		t.Fatal("paper and meta must stay distinct")
	}
	if domain.Color("muted") != domain.Color("meta") {
		t.Fatal("dim copy must be meta, not a third gray")
	}
	if _, ok := domain.ChromeHex["ready"]; ok {
		t.Fatal("ready must not be chrome-locked")
	}
	if domain.Color("ready") == domain.Color("stick") {
		t.Fatal("status color collided with connector")
	}
}

func TestLogoTokensExact(t *testing.T) {
	want := map[string]domain.OKLCH{
		"logoRed":    {0.63, 0.22, 25},
		"logoOrange": {0.72, 0.18, 55},
		"logoYellow": {0.85, 0.16, 95},
		"logoGreen":  {0.72, 0.18, 145},
		"logoBlue":   {0.62, 0.16, 250},
		"logoPurple": {0.60, 0.20, 310},
		"logoPink":   {0.70, 0.18, 350},
	}
	if len(domain.LogoTokens) != 7 {
		t.Fatalf("logo set must be seven tokens, got %d", len(domain.LogoTokens))
	}
	for name, token := range want {
		got, ok := domain.OKLCHTokens[name]
		if !ok {
			t.Fatalf("missing %s", name)
		}
		if got != token {
			t.Fatalf("%s: got %v want %v", name, got, token)
		}
		if _, locked := domain.ChromeHex[name]; locked {
			t.Fatalf("%s must stay OKLCH, not chrome-locked", name)
		}
		if !domain.IsLogoToken(name) {
			t.Fatalf("%s should be in the logo set", name)
		}
	}
}

func TestNormalizeHexAndLoginColor(t *testing.T) {
	if got := domain.NormalizeHex("d73a4a"); got != "#d73a4a" {
		t.Fatalf("github label: %q", got)
	}
	if got := domain.NormalizeHex("#D73A4A"); got != "#d73a4a" {
		t.Fatalf("hashed: %q", got)
	}
	if got := domain.NormalizeHex("abc"); got != "#aabbcc" {
		t.Fatalf("short: %q", got)
	}
	if domain.NormalizeHex("nope") != "" || domain.NormalizeHex("") != "" {
		t.Fatal("invalid hex must be empty")
	}
	a := domain.LoginColor("gianni")
	b := domain.LoginColor("gianni")
	if a != b || a == "" || a[0] != '#' {
		t.Fatalf("login color must be stable, got %q %q", a, b)
	}
	if domain.LoginColor("gianni") == domain.LoginColor("lina") {
		t.Fatal("different logins should not collide on this pair")
	}
	for _, login := range []string{"", "gianni", "gm", "lina", "gsimone"} {
		hex := domain.LoginColor(login)
		if domain.IsLowChromaHex(hex) {
			t.Fatalf("LoginColor(%q)=%s is grey/meta/paper", login, hex)
		}
		if hex == domain.Color("meta") || hex == domain.Color("paper") {
			t.Fatalf("LoginColor(%q) must not be chrome grey", login)
		}
	}
	if domain.IsLowChromaHex(domain.Color("meta")) != true || domain.IsLowChromaHex("#d73a4a") {
		t.Fatal("meta is low chroma; bug red is not")
	}
}

func TestPickLogoDotsAreDistinct(t *testing.T) {
	seq := 0
	intn := func(n int) int {
		seq++
		if n < 1 {
			return 0
		}
		return seq % n
	}
	for i := 0; i < 64; i++ {
		got := domain.PickLogoDots(intn)
		seen := map[string]bool{}
		for _, token := range got {
			if !domain.IsLogoToken(token) {
				t.Fatalf("picked %q which is not in the seven", token)
			}
			if seen[token] {
				t.Fatalf("replacement: %v", got)
			}
			seen[token] = true
		}
		if len(seen) != 3 {
			t.Fatalf("need three distinct, got %v", got)
		}
	}
}
