import type { MouseEvent } from "@opentui/core"
import { color } from "../domain/colors.ts"
import type { PrDisplayState } from "../domain/types.ts"

export type PrBallProps = {
  state: PrDisplayState
  selected: boolean
  isLast: boolean
  onHover: (event: MouseEvent) => void
  onMove: (event: MouseEvent) => void
  onLeave: () => void
  onCheckout: () => void
}

const stateColor = {
  merged: "merged",
  draft: "draft",
  "ci-failure": "ciFailure",
  "review-blocked": "reviewBlocked",
  queued: "queued",
  ready: "ready",
  open: "open",
} as const

/** Exactly two terminal cells wide: glyph + connector, or glyph + trailing hit cell at the head. */
export function PrBall(props: PrBallProps) {
  const activate = (event: MouseEvent) => {
    if (event.button !== 0) return
    event.stopPropagation()
    props.onCheckout()
  }

  return (
    <box
      width={2}
      height={1}
      flexDirection="row"
      backgroundColor={props.selected ? color("border") : color("surface")}
      onMouse={props.onHover}
      onMouseOver={props.onHover}
      onMouseMove={props.onMove}
      onMouseOut={props.onLeave}
      onMouseDown={activate}
    >
      <text width={1} fg={color(stateColor[props.state])}>{props.selected ? "◉" : "●"}</text>
      <text width={1} fg={color("border")}>{props.isLast ? " " : "—"}</text>
    </box>
  )
}
