# dango

Shipping order (Cassandra / Gianmarco). Do not skip ahead.

## Now (this PR)

- Full-screen provider / model list. Not a dropdown.
- Enter selects. Esc cancels, back to the stack list.
- Same chrome (paper/meta, bracketed keys). No extra chrome.
- `p` opens the picker. Footer `[ p ] provider`.
- `--provider` still overrides the picker (skip the screen or preselect).
- Do not block first paint / gh fetch on the picker.
- cwd `git remote` + `dango.json` stay. `--repo` still wins. Missing `dango.json` = no generated title.

## Later (keep here; do not ship until asked)

- Nothing else is queued. Do not invent a settings screen.

## Locked (already shipped)

- Live `gh` via `--repo` (also `-repo`).
- Two-col list: name + ball chain. Fixed gutter. No status column.
- Header line 2 is the `--repo` slug + live counts.
- `-story` still forces fixtures and ignores live fetch.
- `--provider` is optional and never blocks fetch or first paint.
- Missing `gh` fails loudly (`LookPath` / `runGH`). No fixture fallback.
- `dango.json` + cwd `git remote` detection. `--repo` still wins over detect.
- `--provider` still overrides json. Missing `dango.json` = no generated title.
- No status column. Status lives on balls + inspector.
- Packed `●-●-●` logo (U+25CF). Three Pablo inks. No dumpling.
- List balls use meaning colors, not logo hues.
- Footer: bracketed keys. No enter / checkout. `?` help overlay.
