# dango

Shipping order (Cassandra / Gianmarco). Do not skip ahead.

## Now (this PR)

- `--repo` is `owner/name` (live `gh`) OR a JSON file of authored stacks. One flag.
- `-story` stays. Fixtures when set. Live `--repo` is unchanged.
- Default demos are the good small ones: `mixed`, `freight`, `pair`. Authored, not random.
- Chaos/300 stays as `-story chaos` stress only. Not the default demo.
- Provider comes from `dango.json` or `dango.yml` / `dango.yaml`. `--provider` still overrides.
- Missing config file = no generated title. `dango.json` is not a stack dump.
- No picker screen. No `[ p ] provider`. Picker is dead.
- Header is packed `●-●-● DANGO` only. No dumpling. No emoji.
- Inspector: status color on the status value only. Title and other facts stay paper/meta. Diff keeps +/− colors.

## Later (keep here; do not ship until asked)

- Nothing else is queued. Do not invent a settings screen. Do not ship a picker.

## Locked (already shipped)

- Live `gh` via `--repo owner/name` (also `-repo`).
- Two-col list: name + ball chain. Fixed gutter. No status column.
- Header line 2 is the repo slug + live/authored counts.
- `--provider` is optional and never blocks fetch or first paint.
- Missing `gh` fails loudly (`LookPath` / `runGH`). No fixture fallback.
- cwd `git remote` detection. `--repo` still wins over detect. `-story` ignores both.
- No status column. Status lives on balls + inspector.
- Packed `●-●-●` logo (U+25CF). Three Pablo inks. No dumpling.
- List balls use meaning colors, not logo hues.
- Footer: bracketed keys. No enter / checkout. `?` help overlay.
