# dango

Shipping order (Cassandra / Gianmarco). Do not skip ahead.

## Now (this PR)

Size / speed. #6 shipped (live titles/descriptions, two-line inspector slot, no `Covers` wrapper).

- Smaller stripped linux binary (`CGO_ENABLED=0 -ldflags='-s -w'`).
- Faster first paint / live `gh` fetch / TUI render.
- Product path unchanged: no-flag = cwd remote, `--repo owner/name` live `gh`, `--repo *.json` file, `.` copy, two-line inspector, no Covers, no picker.
- Tests 100% deterministic. No `math/rand`, no shuffled fixtures, no chaos story.

## Later

Build / publish / install the Go binary. Do not start a release cut from this PR.

## Locked (already shipped)

- Live `gh` via `--repo owner/name` (also `-repo`).
- Two-col list: name + ball chain. Fixed gutter. No status column.
- Header line 2 is the repo slug + counts.
- `--provider` is optional and never blocks fetch or first paint.
- Missing `gh` fails loudly (`LookPath` / `runGH`). No fixture fallback.
- cwd git-remote detect when `--repo` is omitted. `--repo` wins (owner/name or JSON). Detect failure is a process error, not examples.
- No status column. Status lives on balls + inspector.
- Packed `●-●-●` logo (U+25CF). Three Pablo inks. No dumpling.
- List balls use meaning colors, not logo hues.
- Footer: bracketed keys. No enter / checkout. `?` help overlay. `[ . ] copy` only.
- Inspector `labels` + `author` rows. Live labels come from `gh pr list --json`.
- Agent titles/descriptions land in place. Inspector description is two wrap lines. No invented `Covers` prefix. No picker. `-story` stays a hidden hook.
