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

The app is a native terminal UI, not a dev server. The header mark is
`●-●-● DANGO` over `org/reponame  •  N stacks / M layers`. Type is paper
`#f2ebe0` or meta `#9a8f82`. The inspector is a full-height right pane on the
same field, divided by one dim vertical rule.

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
| hover / click a ball | Inspect / select that layer |
| `o`, `.`, `r` | Open the PR URL, copy the branch, simulate refresh |
| `/` | Filter local fixtures |
| `?` | Open or close the help overlay |
| `Esc`, `q` | Close, quit safely |

## Test

```bash
go test ./...
```

or `make test`.

Domain tests cover display-state precedence and the OKLCH palette. App tests
cover selection, filtering, fixture frames at 40/80/120/160, keyboard, and
mouse hits on the two-cell PR balls.

## Layout

- Header: `●-●-● DANGO`, then `org/reponame  •  N stacks / M layers`
- 2-column side pad, 1 blank row at the top
- Stack list on the left; one `│` rule; inspector pane on the right
- Selected row `#242018` with paper ink; everything else meta
- **40×20 / 80×24 / 120×30 / 160×40** — column stays on-screen
