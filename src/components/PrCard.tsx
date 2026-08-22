import { color, type ColorToken } from "../domain/colors.ts"
import { displayStateDetail, displayStateLabel, getDisplayState } from "../domain/displayState.ts"
import type { PullRequest } from "../domain/types.ts"
import type { CardPlacement } from "./layout.ts"

export type PrCardProps = { pr: PullRequest; placement: Pick<CardPlacement, "width" | "height" | "compact"> }

const clip = (value: string, max: number) => value.length <= max ? value : `${value.slice(0, Math.max(0, max - 1))}…`

const ciLine = (pr: PullRequest) => {
  if (pr.ci.state === "success") return `CI ✓ ${pr.ci.total || "all"} checks`
  if (pr.ci.state === "failure") return `CI × ${pr.ci.failed || 1} failed · ${pr.ci.total} total`
  if (pr.ci.state === "pending") return `CI ◌ ${pr.ci.pending || 1} pending · ${pr.ci.total} total`
  return "CI — not reported"
}

const reviewLine = (pr: PullRequest) => {
  if (pr.mergeable === false) return "Review ! merge conflict"
  if (pr.changesRequested) return "Review ! changes requested"
  if (pr.approvals > 0) return `Review ✓ ${pr.approvals} approval${pr.approvals === 1 ? "" : "s"}`
  return "Review ◌ no decision yet"
}

export function PrCard(props: PrCardProps) {
  const state = () => getDisplayState(props.pr)
  const maxLine = () => Math.max(8, props.placement.width - 4)
  const stateColor = (): ColorToken => {
    const value = state()
    return value === "ci-failure" ? "ciFailure" : value === "review-blocked" ? "reviewBlocked" : value
  }

  return (
    <box
      width={props.placement.width}
      height={props.placement.height}
      zIndex={20}
      border
      borderColor={color("focus")}
      backgroundColor={color("surfaceRaised")}
      paddingX={1}
      flexDirection="column"
    >
      <text truncate fg={color("text")}>{clip(`#${props.pr.number} ${props.pr.title}`, maxLine())}</text>
      <text truncate fg={color(stateColor())}>{clip(`${displayStateLabel[state()]} · ${displayStateDetail(props.pr)}`, maxLine())}</text>
      {props.placement.compact ? (
        <>
          <text truncate fg={color("muted")}>{clip(`${ciLine(props.pr)} · ${reviewLine(props.pr)}`, maxLine())}</text>
          <text truncate fg={color("text")}>{clip(`+${props.pr.additions} −${props.pr.deletions} · ${props.pr.changedFiles} files`, maxLine())}</text>
          <text truncate fg={color("muted")}>{clip(props.pr.branch, maxLine())}</text>
          <text truncate fg={color("focus")}>click checkout · o open</text>
        </>
      ) : (
        <>
          <text truncate fg={color("muted")}>{clip(ciLine(props.pr), maxLine())}</text>
          <text truncate fg={color("muted")}>{clip(reviewLine(props.pr), maxLine())}</text>
          <text truncate fg={color("text")}>{clip(`+${props.pr.additions} −${props.pr.deletions} · ${props.pr.changedFiles} files`, maxLine())}</text>
          <text truncate fg={color("muted")}>{clip(props.pr.branch, maxLine())}</text>
          <text truncate fg={color("focus")}>click checkout · o open</text>
        </>
      )}
    </box>
  )
}
