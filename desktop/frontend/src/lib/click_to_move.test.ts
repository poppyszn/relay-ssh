// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from "vitest";
import { installClickToMove } from "./click_to_move";
import type { BufLike, CellLike, LineLike } from "./clickcursor";

const COLS = 20, ROWS = 10;
function line(text: string): LineLike {
  const cells: CellLike[] = [...text].map((ch) => ({ getChars: () => ch, getWidth: () => 1, isDim: () => 0, isFgPalette: () => 0, getFgColor: () => 0 }));
  while (cells.length < COLS) cells.push({ getChars: () => "", getWidth: () => 1, isDim: () => 0, isFgPalette: () => 0, getFgColor: () => 0 });
  return { isWrapped: false, length: COLS, getCell: (x) => cells[x] };
}

afterEach(() => { document.body.innerHTML = ""; });

function setup(o: { type?: string; mouse?: string; selection?: boolean; start?: { line: number; col: number } | undefined; enabled?: boolean; appCursor?: boolean } = {}) {
  const element = document.createElement("div");
  const screen = document.createElement("div");
  screen.className = "xterm-screen";
  element.append(screen);
  document.body.append(element);
  // 200 x 100 px screen: 10 px per column, 10 px per row
  screen.getBoundingClientRect = () => ({ left: 0, top: 0, width: 200, height: 100, right: 200, bottom: 100, x: 0, y: 0, toJSON() {} }) as DOMRect;
  const lines = [line("$ git status")];
  const buffer = { baseY: 0, viewportY: 0, cursorX: 12, cursorY: 0, type: o.type ?? "normal", getLine: (y: number) => lines[y] } as BufLike & { type: string };
  const term = { element, cols: COLS, rows: ROWS, buffer: { active: buffer }, modes: { mouseTrackingMode: o.mouse ?? "none", applicationCursorKeysMode: !!o.appCursor }, hasSelection: () => !!o.selection } as never;
  const sent: string[] = [];
  const tracker = { inputStart: () => ("start" in o ? o.start : { line: 0, col: 2 }) };
  const dispose = installClickToMove(term, tracker, (s) => sent.push(s), () => o.enabled ?? true);
  const mouse = (type: string, x: number, y: number, extra: MouseEventInit = {}) =>
    screen.dispatchEvent(new MouseEvent(type, { clientX: x, clientY: y, button: 0, bubbles: true, ...extra }));
  const click = (x: number, y: number, extra: MouseEventInit = {}) => { mouse("mousedown", x, y, extra); mouse("mouseup", x, y, extra); };
  return { sent, click, mouse, screen, dispose };
}

describe("clicking in the prompt line", () => {
  it("sends the arrow keys that move the cursor to the click", () => {
    const t = setup();
    t.click(65, 5); // column 6 on row 0
    expect(t.sent).toEqual(["\x1b[D".repeat(6)]);
  });

  it("uses application cursor keys when the shell asked for them", () => {
    const t = setup({ appCursor: true });
    t.click(65, 5);
    expect(t.sent).toEqual(["\x1bOD".repeat(6)]);
  });

  for (const [why, o] of [
    ["a program is running (no prompt waiting)", { start: undefined }],
    ["the alternate screen is up (vim, less)", { type: "alternate" }],
    ["the program reports the mouse itself (tmux, htop)", { mouse: "vt200" }],
    ["text is selected", { selection: true }],
    ["the setting is off", { enabled: false }],
  ] as const) {
    it(`does nothing when ${why}`, () => {
      const t = setup(o as never);
      t.click(65, 5);
      expect(t.sent).toEqual([]);
    });
  }

  it("does nothing for a drag (selecting text), a right click, or a modified click", () => {
    const t = setup();
    t.mouse("mousedown", 65, 5); t.mouse("mouseup", 120, 5); // dragged 55 px
    t.click(65, 5, { button: 2 });
    for (const mod of [{ metaKey: true }, { ctrlKey: true }, { altKey: true }, { shiftKey: true }]) t.click(65, 5, mod);
    expect(t.sent).toEqual([]);
  });

  it("ignores a click on a block's menu button", () => {
    const t = setup();
    const btn = document.createElement("button");
    btn.className = "block-menu";
    t.screen.append(btn);
    btn.dispatchEvent(new MouseEvent("mousedown", { clientX: 65, clientY: 5, bubbles: true }));
    btn.dispatchEvent(new MouseEvent("mouseup", { clientX: 65, clientY: 5, bubbles: true }));
    expect(t.sent).toEqual([]);
  });

  it("stops when disposed", () => {
    const t = setup();
    t.dispose.dispose();
    t.click(65, 5);
    expect(t.sent).toEqual([]);
  });
});
