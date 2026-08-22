# dango

A small native terminal UI for reading a GitHub pull-request stack at a glance.

This milestone uses deterministic local fixtures only. It does not call GitHub,
run `git`, open a browser, or check out a branch. Checkout, open, and refresh
are clearly labelled as simulations. Every status line says fixture or
simulated.

The previous Bun / OpenTUI / Solid tree is gone. The product path is a small
Go binary (Go 1.24+) built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lipgloss](https://github.com/charmbracelet/lipgloss).

## Run

```bash
go run ./cmd/dango
```

Or build a static-ish binary:

```bash
make build
./dango
```

`make build` is `CGO_ENABLED=0 go build -ldflags="-s -w" -o dango ./cmd/dango`.

The app is a native terminal UI, not a dev server. It handles 40×20, 80×24,
120×30, and 160×40 terminals: the inspector docks below the list at compact
sizes and becomes a stable right pane on wide terminals.

`go run ./cmd/dango` is the app: local fixture / test data. There is no live GitHub fetch yet. `o` opens the PR URL in your browser. There is no GitHub
client yet — checkout / open / refresh are labelled simulations.

```bash
go run ./cmd/dango --frame 80x24      # print one frame and exit
```

## Controls

| Input | Result |
| --- | --- |
| `↑` / `↓` | Select a stack |
| `←` / `→`, `Home` / `End` | Select a layer from base to head |
| hover / click a ball | Inspect / simulate checkout |
| `Enter`, `o`, `r` | Simulate checkout, open, refresh |
| `/` | Filter local fixtures |
| `Esc`, `?`, `q` | Close, toggle concise help, quit safely |

## Test

```bash
go test ./...
```

or `make test`.

Domain tests cover display-state precedence and the OKLCH palette. App tests
cover selection, filtering, fixture frames at 40/80/120/160, keyboard, and
mouse hits on the two-cell PR balls.

## Layout

- **40×20** — compact list, inspector docks below, short footer
- **80×24** — full stack rows, inspector still below the list
- **120×30 / 160×40** — inspector becomes a stable right pane
