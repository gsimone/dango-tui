import type { MouseEvent } from "@opentui/core"
import { color } from "../domain/colors.ts"
import { getDisplayState } from "../domain/displayState.ts"
import type { PullRequest, Stack } from "../domain/types.ts"
import { PrBall } from "./PrBall.tsx"
import { getRowLayout } from "./layout.ts"

export type StackRowProps = {
  stack: Stack
  stackIndex: number
  selectedStackIndex: number
  selectedPrIndex: number
  terminalWidth: number
  onHoverPr: (stackIndex: number, prIndex: number, event: MouseEvent) => void
  onMovePr: (stackIndex: number, prIndex: number, event: MouseEvent) => void
  onLeavePr: () => void
  onCheckout: (pr: PullRequest) => void
}

const clip = (value: string, length: number) => value.length <= length ? value : `${value.slice(0, Math.max(0, length - 1))}…`

const stackHealth = (stack: Stack) => {
  const head = stack.prs.at(-1)
  if (!head) return "no layers"
  const state = getDisplayState(head)
  if (state === "ready") return "head ready"
  if (state === "ci-failure") return "head CI failed"
  if (state === "review-blocked") return "head blocked"
  if (state === "queued") return "head queued"
  if (state === "draft") return "head draft"
  if (state === "merged") return "merged"
  return "head pending"
}

export function StackRow(props: StackRowProps) {
  const layout = () => getRowLayout(props.terminalWidth, props.stack.prs.length)
  const isSelectedStack = () => props.stackIndex === props.selectedStackIndex
  const stackName = () => `${isSelectedStack() ? "▸" : "·"} ${props.stack.name}`
  const summary = () => `${stackHealth(props.stack)} · ${props.stack.description ?? ""}`

  return (
    <box width="100%" height={1} flexDirection="row" backgroundColor={isSelectedStack() ? color("surfaceRaised") : color("surface")}>
      <text width={layout().nameWidth} truncate fg={isSelectedStack() ? color("text") : color("muted")}>
        {clip(stackName(), layout().nameWidth)}
      </text>
      <box width={layout().ballsWidth} height={1} marginLeft={1} flexDirection="row">
        {props.stack.prs.map((pr, prIndex) => (
          <PrBall
            state={getDisplayState(pr)}
            selected={isSelectedStack() && prIndex === props.selectedPrIndex}
            isLast={prIndex === props.stack.prs.length - 1}
            onHover={(event) => props.onHoverPr(props.stackIndex, prIndex, event)}
            onMove={(event) => props.onMovePr(props.stackIndex, prIndex, event)}
            onLeave={props.onLeavePr}
            onCheckout={() => props.onCheckout(pr)}
          />
        ))}
      </box>
      {!layout().compact && (
        <text flexGrow={1} marginLeft={1} truncate fg={color("muted")}>
          {summary()}
        </text>
      )}
    </box>
  )
}
