package live

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

func TestMain(m *testing.M) {
	getURL = func(string) ([]byte, error) {
		return nil, errors.New("avatar http disabled in tests")
	}
	os.Exit(m.Run())
}

func TestDominantHexPrefersChromaticBucket(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	grey := color.RGBA{R: 140, G: 140, B: 142, A: 255}
	red := color.RGBA{R: 200, G: 40, B: 40, A: 255}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, grey)
		}
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			img.Set(x, y, red)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	hex, err := dominantHex(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, ok := domain.ParseRGB(hex)
	if !ok || r < 160 || g > 80 || b > 80 {
		t.Fatalf("wanted the red patch, got %s", hex)
	}

	greyOnly := solidPNG(t, 140, 140, 142)
	if _, err := dominantHex(greyOnly); err == nil {
		t.Fatal("grey-only avatar must not invent a color")
	}
	old := getURL
	getURL = func(string) ([]byte, error) { return greyOnly, nil }
	t.Cleanup(func() { getURL = old })
	if got := resolveAuthorColor("gm", "https://avatars.example/grey.png"); got != domain.LoginColor("gm") {
		t.Fatalf("grey avatar falls back to login, got %s", got)
	}
}

func TestAvatarFetchFailureFallsBackToLogin(t *testing.T) {
	old := getURL
	getURL = func(string) ([]byte, error) { return nil, errors.New("nope") }
	t.Cleanup(func() { getURL = old })
	if got := resolveAuthorColor("gm", "https://avatars.example/x.png"); got != domain.LoginColor("gm") {
		t.Fatalf("fallback %q", got)
	}
	if got := resolveAuthorColor("gm", ""); got != domain.LoginColor("gm") {
		t.Fatalf("empty url %q", got)
	}
}

func TestAuthorColorCachePerLogin(t *testing.T) {
	var calls []string
	old := getURL
	getURL = func(raw string) ([]byte, error) {
		calls = append(calls, raw)
		switch raw {
		case "https://avatars.example/gm.png":
			return solidPNG(t, 200, 40, 40), nil
		case "https://avatars.example/lina.png":
			return solidPNG(t, 40, 40, 200), nil
		default:
			return nil, errors.New("unexpected " + raw)
		}
	}
	t.Cleanup(func() { getURL = old })

	prs := []RemotePR{
		{Author: "gm", AvatarURL: "https://avatars.example/gm.png"},
		{Author: "gm", AvatarURL: "https://avatars.example/other.png"},
		{Author: "lina", AvatarURL: "https://avatars.example/lina.png"},
	}
	applyAuthorColors(prs)
	if len(calls) != 2 {
		t.Fatalf("one fetch per login, got %v", calls)
	}
	if prs[0].AuthorColor != "#c82828" || prs[1].AuthorColor != "#c82828" {
		t.Fatalf("same login shares sampled ink: %q %q", prs[0].AuthorColor, prs[1].AuthorColor)
	}
	if prs[2].AuthorColor != "#2828c8" {
		t.Fatalf("other login %q", prs[2].AuthorColor)
	}
}

func TestAvatarURLForUsesGhThenGithubPNG(t *testing.T) {
	if got := avatarURLFor("gm", " https://avatars.example/gm.png "); got != "https://avatars.example/gm.png" {
		t.Fatalf("gh url %q", got)
	}
	if got := avatarURLFor("gm", ""); got != "https://github.com/gm.png" {
		t.Fatalf("constructed %q", got)
	}
	if got := avatarURLFor("", ""); got != "" {
		t.Fatalf("empty login %q", got)
	}
}

func TestApplyAuthorColorsConstructsGithubPNG(t *testing.T) {
	var gotURL string
	old := getURL
	getURL = func(raw string) ([]byte, error) {
		gotURL = raw
		return solidPNG(t, 200, 40, 40), nil
	}
	t.Cleanup(func() { getURL = old })

	prs := []RemotePR{{Author: "gm"}}
	applyAuthorColors(prs)
	if gotURL != "https://github.com/gm.png" {
		t.Fatalf("slim author has no avatarUrl, fetch %q", gotURL)
	}
	if prs[0].AuthorColor != "#c82828" {
		t.Fatalf("sampled %q", prs[0].AuthorColor)
	}
}

func solidPNG(t *testing.T, r, g, b uint8) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
