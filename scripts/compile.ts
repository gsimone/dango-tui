import { mkdirSync } from "node:fs"
import { dirname, resolve } from "node:path"
import solidPlugin from "@opentui/solid/bun-plugin"

const platform = process.platform
const arch = process.arch

function defaultAssetName(): string {
  if (platform === "linux" && arch === "x64") return "dango-linux-x64"
  if (platform === "linux" && arch === "arm64") return "dango-linux-arm64"
  if (platform === "darwin" && arch === "x64") return "dango-darwin-x64"
  if (platform === "darwin" && arch === "arm64") return "dango-darwin-arm64"
  if (platform === "win32" && arch === "x64") return "dango-windows-x64.exe"
  if (platform === "win32" && arch === "arm64") return "dango-windows-arm64.exe"
  throw new Error(`No release asset name for ${platform}-${arch}`)
}

function parseOutfile(): string {
  const flagIndex = process.argv.indexOf("--outfile")
  if (flagIndex >= 0) {
    const value = process.argv[flagIndex + 1]
    if (!value) throw new Error("--outfile requires a path")
    return resolve(value)
  }
  return resolve("dist", defaultAssetName())
}

const outfile = parseOutfile()
const linuxLibc = process.env.OPENTUI_LIBC === "musl" ? "musl" : "glibc"

mkdirSync(dirname(outfile), { recursive: true })

const result = await Bun.build({
  entrypoints: [resolve("src/index.tsx")],
  target: "bun",
  plugins: [solidPlugin],
  compile: {
    outfile,
    // JSX is already transformed by the Solid plugin. A nearby bunfig.toml
    // preload would otherwise make the binary look for @opentui/solid/preload.
    autoloadBunfig: false,
    autoloadDotenv: false,
  },
  ...(platform === "linux"
    ? {
        define: {
          "process.env.OPENTUI_LIBC": JSON.stringify(linuxLibc),
        },
      }
    : {}),
})

if (!result.success) {
  for (const log of result.logs) {
    console.error(log)
  }
  process.exit(1)
}

console.log(`compiled ${outfile}`)
