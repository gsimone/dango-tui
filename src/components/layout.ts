export type TerminalSize = { width: number; height: number }

export type RowLayout = {
  compact: boolean
  nameWidth: number
  ballsWidth: number
  descriptionWidth: number
}

export const ROOT_PADDING_X = 1
export const LIST_START_Y = 3

export function getRowLayout(width: number, prCount: number): RowLayout {
  const contentWidth = Math.max(1, width - ROOT_PADDING_X * 2)
  const ballsWidth = prCount * 2
  const compact = width <= 50
  const desiredName = compact ? 14 : 22
  const gap = 1
  const nameWidth = Math.max(8, Math.min(desiredName, contentWidth - ballsWidth - gap))
  return {
    compact,
    nameWidth,
    ballsWidth,
    descriptionWidth: Math.max(0, contentWidth - nameWidth - ballsWidth - gap),
  }
}

export function getBallPoint(size: TerminalSize, stackIndex: number, prIndex: number, prCount: number): { x: number; y: number } {
  const layout = getRowLayout(size.width, prCount)
  return {
    x: ROOT_PADDING_X + layout.nameWidth + 1 + prIndex * 2,
    y: LIST_START_Y + stackIndex,
  }
}

export type CardPlacement = { left: number; top: number; width: number; height: number; compact: boolean }

/** A stable inspector pane: alongside the list when there is room, below it otherwise. */
export function getInspectorSize(size: TerminalSize): Pick<CardPlacement, "width" | "height" | "compact"> {
  const compact = size.width <= 50
  return {
    compact,
    width: compact ? Math.max(16, size.width - 2) : size.width >= 100 ? Math.min(48, Math.max(40, Math.floor(size.width * 0.38))) : Math.max(16, size.width - 2),
    height: compact ? 8 : 9,
  }
}

/** Keeps the sibling overlay inside the usable terminal, clear of status/footer. */
export function clampCardPlacement(size: TerminalSize, anchor: { x: number; y: number }): CardPlacement {
  const compact = size.width <= 50
  const cardWidth = Math.max(16, Math.min(compact ? 38 : 56, size.width - 2))
  const cardHeight = compact ? 8 : 9
  const maxLeft = Math.max(1, size.width - cardWidth - 1)
  const usableBottom = Math.max(1, size.height - 2)
  const below = anchor.y + 1
  const top = below + cardHeight <= usableBottom ? below : Math.max(1, anchor.y - cardHeight)
  return {
    left: Math.min(Math.max(1, anchor.x + 2), maxLeft),
    top,
    width: cardWidth,
    height: Math.min(cardHeight, Math.max(3, usableBottom - 1)),
    compact,
  }
}
