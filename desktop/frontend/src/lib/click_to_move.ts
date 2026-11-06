import type { IDisposable, Terminal } from "@xterm/xterm";
import { cursorMoveKeys, type BufLike } from "./clickcursor";

/** What the click handler needs from the block tracker. */
export interface PromptWatcher {
  inputStart(): { line: number; col: number } | undefined;
}

type TermLike = Pick<Terminal, "element" | "cols" | "rows" | "hasSelection"> & {
  buffer: { active: BufLike & { type: string } };
  modes: Pick<Terminal["modes"], "mouseTrackingMode" | "applicationCursorKeysMode">;
};

const DRAG_PX = 4;

/**
 * Clicking in the command being typed moves the shell's cursor there (see clickcursor.ts). It acts only on a plain
 * left click that is not the end of a drag, not on a program that handles the mouse or the alternate screen, and only
 * while a prompt is waiting for input. `send` writes keystrokes to the shell; `enabled` is the user's setting.
 */
export function installClickToMove(term: TermLike, prompts: PromptWatcher, send: (keys: string) => void, enabled: () => boolean): IDisposable {
  const el = term.element;
  if (!el) return { dispose() {} };
  let down: { x: number; y: number } | null = null;

  const onDown = (e: MouseEvent) => { down = e.button === 0 ? { x: e.clientX, y: e.clientY } : null; };
  const onUp = (e: MouseEvent) => {
    const from = down;
    down = null;
    if (!from || e.button !== 0 || e.metaKey || e.ctrlKey || e.altKey || e.shiftKey) return;
    if (Math.hypot(e.clientX - from.x, e.clientY - from.y) > DRAG_PX) return; // selecting text
    if ((e.target as Element | null)?.closest?.(".block-menu")) return;
    if (!enabled() || term.hasSelection()) return;
    const buf = term.buffer.active;
    if (buf.type !== "normal" || term.modes.mouseTrackingMode !== "none") return;
    const start = prompts.inputStart();
    const screen = el.querySelector<HTMLElement>(".xterm-screen");
    if (!start || !screen) return;

    const r = screen.getBoundingClientRect(); // includes any shift applied to anchor content to the bottom
    const col = Math.floor((e.clientX - r.left) / (r.width / term.cols));
    const row = Math.floor((e.clientY - r.top) / (r.height / term.rows));
    if (col < 0 || col >= term.cols || row < 0 || row >= term.rows) return;
    const keys = cursorMoveKeys({ buf, cols: term.cols, start, click: { line: buf.viewportY + row, col }, appCursor: term.modes.applicationCursorKeysMode });
    if (keys) send(keys);
  };

  el.addEventListener("mousedown", onDown, true);
  el.addEventListener("mouseup", onUp, true);
  return { dispose: () => { el.removeEventListener("mousedown", onDown, true); el.removeEventListener("mouseup", onUp, true); } };
}
