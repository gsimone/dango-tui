import type { Stack } from "../domain/types.ts"

export type Selection = { stackIndex: number; prIndex: number }

export type AppState = {
  selection: Selection
  query: string
  searching: boolean
  cardVisible: boolean
  feedback: string
}

export const initialAppState = (): AppState => ({
  selection: { stackIndex: 0, prIndex: 0 },
  query: "",
  searching: false,
  cardVisible: true,
  feedback: "",
})

export function filterStacks(stacks: Stack[], query: string): Stack[] {
  const needle = query.trim().toLowerCase()
  if (!needle) return stacks
  return stacks.filter((stack) => {
    const stackText = `${stack.number} ${stack.name} ${stack.description ?? ""} ${stack.baseRef}`.toLowerCase()
    return stackText.includes(needle) || stack.prs.some((pr) => `${pr.number} ${pr.title} ${pr.branch} ${pr.author}`.toLowerCase().includes(needle))
  })
}

export function clampSelection(selection: Selection, stacks: Stack[]): Selection {
  if (stacks.length === 0) return { stackIndex: 0, prIndex: 0 }
  const stackIndex = Math.min(Math.max(selection.stackIndex, 0), stacks.length - 1)
  const prs = stacks[stackIndex]?.prs ?? []
  return { stackIndex, prIndex: Math.min(Math.max(selection.prIndex, 0), Math.max(0, prs.length - 1)) }
}

export function moveSelection(selection: Selection, stacks: Stack[], direction: "up" | "down" | "left" | "right" | "home" | "end"): Selection {
  const current = clampSelection(selection, stacks)
  if (direction === "left") return clampSelection({ ...current, prIndex: current.prIndex - 1 }, stacks)
  if (direction === "right") return clampSelection({ ...current, prIndex: current.prIndex + 1 }, stacks)
  if (direction === "home") return { ...current, prIndex: 0 }
  if (direction === "end") return { ...current, prIndex: Math.max(0, (stacks[current.stackIndex]?.prs.length ?? 1) - 1) }
  return clampSelection({ stackIndex: current.stackIndex + (direction === "up" ? -1 : 1), prIndex: current.prIndex }, stacks)
}
