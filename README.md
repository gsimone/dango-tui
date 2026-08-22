# stacks

A small native OpenTUI for reading a GitHub pull-request stack at a glance.
This milestone deliberately uses deterministic local fixtures: it does not call
GitHub, run `git`, open a browser, or check out a branch. Checkout/open/refresh
feedback is clearly labelled as a simulation.

## Run

```bash
bun run stacks
```

In this workspace Bun is available at
`/tmp/stacks-bun-tooling/node_modules/.bin/bun`.

The app is native terminal UI, not a dev server. It handles 40×20, 80×24,
120×30, and 160×40 terminals: the inspector docks below the list at compact
sizes and becomes a stable right pane on wide terminals.

## Controls

| Input | Result |
| --- | --- |
| `↑` / `↓` | Select a stack |
| `←` / `→`, `Home` / `End` | Select a layer from base to head |
| hover / click a ball | Inspect / simulate checkout |
| `Enter`, `o`, `r` | Simulate checkout, open, refresh |
| `/` | Filter local fixtures |
| `Esc`, `?`, `q` | Close, toggle concise help, quit safely |
| `[` / `]` | Cycle fixture stories |

The fixture cycle includes current, stale-cache, refresh-error, and empty
repository states. No network state is implied: every status says fixture or
simulated.

## Verify

```bash
bun run check
```

`src/stories/native-frames.html` is a static, dev-only sheet of copied native
`testRender` frames for 40/80/120/160-column visual review. It contains no
terminal, process, WebSocket, or server implementation.

The frozen legacy browser bridge remains outside the native product path. Its
tests are opt-in only: `bun run test:web` or `bun run check:web`.
