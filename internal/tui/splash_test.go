package tui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/gsimone/dango-tui/internal/domain"
)

func TestDangoBlockIsThreeShadeRows(t *testing.T) {
	if len(dangoBlock) != 3 {
		t.Fatalf("3-row letters, got %d", len(dangoBlock))
	}
	width := displayWidth(dangoBlock[0])
	for i, row := range dangoBlock {
		if displayWidth(row) != width {
			t.Fatalf("row %d width %d, want %d: %q", i, displayWidth(row), width, row)
		}
	}
	joined := strings.Join(dangoBlock[:], "\n")
	for _, ch := range []string{"░", "▒", "▓", "█"} {
		if !strings.Contains(joined, ch) {
			t.Fatalf("missing %s in\n%s", ch, joined)
		}
	}
	if strings.Contains(joined, "_") || strings.Contains(joined, "|") {
		t.Fatalf("not a figlet:\n%s", joined)
	}
}

func TestSplashPaintsBeforeFetchAndDies(t *testing.T) {
	fetches := 0
	m := New(Options{
		Repo:   "owner/name",
		Width:  80,
		Height: 24,
		Fetch: func(string) ([]domain.Stack, error) {
			fetches++
			return []domain.Stack{{
				ID:  "s",
				PRs: []domain.PullRequest{{Number: 1, Title: "landed layer"}},
			}}, nil
		},
	})
	if fetches != 0 {
		t.Fatalf("constructor fetched %d times", fetches)
	}
	if !m.splash() {
		t.Fatal("live open is a splash until the list exists")
	}
	raw := m.View()
	frame := stripANSI(raw)
	if fetches != 0 {
		t.Fatalf("View fetched %d times", fetches)
	}
	for _, row := range dangoBlock {
		if !strings.Contains(frame, row) {
			t.Fatalf("missing %q:\n%s", row, frame)
		}
	}
	if !strings.Contains(frame, "●-●-●") {
		t.Fatalf("dots:\n%s", frame)
	}
	if !strings.Contains(frame, "fetching owner/name") {
		t.Fatalf("meta line:\n%s", frame)
	}
	if strings.Contains(frame, "YOINKS") {
		t.Fatalf("no joke words:\n%s", frame)
	}
	mr, mg, mb := rgb(domain.Color("meta"))
	pr, pg, pb := rgb(domain.Color("paper"))
	if !strings.Contains(raw, fmt.Sprintf("38;2;%d;%d;%d", mr, mg, mb)) {
		t.Fatal("fetching line is meta")
	}
	if !strings.Contains(raw, fmt.Sprintf("38;2;%d;%d;%d", pr, pg, pb)) {
		t.Fatal("block letters are paper")
	}

	next, _ := applyFetch(m)
	if fetches != 1 {
		t.Fatalf("Init fetches once, got %d", fetches)
	}
	if next.splash() {
		t.Fatal("splash dies once fetch returns")
	}
	listed := stripANSI(next.View())
	if strings.Contains(listed, "fetching owner/name") || strings.Contains(listed, dangoBlock[0]) {
		t.Fatalf("splash leftover:\n%s", listed)
	}
	if !strings.Contains(listed, "landed layer") {
		t.Fatalf("list:\n%s", listed)
	}
}

func rgb(hex string) (r, g, b int) {
	r, g, b, _ = domain.ParseRGB(hex)
	return r, g, b
}
