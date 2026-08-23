# dango

Shipping order (Cassandra / Gianmarco). Do not skip ahead.

## Now (this PR)

Agent titles/descriptions.

- Fetch paints first with the real gh names. Never block first paint on a summarizer.
- When a description lands it fills the inspector pane in place — same rows, no card morph, no spinner on the list.
- Title swaps the list name in place when the generated clause is not the gh title.
- `--provider local` / `demo` invent one `Covers {clause}.` sentence. Never paste `pr.Body`, HTML comments, or `CURSOR_AGENT` markers. Never echo GhTitle. No Codex network call.
- Provider from `dango.json` / `dango.yml` / `dango.yaml`, or `--provider` which wins. No new flags.
- Missing provider = no generated title. Keep the gh name.
- Pablo chrome (this PR only): pane fills in place, same rows, no card morph, no list spinner.
- Descriptions are two lines max. `inspectorDescLines` is 2. `Covers …` may wrap but wrap is clipped to 2 — never a third line, never grow the card.

## Later

Build / publish / install the Go binary. The old #2 release PR stays parked. Do not start a release cut until a description has actually landed in the TUI.

## Dead

- Picker. No `[ p ] provider`. No settings screen.
- `,` copy. `[ . ] copy` only.
- Stacking on #5. #5 is already on main (no-flag = cwd origin, `--repo owner/name` live, `--repo *.json` dump, `.` copy, labels/author). This PR is off main, not #5.
