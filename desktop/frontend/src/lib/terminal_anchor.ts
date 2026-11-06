// Content sits at the bottom of the pane until the buffer fills, the way a block terminal starts. The screen is
// shifted down by the unused rows, so the shift needs "how many rows are in use". That must come from the content,
// not from where the cursor happens to be: programs that redraw in place (docker pull, pip, npm, curl progress) move the
// cursor up N lines, rewrite, and move back, many times a second, and following the cursor made the whole screen
// jump up and down with each update.

/** The slice of an xterm buffer this needs. */
export interface AnchorBuffer {
  type: string;
  baseY: number;
  cursorY: number;
  getLine(y: number): { translateToString(trimRight?: boolean): string } | undefined;
}

/** Rows from the top that hold anything: the last row with text, or the cursor's row if that is lower. */
export function rowsInUse(buf: AnchorBuffer, rows: number): number {
  let last = -1;
  for (let y = rows - 1; y >= 0; y--) {
    const text = buf.getLine(buf.baseY + y)?.translateToString(true);
    if (text && text.trim() !== "") { last = y; break; }
  }
  return Math.max(last + 1, buf.cursorY + 1);
}

/**
 * The rows to keep anchored. It only grows as content does, so an in-place redraw (the cursor moving up, a line
 * briefly erased) cannot make the screen jump; it starts over when the screen is cleared (nothing but a prompt on the
 * first row).
 */
export function nextExtent(prev: number, buf: AnchorBuffer, rows: number): number {
  const used = rowsInUse(buf, rows);
  if (used <= 1 && buf.cursorY === 0) return used;
  return Math.min(rows, Math.max(prev, used));
}
