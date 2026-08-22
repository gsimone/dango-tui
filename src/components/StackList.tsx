import type { MouseEvent } from "@opentui/core"
import { color } from "../domain/colors.ts"
import type { PullRequest, Stack } from "../domain/types.ts"
import { StackRow } from "./StackRow.tsx"

export type StackListProps = {
  stacks: Stack[]
  selectedStackIndex: number
  selectedPrIndex: number
  terminalWidth: number
  emptyMessage?: string
  onHoverPr: (stackIndex: number, prIndex: number, event: MouseEvent) => void
  onMovePr: (stackIndex: number, prIndex: number, event: MouseEvent) => void
  onLeavePr: () => void
  onCheckout: (pr: PullRequest) => void
}

export function StackList(props: StackListProps) {
  return (
    <box width="100%" flexGrow={1} flexDirection="column" overflow="hidden">
      {props.stacks.length === 0 ? (
        <text fg={color("muted")}>{props.emptyMessage ?? "No fixture stack matches this filter."}</text>
      ) : (
        props.stacks.map((stack, index) => (
          <StackRow
            stack={stack}
            stackIndex={index}
            selectedStackIndex={props.selectedStackIndex}
            selectedPrIndex={props.selectedPrIndex}
            terminalWidth={props.terminalWidth}
            onHoverPr={props.onHoverPr}
            onMovePr={props.onMovePr}
            onLeavePr={props.onLeavePr}
            onCheckout={props.onCheckout}
          />
        ))
      )}
    </box>
  )
}
