# dango

Shipping order (Cassandra / Gianmarco). Do not skip ahead.

## Now (this PR)

Agent titles/descriptions.

- Fetch paints first with the real gh names. Never block first paint on a summarizer.
- When a description lands it fills the inspector pane in place — same rows, no card morph, no spinner on the list.
- Title swaps the list name in place when the generated clause is not the gh title.
- `--provider local` / `demo` write a two-line meta clause only if it is not the gh title pasted back. No invented `Covers …` prefix. If there is no distinct sentence, leave the reserved slot empty (dim). Never paste `pr.Body`, HTML comments, or agent markers. No Codex network call.
- Provider from `dango.json` / `dango.yml` / `dango.yaml`, or `--provider` which wins. No new flags.
- Missing provider = no generated title. Keep the gh name.
- Pablo chrome (this PR only): pane fills in place, same rows, no card morph, no list spinner.
- Inspector description is two lines under the title, meta ink, then stop. Same reserved slot every time. Card height does not jump. Wrap is clipped to `inspectorDescLines` (2) — never a third line.

## Later

Build / publish / install the Go binary. The old #2 release PR stays parked. Do not start a release cut until a description has actually landed in the TUI.

## Dead

- Picker. No `[ p ] provider`. No settings screen.
- `,` copy. `[ . ] copy` only.
- Stacking on #5. #5 is already on main (no-flag = cwd origin, `--repo owner/name` live, `--repo *.json` dump, `.` copy, labels/author). This PR is off main, not #5.
