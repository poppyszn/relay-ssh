import { describe, expect, it } from "vitest";
import { cursorMoveKeys, type BufLike, type CellLike, type LineLike } from "./clickcursor";

const COLS = 20;
const wide = (ch: string) => ch.codePointAt(0)! > 0x2e80;

// A line of cells from text. `ghostFrom` marks the cells from that column on as a dim autosuggestion.
function line(text: string, o: { wrapped?: boolean; ghostFrom?: number } = {}): LineLike {
  const cells: CellLike[] = [];
  for (const ch of [...text]) {
    const ghost = o.ghostFrom !== undefined && cells.length >= o.ghostFrom;
    cells.push({ getChars: () => ch, getWidth: () => (wide(ch) ? 2 : 1), isDim: () => (ghost ? 1 : 0), isFgPalette: () => 0, getFgColor: () => 0 });
    if (wide(ch)) cells.push({ getChars: () => "", getWidth: () => 0, isDim: () => 0, isFgPalette: () => 0, getFgColor: () => 0 });
  }
  while (cells.length < COLS) cells.push({ getChars: () => "", getWidth: () => 1, isDim: () => 0, isFgPalette: () => 0, getFgColor: () => 0 });
  return { isWrapped: !!o.wrapped, length: cells.length, getCell: (x) => cells[x] };
}

const buf = (lines: LineLike[], cursorX: number, cursorY: number): BufLike => ({ baseY: 0, viewportY: 0, cursorX, cursorY, getLine: (y) => lines[y] });
const RIGHT = "\x1b[C", LEFT = "\x1b[D";
// A prompt "$ " (2 cells) and the typed text "git status" after it, on row 0.
const PROMPT = 2;
const start = { line: 0, col: PROMPT };

describe("click to move the cursor at the prompt", () => {
  const typed = [line("$ git status")];

  it("moves left to where you clicked", () => {
    expect(cursorMoveKeys({ buf: buf(typed, 12, 0), cols: COLS, start, click: { line: 0, col: 6 } })).toBe(LEFT.repeat(6));
  });

  it("moves right to where you clicked", () => {
    expect(cursorMoveKeys({ buf: buf(typed, 4, 0), cols: COLS, start, click: { line: 0, col: 9 } })).toBe(RIGHT.repeat(5));
  });

  it("does nothing when you click where the cursor already is", () => {
    expect(cursorMoveKeys({ buf: buf(typed, 8, 0), cols: COLS, start, click: { line: 0, col: 8 } })).toBe("");
  });

  it("clicking past the end of the text goes to the end of the text, not further", () => {
    expect(cursorMoveKeys({ buf: buf(typed, 4, 0), cols: COLS, start, click: { line: 0, col: 19 } })).toBe(RIGHT.repeat(8)); // to column 12
  });

  it("clicking on the prompt itself goes to the start of the input, not into the prompt", () => {
    expect(cursorMoveKeys({ buf: buf(typed, 8, 0), cols: COLS, start, click: { line: 0, col: 0 } })).toBe(LEFT.repeat(6));
  });

  it("never presses Right through a dim autosuggestion (zsh would accept it)", () => {
    const withSuggestion = [line("$ git status --short", { ghostFrom: 12 })]; // "git status" typed, " --short" suggested
    expect(cursorMoveKeys({ buf: buf(withSuggestion, 4, 0), cols: COLS, start, click: { line: 0, col: 19 } })).toBe(RIGHT.repeat(8));
    // already at the end of what is typed: a click in the suggestion moves nowhere
    expect(cursorMoveKeys({ buf: buf(withSuggestion, 12, 0), cols: COLS, start, click: { line: 0, col: 17 } })).toBe("");
  });

  it("uses the application cursor key codes when the program asked for them", () => {
    expect(cursorMoveKeys({ buf: buf(typed, 12, 0), cols: COLS, start, click: { line: 0, col: 10 }, appCursor: true })).toBe("\x1bOD".repeat(2));
    expect(cursorMoveKeys({ buf: buf(typed, 4, 0), cols: COLS, start, click: { line: 0, col: 6 }, appCursor: true })).toBe("\x1bOC".repeat(2));
  });

  it("follows a command that wrapped onto the next row", () => {
    // 18 typed characters after the prompt fill row 0 (20 cols) and continue on row 1
    const wrapped = [line("$ abcdefghijklmnopqr"), line("stuvwx", { wrapped: true })];
    // cursor at the end (row 1, col 6); click on row 0 col 5 (character 'd'): 6 + 20 - 2 - 5... the offset difference
    const keys = cursorMoveKeys({ buf: buf(wrapped, 6, 1), cols: COLS, start, click: { line: 0, col: 5 } });
    expect(keys).toBe(LEFT.repeat(18 + 6 - 3)); // end offset 24, 'd' is character 3
  });

  it("counts characters, not cells, for wide characters", () => {
    const cjk = [line("$ 日本語abc")]; // 3 wide characters (6 cells) then abc
    // cursor after 'abc' (col 2+6+3 = 11); click on 'a' (col 8): 3 characters back
    expect(cursorMoveKeys({ buf: buf(cjk, 11, 0), cols: COLS, start, click: { line: 0, col: 8 } })).toBe(LEFT.repeat(3));
  });

  it("does nothing for a click outside the command being typed", () => {
    const earlier = [line("previous output"), line("$ git status")];
    const s = { line: 1, col: PROMPT };
    expect(cursorMoveKeys({ buf: buf(earlier, 12, 1), cols: COLS, start: s, click: { line: 0, col: 3 } })).toBe("");
  });

  it("does nothing if the cursor is not in the input (a program is drawing elsewhere)", () => {
    expect(cursorMoveKeys({ buf: buf(typed, 3, 5), cols: COLS, start, click: { line: 0, col: 6 } })).toBe("");
  });

  it("treats an empty input as having nothing to move through", () => {
    expect(cursorMoveKeys({ buf: buf([line("$ ")], 2, 0), cols: COLS, start, click: { line: 0, col: 10 } })).toBe("");
  });
});
