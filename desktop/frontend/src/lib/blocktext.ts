// Pulls a block's command and output text out of xterm's buffer. Pure functions over a
// minimal buffer interface so they can be tested without a terminal.

export interface LineLike {
  isWrapped: boolean;
  translateToString(trimRight?: boolean, startColumn?: number, endColumn?: number): string;
}
export interface BufLike {
  length: number;
  getLine(y: number): LineLike | undefined;
}

/** Where a block's command is, as absolute buffer rows. endLine is exclusive (where the next prompt starts). */
export interface BlockRange {
  cmdLine: number;
  /** Column where the typed command starts, i.e. where the prompt ended. */
  cmdCol: number;
  endLine: number;
}

/** Joins rows[from, to) into text: a wrapped row continues the one above, any other row starts a new line. */
function join(buf: BufLike, from: number, to: number, startCol = 0): string {
  let out = "";
  for (let y = from; y < to; y++) {
    const line = buf.getLine(y);
    if (!line) break;
    // xterm only trims never-written cells; a row of spaces that was actually written (zsh draws
    // one after output with no trailing newline) must be trimmed here.
    const text = line.translateToString(true, y === from ? startCol : 0).trimEnd();
    out += y > from && !line.isWrapped ? "\n" + text : text;
  }
  return out;
}

export function blockText(buf: BufLike, r: BlockRange): { command: string; output: string } {
  if (!buf.getLine(r.cmdLine)) return { command: "", output: "" };
  // the command may have wrapped onto following rows
  let cmdEnd = r.cmdLine + 1;
  while (cmdEnd < r.endLine && buf.getLine(cmdEnd)?.isWrapped) cmdEnd++;
  const command = join(buf, r.cmdLine, cmdEnd, r.cmdCol).trim();
  // output: everything after the command up to the next prompt, minus the trailing padding rows
  const output = join(buf, cmdEnd, r.endLine).replace(/\n+$/, "").replace(/^\n+/, "");
  return { command, output };
}

/** Which buffer row a pointer is over, given the screen's top, the row height and the scroll offset. */
export function rowAt(clientY: number, screenTop: number, cellHeight: number, viewportY: number): number {
  if (clientY < screenTop || cellHeight <= 0) return -1;
  return Math.floor((clientY - screenTop) / cellHeight) + viewportY;
}
