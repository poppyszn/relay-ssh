import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebglAddon } from "@xterm/addon-webgl";
import "@xterm/xterm/css/xterm.css";
import { TerminalService } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";

// Ack to Go after this many bytes have been parsed by xterm. It must stay well
// below Go's high watermark (1 MiB) or the pipe could stall waiting for an ack.
const ACK_BATCH = 128 * 1024;

export interface Session {
  term: Terminal;
  ws: WebSocket;
  webgl: boolean;
  /** Called with each raw chunk before it is written to xterm. */
  onChunk?: (data: Uint8Array) => void;
  dispose(): void;
}

export async function openSession(el: HTMLElement, target: string, scrollback = 10000): Promise<Session> {
  const ep = await TerminalService.Endpoint();
  const term = new Terminal({ scrollback, fontFamily: "Menlo, monospace", fontSize: 13, allowProposedApi: true });
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
