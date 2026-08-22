# dango

Shipping order (Cassandra / Gianmarco). Do not skip ahead.

## Now (this PR)

- Live `gh` via `--repo owner/name` (also `-repo`).
- Two-col list: name + ball chain. Fixed gutter. No status column.
- Header line 2 is the `--repo` slug + live counts.
- `-story` still forces fixtures and ignores live fetch.
- `--provider` is optional and never blocks fetch or first paint.
- No picker. No settings screen.

## Next PR (after fetch)

- `dango.json` + cwd `git remote` detection.
- `--repo` still wins over detect.
- `--provider` still overrides json.
- Missing `dango.json` = no generated title.
- No settings screen in that PR.

## Later (keep here; do not ship until asked)

- Full-screen provider / model list.
- Enter selects. Esc cancels. Same chrome.
- `--provider` still overrides the picker.
- Do not implement this in the next code drop.

## Locked (already shipped)

- No status column. Status lives on balls + inspector.
- Packed `●-●-●` logo (U+25CF). Three Pablo inks. No dumpling.
- List balls use meaning colors, not logo hues.
- Footer: bracketed keys. No enter / checkout. `?` help overlay.
