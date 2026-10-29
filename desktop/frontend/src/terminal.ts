import { Terminal, type IDisposable } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { SearchAddon } from "@xterm/addon-search";
import { BlockTracker, type BlockRef } from "./blocks";
import { settings } from "./settings.svelte";
import "@xterm/xterm/css/xterm.css";
import { TerminalService } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

// Ack to Go after this many bytes have been parsed by xterm. It must stay well
// below Go's high watermark (1 MiB) or the pipe could stall waiting for an ack.
const ACK_BATCH = 128 * 1024;

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
  fit: FitAddon;
  search: SearchAddon;
  /** Re-reads theme, font, size and density from settings and refits. */
  applySettings(): void;
  /** Called with each raw chunk before it is written to xterm. */
  onChunk?: (data: Uint8Array) => void;
  /** True once the process has written anything; ssh is silent until it needs input or the login completes. */
  received: boolean;
  /** Called once, when the first bytes arrive. */
  onFirstData?: () => void;
  /** True once ssh has finished authenticating (its login marker, or a "mux" frame for a shared login). */
  authed: boolean;
  /** When (ms since the epoch) authentication was noticed, and when output last arrived. */
  authedAt: number;
  lastData: number;
  /** How authentication was noticed (for diagnostics). */
  authBy: "" | "marker" | "mux";
  /** Called once, when the shell first emits prompt markers (command blocks are active). */
  onBlocks?: () => void;
  /** Called when the user presses a block's menu button; `anchor` is the button's position. */
  onBlockMenu?: (block: BlockRef, anchor: DOMRect) => void;
  /** Called when Go reports that password auto-fill succeeded. */
  onFilled?: () => void;
  /** Called when Go reports that password auto-fill was refused. */
  onFillRefused?: (why?: string) => void;
  dispose(): void;
}

export async function openSession(el: HTMLElement, target: string, scrollback = 10000): Promise<Session> {
  await document.fonts.load(`400 ${settings.fontSize}px "JetBrains Mono"`).catch(() => []);
  // The WebGL renderer is its own chunk: fetched alongside the endpoint, not in the startup bundle.
  const [ep, webglMod] = await Promise.all([TerminalService.Endpoint(), import("@xterm/addon-webgl").catch(() => null)]);
  const term = new Terminal({ scrollback, allowProposedApi: true, ...settings.xterm });
  const fit = new FitAddon();
  const search = new SearchAddon();
  term.loadAddon(fit);
  term.loadAddon(search);
  term.open(el);
  let webgl = false;
  try {
    const gl = new webglMod!.WebglAddon();
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
  try {
    await new Promise<void>((res, rej) => {
      ws.onopen = () => res();
      ws.onerror = () => rej(new Error("terminal socket failed (origin, token or CSP)"));
    });
  } catch (e) {
    // Don't leave a half-built terminal (and its WebGL context) in the pane.
    term.dispose();
    throw e;
  }

  const enc = new TextEncoder();
  const s: Session = {
    term, ws, webgl, fit, search, received: false, authed: false, authedAt: 0, lastData: 0, authBy: "",
    applySettings() {
      Object.assign(term.options, settings.xterm);
      fit.fit();
    },
    dispose: () => { ws.close(); term.dispose(); },
  };
  const tracker = new BlockTracker(term as never, () => s.onBlocks?.(), (b, r) => s.onBlockMenu?.(b, r));
  const anchor = anchorBottom(term);
  term.onSelectionChange(() => {
    if (settings.copyOnSelect && term.hasSelection()) navigator.clipboard?.writeText(term.getSelection()).catch(() => {});
  });
  let unacked = 0;
  const authenticated = (by: "marker" | "mux") => {
    if (s.authed) return;
    s.authed = true;
    s.authBy = by;
    s.authedAt = Date.now();
    if (ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ t: "authed" }));
    }
  };
  // ssh prints this (via its LocalCommand) once the login has authenticated; the parser swallows it.
  term.parser.registerOscHandler(7771, (d) => (d === "auth" ? (authenticated("marker"), true) : false));
  ws.onmessage = (e) => {
    if (typeof e.data === "string") {
      try {
        const msg = JSON.parse(e.data) as { t?: string; why?: string };
        if (msg.t === "mux") authenticated("mux");
        else if (msg.t === "filled") s.onFilled?.();
        else if (msg.t === "fillrefused") s.onFillRefused?.(msg.why);
      } catch {}
      return;
    }
    const data = new Uint8Array(e.data as ArrayBuffer);
    s.lastData = Date.now();
    if (!s.received) {
      s.received = true;
      s.onFirstData?.();
    }
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
  const flush = setInterval(() => {
    if (unacked > 0 && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ t: "ack", n: unacked }));
      unacked = 0;
    }
  }, 50);
  term.onData((d) => ws.send(enc.encode(d)));
  term.onResize(({ cols, rows }) => ws.send(JSON.stringify({ t: "resize", c: cols, r: rows })));
  const resize = new ResizeObserver(() => fit.fit());
  resize.observe(el);
  const dispose = s.dispose;
  s.dispose = () => { clearInterval(flush); resize.disconnect(); tracker.dispose(); anchor.dispose(); dispose(); };
  return s;
}
