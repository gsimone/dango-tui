# dango

[![CI](https://github.com/gsimone/dango-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/gsimone/dango-tui/actions/workflows/ci.yml)

A small native terminal UI for reading a GitHub pull-request stack at a glance.

The product path is a Go binary (Go 1.24+) built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea). No Bubbles restyle. No picker.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/gsimone/dango-tui/main/install.sh | bash
```

That installs `dango` into `~/.local/bin` from the rolling `nightly` prerelease (linux/amd64 or darwin/arm64). Not a versioned 0.1.0. Other platforms are refused.

## Run

Dev, from this branch:

```bash
go run ./cmd/dango
```

- `go run ./cmd/dango`: always the cwd git remote (origin, else first remote). Live `gh`. No silent examples.
- No GitHub remote: the process dies. Pass `--repo archetype-labs/app` or `--repo testdata/test.json`.
- `--repo archetype-labs/app`: live `gh`. `--repo` wins over detect.
- `--repo testdata/test.json`: JSON dump of authored stacks. Never sent to `gh`. Examples exist only this way.

```bash
go run ./cmd/dango --repo archetype-labs/app
go run ./cmd/dango --repo testdata/test.json
go run ./cmd/dango --frame 120x30     # print the product pane and exit
```

Or build:

```bash
make build
./dango
```

`make build` is `CGO_ENABLED=0 go build -ldflags="-s -w" -o dango ./cmd/dango`.

Nightly (not 0.1.0, no semver): after CI on `main`, plus 02:00 UTC and `workflow_dispatch`, the `Nightly` workflow replaces a single prerelease tag `nightly` with stripped `dango-linux-amd64` and `dango-darwin-arm64`. No darwin/amd64. PRs do not publish. `make dist` builds those locally.

Provider comes from `dango.json` / `dango.yml` / `dango.yaml`. `--provider` overrides. Missing config file = no generated list title; the list keeps a short stack title (ticket id, or the gh title when it is already short). After first paint, local `Describe()` fills the inspector (two clipped meta lines) with no provider and no `dango.json`. A set provider may still swap a short stack title in place. `dango.json` is not a stack dump.

A stack is two or more open PRs. A single open PR is not a stack and does not appear as a one-ball row. List names stay short (paper, leftover width, one line).

The header mark is `●-●-● DANGO` over the repo slug and counts. Type is paper
`#f2ebe0` or meta `#9a8f82`. Two-col list: name + ball chain. Inspector is the
right pane on the same field.

A live repo paints a 3-row ░▒▓█ DANGO, then `●-●-●`, then
`fetching archetype-labs/app` in meta before `gh` returns. The splash dies
once the list exists. A failed fetch stays on the splash and turns the
loading line into the error (`[ . ]` copies the whole block, including the
exact `gh` argv). `--frame` on a live `--repo` is that first frame.

## Controls

| Input | Result |
| --- | --- |
| `↑` / `↓` | Select a stack |
| `←` / `→`, `Home` / `End` | Select a layer from base to head |
| hover / click a ball | Inspect / select that layer |
| `o` | Open the PR URL |
| `.` | Copy the selected layer branch. Footer toast `copied {branch}`, then it dies |
| `r` | Refresh (live `gh` or reload the JSON dump) |
| `/` | Filter |
| `?` | Open or close the help overlay |
| `Esc`, `q` | Close, quit safely |

Footer shows `[ . ] copy`. There is no `,` binding.

## Test

```bash
go test ./...
```

or `make test`.

Coverage (writes `coverage.out`, prints per-function stats and a total %):

```bash
make cover
```

`make mutate` runs the hand-written `Mutant_*` tests (`go test -run Mutant`). Those are ordinary tests, not a mutator.

The real mutator is [gremlins](https://github.com/go-gremlins/gremlins) v0.6.0, the same tool CI uses. Install the release binary for your OS from the [v0.6.0 release](https://github.com/go-gremlins/gremlins/releases/tag/v0.6.0), then:

```bash
make mutation
```

That mutates `./internal` (app, cli, data, domain, live, summary, tui), writes `gremlins.json`, and does not fail on a score. `testdata/` is not a Go package. `cmd/dango` has no tests, so it is skipped.

CI on pull requests and pushes to `main` publishes coverage (total %) and real gremlins mutation numbers (killed / survived / timed out / score). Those steps report only — no minimum coverage % or mutation score yet. `go test` still fails the job if tests fail.

## Layout

- Live first frame: 3-row ░▒▓█ DANGO, then `●-●-●`, then `fetching archetype-labs/app` (meta). Dies when the list exists. A failed fetch stays here.
- Header: `●-●-● DANGO`, then `archetype-labs/app  •  N stacks / M layers` (fixtures use `org/reponame`)
- 2-column side pad, 1 blank row at the top
- Stack list on the left; one `│` rule; inspector pane on the right (left/right pad)
- List names are short stack titles in paper, leftover measure, one line. Full GitHub title lives in the right pane.
- Balls: one mark, one meaning (no logo rainbow). Fail wins, then review, then the rest. Open paper `●`, draft meta `○`, CI broken red `●`, needs review amber `◎`, approved green `●`, merged dusk `●`, queued `◌`. The layer you are on is `◉` (same status ink). Selected stack is `▶` plus paper name — the row is not washed.
- Inspector facts include status (status ink on the value only), labels in their GitHub hex, and author (`●` + login). `●` is the dominant color from that user's GitHub avatar, never a picture. Fetch failure uses a stable login ink.
- Splash shows a short SHA (module pseudo-version / `vcs.revision` / `-ldflags`). `go install @commit` must still print it.
- Selected stack is `▶` plus paper name; the row is not washed. List titles stay paper; chrome is meta
- **120×30** is the product: list + right inspector pane. Narrow widths keep a stacked card clipped to content + pad.
