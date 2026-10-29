import type { IDecoration, IDisposable, IMarker, Terminal } from "@xterm/xterm";

// The slice of xterm's Terminal the tracker uses, so it can be tested with a fake.
export type TermLike = Pick<Terminal, "cols" | "registerMarker" | "onData"> & {
  buffer: { active: Pick<Terminal["buffer"]["active"], "baseY" | "cursorY" | "type"> };
  parser: { registerOscHandler(id: number, cb: (data: string) => boolean): IDisposable };
  registerDecoration(o: Parameters<Terminal["registerDecoration"]>[0]): IDecoration | undefined;
};

export function formatDuration(ms: number): string {
  if (ms < 1000) return ms < 100 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(2)}s`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`;
}

/**
 * Turns the shell's OSC 133 prompt markers into per-command blocks drawn as
 * xterm decorations. A = prompt start, B = prompt end (the typed command
 * follows), D;<exit> = command finished. The frontend sees the markers; Go never
 * interprets the stream.
 */
export class BlockTracker {
  private start?: IMarker;
  private cmd?: IMarker;
  private ran = false;
  private t0 = 0;
  /** Enter presses that arrived before their prompt was drawn (type-ahead). */
  private ahead: number[] = [];
  private active = false;
  private subs: IDisposable[] = [];

  constructor(private term: TermLike, private onFirst?: () => void) {
    this.subs.push(
      term.parser.registerOscHandler(133, (data) => this.osc(data)),
      term.onData((d) => {
        if (!/[\r\n]/.test(d)) return;
        if (this.cmd && !this.ran) {
          this.ran = true;
          this.t0 = Date.now();
        } else {
          // Enter while a command is already running, or before the prompt was
          // drawn: it belongs to the next prompt.
          this.ahead.push(Date.now());
        }
      }),
    );
  }

  private osc(data: string): boolean {
    const [kind, arg] = data.split(";");
    if (!this.active) {
      this.active = true;
      this.onFirst?.();
    }
    if (kind === "A") {
      this.reset();
      this.start = this.term.registerMarker(0);
    } else if (kind === "B") {
      this.cmd = this.term.registerMarker(0);
      const t = this.ahead.shift();
      if (t !== undefined) {
        this.ran = true;
        this.t0 = t;
      }
    } else if (kind === "D") {
      this.finish(arg === undefined || arg === "" ? 0 : Number(arg));
    }
    return true;
  }

  private reset() {
    this.start = this.cmd = undefined;
    this.ran = false;
  }

  private finish(code: number) {
    const { start, cmd, ran, t0, term } = this;
    this.reset();
    // Nothing run, no prompt seen yet, or a full-screen program: no block.
    if (!start || !cmd || !ran || term.buffer.active.type === "alternate") return;
    const buf = term.buffer.active;
    const height = Math.max(1, buf.baseY + buf.cursorY - start.line);
    const failed = code !== 0;
    const label = (failed ? `exit ${code} · ` : "") + formatDuration(Date.now() - t0);

    term.registerDecoration({ marker: start, x: 0, width: term.cols, height, layer: "top" })?.onRender((el) => {
      // classList, never className: xterm's own positioning class lives on this element,
      // and onRender fires on every redraw.
      el.classList.add("relay-block", failed ? "err" : "ok");
    });
    // x: 1, not 0: xterm applies the right anchor only for a non-zero offset.
    term.registerDecoration({ marker: cmd, anchor: "right", x: 1, width: 24, height: 1, layer: "top" })?.onRender((el) => {
      el.classList.add("relay-badge", failed ? "err" : "ok");
      el.textContent = label;
    });
  }

  dispose() {
    this.subs.forEach((s) => s.dispose());
  }
}
