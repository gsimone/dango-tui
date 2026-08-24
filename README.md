# dango

[![CI](https://github.com/gsimone/dango-tui/actions/workflows/ci.yml/badge.svg)](https://github.com/gsimone/dango-tui/actions/workflows/ci.yml)

A small native terminal UI for reading a GitHub pull-request stack at a glance.

The product path is a Go binary (Go 1.24+) built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea). No Bubbles restyle. No picker.

## Run

From this branch:

```bash
go run ./cmd/dango
```

- `go run ./cmd/dango`: always the cwd git remote (origin, else first remote). Live `gh`. No silent examples.
- No GitHub owner/name remote: the process dies. Pass `--repo owner/name` or `--repo testdata/test.json`.
- `--repo owner/name`: live `gh`. `--repo` wins over detect.
- `--repo testdata/test.json`: JSON dump of authored stacks. Never sent to `gh`. Examples exist only this way.

```bash
go run ./cmd/dango --repo owner/name
go run ./cmd/dango --repo testdata/test.json
go run ./cmd/dango --frame 80x24      # print one frame and exit
```

Or build:

```bash
make build
./dango
```

`make build` is `CGO_ENABLED=0 go build -ldflags="-s -w" -o dango ./cmd/dango`.

Provider comes from `dango.json` / `dango.yml` / `dango.yaml`. `--provider` overrides. Missing config file = no generated title. A set provider writes a short stack title (list name, in place) and a description (inspector only) after first paint. First paint is always the gh name. `dango.json` is not a stack dump.

The header mark is `●-●-● DANGO` over the repo slug and counts. Type is paper
`#f2ebe0` or meta `#9a8f82`. Two-col list: name + ball chain. Inspector is the
right pane on the same field.

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

- Header: `●-●-● DANGO`, then `owner/name  •  N stacks / M layers` (examples use `org/reponame`)
- 2-column side pad, 1 blank row at the top
- Stack list on the left; one `│` rule; inspector pane on the right
- Inspector facts include status (status ink on the value only), labels in their GitHub hex, and author (`●` + login)
- Selected row `#242018` with paper ink; everything else meta
- **40×20 / 80×24 / 120×30 / 160×40** — column stays on-screen
