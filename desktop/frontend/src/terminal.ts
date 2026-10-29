import { Terminal, type IDisposable } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebglAddon } from "@xterm/addon-webgl";
import { BlockTracker } from "./blocks";
import "@xterm/xterm/css/xterm.css";
import { TerminalService } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";

// Ack to Go after this many bytes have been parsed by xterm. It must stay well
// below Go's high watermark (1 MiB) or the pipe could stall waiting for an ack.
const ACK_BATCH = 128 * 1024;

// Matches theme.css: deep navy background, soft blue accent.
const THEME = {
  background: "#1b2634", foreground: "#d3dcea", cursor: "#f07b86", selectionBackground: "#2d4466",
  black: "#1f2a3a", red: "#f07b86", green: "#9ad07a", yellow: "#e6c07b", blue: "#4aa8ee", magenta: "#c792ea", cyan: "#56d4dd", white: "#d3dcea",
  brightBlack: "#5c6b82", brightRed: "#ff939d", brightGreen: "#b4e394", brightYellow: "#f3d398", brightBlue: "#7cc0f5", brightMagenta: "#d9aaf2", brightCyan: "#7ee3ea", brightWhite: "#ffffff",
};

/**
 * Keeps content at the bottom of the pane until the buffer fills, the way a
 * block terminal starts: new output pushes older output up and the top stays
 * empty. xterm has no such mode, so the screen is shifted down by the unused
 * rows. Full-screen programs and a scrolled buffer are left alone.
 */
export function anchorBottom(term: Terminal): IDisposable {
  const apply = () => {
    const screen = term.element?.querySelector<HTMLElement>(".xterm-screen");
    if (!screen || !term.rows) return;
    const buf = term.buffer.active;
    let shift = 0;
    if (buf.type === "normal" && buf.baseY === 0) {
      shift = Math.max(0, term.rows - (buf.cursorY + 1)) * (screen.clientHeight / term.rows);
    }
    screen.style.transform = shift > 0 ? `translateY(${Math.round(shift)}px)` : "";
  };
  // Command blocks bleed to the pane edges. The cell grid rarely fills the pane
  // exactly (padding plus leftover pixels), so measure how far the screen sits
  // from each edge and expose it to CSS.
  const bleed = () => {
    const el = term.element;
    const screen = el?.querySelector<HTMLElement>(".xterm-screen");
    if (!el || !screen) return;
    const e = el.getBoundingClientRect();
    const s = screen.getBoundingClientRect();
    el.style.setProperty("--bleed-l", `${Math.max(0, Math.round(s.left - e.left))}px`);
    el.style.setProperty("--bleed-r", `${Math.max(0, Math.round(e.right - s.right))}px`);
  };
  const ro = term.element ? new ResizeObserver(bleed) : undefined;
  if (term.element) ro?.observe(term.element);
  const subs = [term.onWriteParsed(apply), term.onCursorMove(apply), term.onResize(() => { apply(); bleed(); }), term.onScroll(apply)];
  apply();
  bleed();
  return { dispose: () => { ro?.disconnect(); subs.forEach((s) => s.dispose()); } };
}

export interface Session {
  term: Terminal;
  ws: WebSocket;
  webgl: boolean;
  /** Called with each raw chunk before it is written to xterm. */
  onChunk?: (data: Uint8Array) => void;
  /** Called once, when the shell first emits prompt markers (command blocks are active). */
  onBlocks?: () => void;
  dispose(): void;
}

export async function openSession(el: HTMLElement, target: string, scrollback = 10000): Promise<Session> {
  await document.fonts.load('400 13px "JetBrains Mono"').catch(() => []);
  const ep = await TerminalService.Endpoint();
  const term = new Terminal({ scrollback, fontFamily: '"JetBrains Mono", "SF Mono", Menlo, Consolas, monospace', fontSize: 14, lineHeight: 1.25, allowProposedApi: true, cursorBlink: true, theme: THEME });
  const fit = new FitAddon();
  term.loadAddon(fit);
  term.open(el);
  let webgl = false;
  try {
    const gl = new WebglAddon();
    gl.onContextLoss(() => gl.dispose());
    term.loadAddon(gl);
    webgl = true;
  } catch {
    // Falls back to xterm's DOM/canvas renderer.
  }
  fit.fit();

  const q = new URLSearchParams({ token: ep.token, target, cols: String(term.cols), rows: String(term.rows) });
  const ws = new WebSocket(`${ep.url}?${q}`);
  ws.binaryType = "arraybuffer";
  await new Promise<void>((res, rej) => {
    ws.onopen = () => res();
    ws.onerror = () => rej(new Error("terminal socket failed (origin, token or CSP)"));
  });

  const enc = new TextEncoder();
  const s: Session = { term, ws, webgl, dispose: () => { ws.close(); term.dispose(); } };
  const tracker = new BlockTracker(term as never, () => s.onBlocks?.());
  const anchor = anchorBottom(term);
  const dispose = s.dispose;
  s.dispose = () => { tracker.dispose(); anchor.dispose(); dispose(); };
  let unacked = 0;
  ws.onmessage = (e) => {
    const data = new Uint8Array(e.data as ArrayBuffer);
    s.onChunk?.(data);
    term.write(data, () => {
      unacked += data.length;
      if (unacked >= ACK_BATCH) {
        ws.send(JSON.stringify({ t: "ack", n: unacked }));
        unacked = 0;
      }
    });
  };
  // Flush the remainder when the stream goes quiet so Go's counter never drifts.
  setInterval(() => {
    if (unacked > 0 && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ t: "ack", n: unacked }));
      unacked = 0;
    }
  }, 50);
  term.onData((d) => ws.send(enc.encode(d)));
  term.onResize(({ cols, rows }) => ws.send(JSON.stringify({ t: "resize", c: cols, r: rows })));
  new ResizeObserver(() => fit.fit()).observe(el);
  return s;
}
