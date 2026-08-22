import type { KeyEvent, MouseEvent } from "@opentui/core"
import { Show, createEffect, createMemo, createSignal } from "solid-js"
import { color } from "../domain/colors.ts"
import type { PullRequest } from "../domain/types.ts"
import { fixtureStories, storyById, type FixtureStory } from "../data/fixtures.ts"
import { clampSelection, filterStacks, initialAppState, moveSelection, type Selection } from "./model.ts"
import { PrCard } from "../components/PrCard.tsx"
import { StackList } from "../components/StackList.tsx"
import { getInspectorSize } from "../components/layout.ts"
import { useKeyboard, useRenderer, useTerminalDimensions } from "@opentui/solid"

export type AppProps = {
  mode: "stacks" | "stories"
  initialStoryId?: string
}

export function App(props: AppProps) {
  const renderer = useRenderer()
  const terminal = useTerminalDimensions()
  const boot = initialAppState()
  const [storyIndex, setStoryIndex] = createSignal(Math.max(0, fixtureStories.findIndex((story) => story.id === (props.initialStoryId ?? "mixed"))))
  const story = createMemo<FixtureStory>(() => fixtureStories[storyIndex()] ?? storyById("mixed"))
  const [selection, setSelection] = createSignal<Selection>(boot.selection)
  const [query, setQuery] = createSignal(boot.query)
  const [searching, setSearching] = createSignal(boot.searching)
  const [cardVisible, setCardVisible] = createSignal(boot.cardVisible)
  const [feedback, setFeedback] = createSignal(boot.feedback)
  const [helpVisible, setHelpVisible] = createSignal(false)
  const stacks = createMemo(() => filterStacks(story().stacks, query()))

  const selectedStack = createMemo(() => stacks()[selection().stackIndex])
  const selectedPr = createMemo(() => selectedStack()?.prs[selection().prIndex])

  createEffect(() => {
    const current = selection()
    const next = clampSelection(current, stacks())
    if (next.stackIndex !== current.stackIndex || next.prIndex !== current.prIndex) setSelection(next)
  })

  const choose = (next: Selection, reveal = true) => {
    const resolved = clampSelection(next, stacks())
    setSelection(resolved)
    if (reveal) setCardVisible(true)
  }

  const checkout = (pr: PullRequest) => {
    setFeedback(`Checked out ${pr.branch} · fixture simulation`)
    setCardVisible(true)
  }

  const open = (pr: PullRequest) => {
    setFeedback(`Opened #${pr.number} · fixture simulation`)
    setCardVisible(true)
  }

  const switchStory = (offset: number) => {
    const nextIndex = (storyIndex() + offset + fixtureStories.length) % fixtureStories.length
    setStoryIndex(nextIndex)
    const next = { stackIndex: 0, prIndex: 0 }
    setSelection(next)
    setQuery("")
    setSearching(false)
    setCardVisible(true)
    setFeedback(`Story: ${fixtureStories[nextIndex]?.label ?? "fixture"}`)
  }

  const hover = (stackIndex: number, prIndex: number, _event: MouseEvent) => {
    choose({ stackIndex, prIndex })
  }

  const handleKey = (event: KeyEvent) => {
    if (searching()) {
      if (event.name === "escape") {
        event.preventDefault()
        if (query()) setQuery("")
        else setSearching(false)
        setCardVisible(true)
      }
      return
    }

    const selected = selection()
    if (event.name === "up" || event.name === "down" || event.name === "left" || event.name === "right" || event.name === "home" || event.name === "end") {
      event.preventDefault()
      choose(moveSelection(selected, stacks(), event.name))
      return
    }
    if (event.name === "return" || event.name === "enter") {
      event.preventDefault()
      const pr = selectedPr()
      if (pr) checkout(pr)
      return
    }
    if (event.name === "o") {
      const pr = selectedPr()
      if (pr) open(pr)
      return
    }
    if (event.name === "r") {
      setFeedback("Fixture data refreshed · no network")
      return
    }
    if (event.name === "/") {
      event.preventDefault()
      setQuery("")
      setSearching(true)
      setCardVisible(false)
      return
    }
    if (event.name === "?") {
      setHelpVisible((visible) => !visible)
      return
    }
    if (event.name === "escape") {
      setCardVisible(false)
      return
    }
    if (event.name === "[") {
      switchStory(-1)
      return
    }
    if (event.name === "]") {
      switchStory(1)
      return
    }
    if (event.name === "q") renderer.destroy()
  }

  useKeyboard(handleKey)

  const activePr = () => selectedPr()
  const inspector = () => getInspectorSize(terminal())
  const isWide = () => terminal().width >= 100
  const isCompact = () => terminal().width <= 50
  const title = () => props.mode === "stories" ? "STACKS UI LAB" : "STACKS"
  const stackCount = () => story().stacks.length
  const layerCount = () => story().stacks.reduce((count, stack) => count + stack.prs.length, 0)
  const sourceState = () => {
    if (story().cacheState === "stale") return "fixture cache · stale (simulated)"
    if (story().cacheState === "error") return "fixture refresh failed · no cached stacks"
    return "fixture cache · current · no network"
  }
  const emptyMessage = () => {
    if (query()) return "No fixture stack matches this filter."
    if (story().cacheState === "error") return "Refresh failed in this fixture. No cached stacks are available."
    return "No open stacks in this fixture repository."
  }
  const footer = () => {
    if (helpVisible()) return isCompact() ? "enter go · o open · r sync · esc · q quit" : "enter checkout · o open · r refresh · esc close · q quit"
    return isCompact() ? "↑↓ stack · ←→ layer · / find · ? help" : "↑↓ stack  ←→ layer  enter checkout  o open  r refresh  / filter  esc close  ? help  [ ] stories  q quit"
  }
  const shortFooter = () => terminal().width <= 90 && !isCompact()
    ? "↑↓ stack · ←→ layer · enter checkout · / filter · ? help · q quit"
    : footer()
  const storyMeta = () => isCompact()
    ? `${story().label} · ${stackCount()} stacks / ${layerCount()} layers`
    : `story ${storyIndex() + 1}/${fixtureStories.length} · ${story().label} · ${stackCount()} stacks / ${layerCount()} layers`

  return (
    <box width="100%" height="100%" flexDirection="column" backgroundColor={color("surface")} paddingX={1}>
      <box height={2} width="100%" flexDirection="column">
        <box height={1} width="100%" flexDirection="row">
          <text flexGrow={1} truncate fg={color("focus")}>{isCompact() ? title() : `${title()} / example/stacks`}</text>
          <text fg={color("muted")}>{props.mode === "stories" ? "fixture story" : "fixture"}</text>
        </box>
        <text height={1} width="100%" truncate fg={color("muted")}>{props.mode === "stories" ? storyMeta() : `${stackCount()} stacks / ${layerCount()} layers · local deterministic data`}</text>
      </box>
      <text height={1} fg={color("border")}>{isCompact() ? "STACK · BASE → HEAD" : "STACK                         LAYERS · BASE → HEAD"}</text>
      <box width="100%" flexGrow={1} flexDirection={isWide() ? "row" : "column"}>
        <StackList
          stacks={stacks()}
          selectedStackIndex={selection().stackIndex}
          selectedPrIndex={selection().prIndex}
          terminalWidth={isWide() ? terminal().width - inspector().width - 1 : terminal().width}
          emptyMessage={emptyMessage()}
          onHoverPr={hover}
          onMovePr={hover}
          onLeavePr={() => setCardVisible(false)}
          onCheckout={checkout}
        />
        <box
          width={isWide() ? inspector().width : "100%"}
          height={isWide() ? "100%" : inspector().height}
          marginLeft={isWide() ? 1 : 0}
          backgroundColor={color("surface")}
          flexDirection="column"
        >
          <Show
            when={cardVisible() && activePr()}
            fallback={<text fg={color("muted")}>{isCompact() ? "Select a layer to inspect." : "Select or hover a layer to inspect."}</text>}
          >
            {(pr) => <PrCard pr={pr()} placement={inspector()} />}
          </Show>
        </box>
      </box>
      <Show
        when={searching()}
        fallback={<text height={1} truncate fg={feedback() ? color("focus") : color("muted")}>{feedback() || sourceState()}</text>}
      >
        <input
          width="100%"
          value={query()}
          placeholder="filter stacks, PR titles, branches, numbers, author"
          placeholderColor={color("muted")}
          textColor={color("text")}
          backgroundColor={color("surfaceRaised")}
          focusedBackgroundColor={color("surfaceRaised")}
          focusedTextColor={color("text")}
          focused
          onInput={setQuery}
        />
      </Show>
      <Show
        when={searching()}
        fallback={<text height={1} truncate fg={color("muted")}>{shortFooter()}</text>}
      >
        <text height={1} truncate fg={color("muted")}>type to filter  backspace edits  esc clears / exits</text>
      </Show>
    </box>
  )
}
