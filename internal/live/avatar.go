package live

import (
	"bytes"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gsimone/dango-tui/internal/domain"
)

// getURL fetches avatar bytes. Tests swap this so live fetch never hits the net.
var getURL = httpGetBytes

var avatarHTTP = &http.Client{Timeout: 3 * time.Second}

func httpGetBytes(rawURL string) ([]byte, error) {
	resp, err := avatarHTTP.Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("avatar http %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 1<<20))
}

func applyAuthorColors(prs []RemotePR) {
	urlByLogin := map[string]string{}
	for _, pr := range prs {
		login := strings.TrimSpace(pr.Author)
		if login == "" || urlByLogin[login] != "" {
			continue
		}
		if u := strings.TrimSpace(pr.AvatarURL); u != "" {
			urlByLogin[login] = u
		}
	}
	cache := map[string]string{}
	for i := range prs {
		login := strings.TrimSpace(prs[i].Author)
		if hex, ok := cache[login]; ok {
			prs[i].AuthorColor = hex
			continue
		}
		hex := resolveAuthorColor(login, avatarURLFor(login, urlByLogin[login]))
		cache[login] = hex
		prs[i].AuthorColor = hex
	}
}

// avatarURLFor prefers the URL from gh (author.avatarUrl when present).
// Slim `gh pr list --json` has no avatarUrl field and author is
// {login,id,name} only, so we fetch https://github.com/{login}.png.
func avatarURLFor(login, fromGH string) string {
	if u := strings.TrimSpace(fromGH); u != "" {
		return u
	}
	login = strings.TrimSpace(login)
	if login == "" {
		return ""
	}
	return "https://github.com/" + login + ".png"
}

func resolveAuthorColor(login, avatarURL string) string {
	fallback := domain.LoginColor(login)
	avatarURL = strings.TrimSpace(avatarURL)
	if avatarURL == "" {
		return fallback
	}
	raw, err := getURL(avatarURL)
	if err != nil || len(raw) == 0 {
		return fallback
	}
	hex, err := dominantHex(raw)
	if err != nil || hex == "" {
		return fallback
	}
	return hex
}

func dominantHex(raw []byte) (string, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 1 || h < 1 {
		return "", fmt.Errorf("empty image")
	}
	stepX, stepY := max(1, w/16), max(1, h/16)
	type bucket struct{ r, g, b, n, chroma int }
	hist := map[uint32]*bucket{}
	var best *bucket
	for y := bounds.Min.Y; y < bounds.Max.Y; y += stepY {
		for x := bounds.Min.X; x < bounds.Max.X; x += stepX {
			cr, cg, cb, ca := img.At(x, y).RGBA()
			if ca < 0x8000 {
				continue
			}
			r, g, b := int(cr>>8), int(cg>>8), int(cb>>8)
			if r+g+b > 720 || r+g+b < 24 || domain.IsLowChroma(r, g, b) {
				continue
			}
			key := uint32(r>>4)<<8 | uint32(g>>4)<<4 | uint32(b>>4)
			slot := hist[key]
			if slot == nil {
				slot = &bucket{}
				hist[key] = slot
			}
			slot.r += r
			slot.g += g
			slot.b += b
			slot.n++
			avgR, avgG, avgB := slot.r/slot.n, slot.g/slot.n, slot.b/slot.n
			slot.chroma = domain.RGBChroma(avgR, avgG, avgB)
			if best == nil || slot.chroma > best.chroma || (slot.chroma == best.chroma && slot.n > best.n) {
				best = slot
			}
		}
	}
	if best == nil || best.n == 0 || best.chroma < domain.LowChroma {
		return "", fmt.Errorf("no chromatic pixels")
	}
	hex := domain.NormalizeHex(fmt.Sprintf("%02x%02x%02x", best.r/best.n, best.g/best.n, best.b/best.n))
	if domain.IsLowChromaHex(hex) {
		return "", fmt.Errorf("sampled grey")
	}
	return hex, nil
}
