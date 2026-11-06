// Clicking in the command you are typing moves the shell's cursor there, as in Warp. A terminal cannot place the shell's
// cursor directly; the shell owns it. So this works out how many characters apart the cursor and the click are, and
// the caller sends that many Left or Right keys, which every line editor understands.
//
// It is only used where Relay knows a prompt is waiting for input (the shell integration markers), so the keys are
// never fed to a running program, and it never presses Right through a dim autosuggestion (zsh-autosuggestions would
// accept it).

/** The slice of an xterm buffer cell this needs. */
export interface CellLike {
  getChars(): string;
  getWidth(): number;
  isDim(): number;
  isFgPalette(): number;
  getFgColor(): number;
}
export interface LineLike {
  isWrapped: boolean;
  length: number;
  getCell(x: number): CellLike | undefined;
}
export interface BufLike {
  baseY: number;
  viewportY: number;
  cursorX: number;
  cursorY: number;
  getLine(y: number): LineLike | undefined;
}

// Palette colours the common autosuggestion styles use (bright black, and the greys around 244).
const GHOST_PALETTE = new Set([8, 90, ...Array.from({ length: 12 }, (_, i) => 237 + i)]);

const isGhost = (c: CellLike) => c.isDim() !== 0 || (c.isFgPalette() !== 0 && GHOST_PALETTE.has(c.getFgColor()));
// A wide character's second cell has width 0; it is one character, not two.
const countsAsChar = (c: CellLike | undefined) => !!c && c.getWidth() > 0;

/** Characters in the line before column x. */
function charsBefore(line: LineLike, x: number): number {
  let n = 0;
  for (let i = 0; i < x && i < line.length; i++) if (countsAsChar(line.getCell(i))) n++;
  return n;
}

export interface MoveRequest {
  buf: BufLike;
  cols: number;
  /** Where the typed input begins: the buffer line and column just after the prompt. */
  start: { line: number; col: number };
  /** The clicked cell, as a buffer line and column. */
  click: { line: number; col: number };
  /** The program asked for application cursor keys (DECCKM). */
  appCursor?: boolean;
}

/** The keys that move the cursor from where it is to the click, or "" if the click should do nothing. */
export function cursorMoveKeys({ buf, start, click, appCursor }: MoveRequest): string {
  const first = buf.getLine(start.line);
  if (!first) return "";
  // The input is the first line plus the lines the terminal wrapped it onto.
  let last = start.line;
  while (buf.getLine(last + 1)?.isWrapped) last++;
  const cursorLine = buf.baseY + buf.cursorY;
  if (cursorLine < start.line || cursorLine > last || click.line < start.line || click.line > last) return "";

  const base = charsBefore(first, start.col);
  const offsetOf = (line: number, col: number): number => {
    let n = 0;
    for (let l = start.line; l < line; l++) n += charsBefore(buf.getLine(l)!, buf.getLine(l)!.length);
    return n + charsBefore(buf.getLine(line)!, col) - base;
  };

  // The end of what was typed: the last cell that is not blank and not a dim suggestion.
  let end = 0;
  for (let l = last; l >= start.line; l--) {
    const ln = buf.getLine(l)!;
    for (let x = ln.length - 1; x >= (l === start.line ? start.col : 0); x--) {
      const c = ln.getCell(x);
      if (c && countsAsChar(c) && c.getChars().trim() !== "" && !isGhost(c)) {
        end = offsetOf(l, x + 1);
        l = -1; // found: stop both loops
        break;
      }
    }
  }

  const current = offsetOf(cursorLine, buf.cursorX);
  const target = Math.max(0, Math.min(end, offsetOf(click.line, click.col)));
  const delta = target - current;
  if (delta === 0) return "";
  const right = appCursor ? "\x1bOC" : "\x1b[C";
  const left = appCursor ? "\x1bOD" : "\x1b[D";
  return (delta > 0 ? right : left).repeat(Math.abs(delta));
}
