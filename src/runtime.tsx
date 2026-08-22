import { render } from "@opentui/solid"
import { color } from "./domain/colors.ts"
import { App, type AppProps } from "./app/App.tsx"

export async function launchFixtureApp(props: AppProps): Promise<void> {
  await render(() => <App {...props} />, {
    exitOnCtrlC: false,
    clearOnShutdown: true,
    useMouse: true,
    enableMouseMovement: true,
    screenMode: "alternate-screen",
    openConsoleOnError: false,
    backgroundColor: color("surface"),
  })
}
