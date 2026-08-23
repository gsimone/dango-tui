# dango

Shipping order (Cassandra / Gianmarco). Do not skip ahead.

## Now (this PR)

- Advertised path: no `--repo` → cwd git remote (origin, else first remote) → live `gh`. Always. No silent examples.
- Detect failure dies loud (exit 2). Error names `--repo owner/name` and `--repo testdata/test.json`.
- Authored examples exist only as `--repo testdata/test.json` (or another `.json` dump). Never the no-flag path.
- `--repo owner/name` is live `gh`. `--repo` wins over detect. `dango.json` is provider config, not a stack dump.
- `-story` is not a user-facing mode. Hidden test/dev hook only. Do not advertise it.
- Provider comes from `dango.json` or `dango.yml` / `dango.yaml`. `--provider` still overrides.
- Missing config file = no generated title.
- No picker screen. No `[ p ] provider`. Picker is dead.
- Header is packed `●-●-● DANGO` only. No dumpling. No emoji.
- Inspector: status color on the status value only. Title and other facts stay paper/meta. Diff keeps +/− colors.
- Same inspector card also has `labels` (each name in its GitHub hex; empty is `none` in meta) and `author` (`●` + login; ● is avatar-dominant or a stable login color). No new chrome.
- `.` copies the selected layer branch. Toast `copied {branch}`, then it dies. No `,`.

## Later (keep here; do not ship until asked)

- Nothing else is queued. Do not invent a settings screen. Do not ship a picker. Do not bring `-story` back as a user mode.

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
- Inspector `labels` + `author` rows. Live labels/avatarUrl come from `gh pr list --json`. No Go GitHub API client.
