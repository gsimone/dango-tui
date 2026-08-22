# dango

Shipping order (Cassandra / Gianmarco). Do not skip ahead.

## Now (this PR)

- Advertised path: no `--repo` → authored example stacks. `--repo owner/name` → live `gh`. Same chrome. No demo theme.
- `--repo` is also a JSON file of authored stacks. One flag. `dango.json` is provider config, not a stack dump.
- `-story` is not a user-facing mode. Hidden test/dev hook only. Do not advertise it.
- Default examples are mixed + pair + freight. Authored, not random, not the 300-row chaos load.
- Provider comes from `dango.json` or `dango.yml` / `dango.yaml`. `--provider` still overrides.
- Missing config file = no generated title.
- No picker screen. No `[ p ] provider`. Picker is dead.
- Header is packed `●-●-● DANGO` only. No dumpling. No emoji.
- Inspector: status color on the status value only. Title and other facts stay paper/meta. Diff keeps +/− colors.
- Same inspector card also has `labels` (each name in its GitHub hex; empty is `none` in meta) and `author` (`●` + login; ● is avatar-dominant or a stable login color). No new chrome.

## Later (keep here; do not ship until asked)

- Nothing else is queued. Do not invent a settings screen. Do not ship a picker. Do not bring `-story` back as a user mode.

## Locked (already shipped)

- Live `gh` via `--repo owner/name` (also `-repo`).
- Two-col list: name + ball chain. Fixed gutter. No status column.
- Header line 2 is the repo slug + counts. Examples use `org/reponame` and the same fetch chrome as live.
- `--provider` is optional and never blocks fetch or first paint.
- Missing `gh` fails loudly (`LookPath` / `runGH`). No fixture fallback.
- No cwd git-remote auto-fetch. Empty `--repo` is examples.
- No status column. Status lives on balls + inspector.
- Packed `●-●-●` logo (U+25CF). Three Pablo inks. No dumpling.
- List balls use meaning colors, not logo hues.
- Footer: bracketed keys. No enter / checkout. `?` help overlay.
- Inspector `labels` + `author` rows. Live labels/avatarUrl come from `gh pr list --json`. No Go GitHub API client.
