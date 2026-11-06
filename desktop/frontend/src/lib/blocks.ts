import type { IDecoration, IDisposable, IMarker, Terminal } from "@xterm/xterm";
import { blockText, rowAt } from "./blocktext";

// The slice of xterm's Terminal the tracker uses, so it can be tested with a fake.
export type TermLike = Pick<Terminal, "cols" | "rows" | "registerMarker" | "onData"> & {
  /** The terminal's DOM element; absent in tests, where hover tracking is skipped. */
  element?: HTMLElement;
  buffer: {
    active: Pick<Terminal["buffer"]["active"], "baseY" | "cursorX" | "cursorY" | "viewportY" | "type" | "length" | "getLine">;
    /** Fires when a full-screen program switches to or from the alternate screen. Absent in some fakes. */
    onBufferChange?: (cb: () => void) => IDisposable;
  };
  parser: { registerOscHandler(id: number, cb: (data: string) => boolean): IDisposable };
  registerDecoration(o: Parameters<Terminal["registerDecoration"]>[0]): IDecoration | undefined;
};

export function formatDuration(ms: number): string {
  if (ms < 1000) return ms < 100 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(2)}s`;
  if (ms < 60_000) return `${(ms / 1000).toFixed(1)}s`;
  return `${Math.floor(ms / 60_000)}m ${Math.round((ms % 60_000) / 1000)}s`;
}

/** What the block menu can copy from one block. */
export interface BlockRef {
  failed: boolean;
  command(): string;
  output(): string;
  /** The command, then its output. */
  all(): string;
}

interface Block {
  start: IMarker;
  cmd: IMarker;
  /** Marks where the next prompt begins, i.e. the end of this block. */
  end: IMarker;
  cmdCol: number;
  failed: boolean;
  badge?: HTMLElement;
}

/** Where a prompt's command will run, from the shell's "A;<branch>;<cwd>" marker. */
export interface PromptContext {
  branch: string;
  cwd: string;
}

/** Parses the argument of an A marker. Old hooks send none; the cwd may itself contain ";". */
export function parseContext(arg: string | undefined): PromptContext | undefined {
  if (!arg) return undefined;
  const i = arg.indexOf(";");
  if (i < 0) return undefined;
  const ctx = { branch: arg.slice(0, i), cwd: arg.slice(i + 1) };
  return ctx.branch || ctx.cwd ? ctx : undefined;
}

const BRANCH = '<svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="6" cy="5" r="2.2"/><circle cx="6" cy="19" r="2.2"/><circle cx="18" cy="8" r="2.2"/><path d="M6 7.2v9.6M18 10.2c0 4-6 3-11 7"/></svg>';

const DOTS = '<svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><circle cx="5" cy="12" r="1.7"/><circle cx="12" cy="12" r="1.7"/><circle cx="19" cy="12" r="1.7"/></svg>';

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
  /** A finished command waiting for the next prompt, so its block can include the padding row after it. */
  private done?: { start: IMarker; cmd: IMarker; cmdCol: number; code: number; ms: number; ctx?: PromptContext };
  private cmdCol = 0;
  /** The pending prompt's directory and branch, shown as the block header. */
  private ctx?: PromptContext;
  private active = false;
  private subs: IDisposable[] = [];
  private blocks: Block[] = [];
  private hot?: Block;

  constructor(private term: TermLike, private onFirst?: () => void, private onMenu?: (ref: BlockRef, anchor: DOMRect) => void) {
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
    this.trackHover();
    this.hideOnAlternateScreen();
  }

  /**
   * Blocks are overlays tied to rows of the normal screen. A full-screen program (vim, less, htop) runs on the
   * alternate screen, and the overlays would be drawn over it (backgrounds, headers, badges), so they are hidden by
   * a class on the terminal element for as long as it is active.
   */
  private hideOnAlternateScreen() {
    const el = this.term.element;
    if (!el) return;
    const sync = () => el.classList.toggle("relay-alt", this.term.buffer.active.type === "alternate");
    sync();
    const sub = this.term.buffer.onBufferChange?.(sync);
    this.subs.push({ dispose: () => { sub?.dispose(); el.classList.remove("relay-alt"); } });
  }

  /**
   * Where typed input begins (the buffer line and column just after the prompt), but only while a prompt is waiting
   * for it: between the prompt and Enter. Clicking may move the cursor then; at any other time keys would go to a
   * running program.
   */
  inputStart(): { line: number; col: number } | undefined {
    if (!this.cmd || this.ran || this.cmd.isDisposed || this.cmd.line < 0) return undefined;
    return { line: this.cmd.line, col: this.cmdCol };
  }

  /** Marks the block under the pointer so its menu button shows. */
  private trackHover() {
    const el = this.term.element;
    if (!el) return;
    const move = (e: MouseEvent) => {
      const screen = el.querySelector<HTMLElement>(".xterm-screen");
      if (!screen) return;
      const r = screen.getBoundingClientRect();
      const row = rowAt(e.clientY, r.top, r.height / this.term.rows, this.term.buffer.active.viewportY);
      this.setHot(this.blocks.find((b) => b.start.line >= 0 && b.start.line <= row && row < b.end.line));
    };
    const leave = () => this.setHot(undefined);
    el.addEventListener("mousemove", move);
    el.addEventListener("mouseleave", leave);
    this.subs.push({ dispose: () => { el.removeEventListener("mousemove", move); el.removeEventListener("mouseleave", leave); } });
  }

  private setHot(b?: Block) {
    if (b === this.hot) return;
    this.hot?.badge?.classList.remove("hot");
    this.hot = b;
    b?.badge?.classList.add("hot");
  }

  private ref(b: Block): BlockRef {
    const text = () => (b.start.isDisposed || b.cmd.isDisposed || b.end.isDisposed
      ? { command: "", output: "" }
      : blockText(this.term.buffer.active, { cmdLine: b.cmd.line, cmdCol: b.cmdCol, endLine: b.end.line }));
    return {
      failed: b.failed,
      command: () => text().command,
      output: () => text().output,
      all: () => { const t = text(); return t.output ? `${t.command}\n${t.output}` : t.command; },
    };
  }

  private osc(data: string): boolean {
    const [kind, ...rest] = data.split(";");
    const arg = rest.join(";");
    if (!this.active) {
      this.active = true;
      this.onFirst?.();
    }
    if (kind === "A") {
      this.draw();
      this.reset();
      this.start = this.term.registerMarker(0);
      this.ctx = parseContext(arg);
    } else if (kind === "B") {
      this.cmd = this.term.registerMarker(0);
      this.cmdCol = this.term.buffer.active.cursorX;
      const t = this.ahead.shift();
      if (t !== undefined) {
        this.ran = true;
        this.t0 = t;
      }
    } else if (kind === "D") {
      this.finish(arg === "" ? 0 : Number(arg));
    }
    return true;
  }

  private reset() {
    this.start = this.cmd = this.ctx = undefined;
    this.ran = false;
  }

  private finish(code: number) {
    const { start, cmd, ran, t0, term, ctx } = this;
    this.reset();
    this.done = undefined;
    // Nothing run, no prompt seen yet, or a full-screen program: no block.
    if (!start || !cmd || !ran || term.buffer.active.type === "alternate") return;
    this.done = { start, cmd, cmdCol: this.cmdCol, code, ms: Date.now() - t0, ctx };
  }

  /** Draws the pending block, ending where the next prompt begins. */
  private draw() {
    const d = this.done;
    this.done = undefined;
    if (!d) return;
    const { term } = this;
    const buf = term.buffer.active;
    const height = Math.max(1, buf.baseY + buf.cursorY - d.start.line);
    const failed = d.code !== 0;
    const label = (failed ? `exit ${d.code} · ` : "") + formatDuration(d.ms);

    term.registerDecoration({ marker: d.start, x: 0, width: term.cols, height, layer: "top" })?.onRender((el) => {
      // classList, never className: xterm's own positioning class lives on this element,
      // and onRender fires on every redraw.
      el.classList.add("relay-block", failed ? "err" : "ok");
    });
    if (d.ctx) this.drawHeader(d.start, d.ctx);
    const end = term.registerMarker(0);
    const block: Block | undefined = end ? { start: d.start, cmd: d.cmd, end, cmdCol: d.cmdCol, failed } : undefined;
    if (block) {
      this.blocks.push(block);
      end.onDispose(() => (this.blocks = this.blocks.filter((x) => x !== block)));
    }
    // x: 1, not 0: xterm applies the right anchor only for a non-zero offset.
    term.registerDecoration({ marker: d.cmd, anchor: "right", x: 1, width: 24, height: 1, layer: "top" })?.onRender((el) => {
      el.classList.add("relay-badge", failed ? "err" : "ok");
      // Built once; onRender fires on every redraw and must not wipe the button.
      let text = el.querySelector<HTMLElement>(".badge-text");
      if (!text) {
        text = document.createElement("span");
        text.className = "badge-text";
        el.append(text);
        if (block) {
          const btn = document.createElement("button");
          btn.className = "block-menu";
          btn.type = "button";
          btn.setAttribute("aria-label", "Block options");
          btn.innerHTML = DOTS;
          // keep the terminal from starting a text selection under the button
          btn.addEventListener("mousedown", (e) => e.stopPropagation());
          btn.addEventListener("click", (e) => { e.stopPropagation(); this.onMenu?.(this.ref(block), btn.getBoundingClientRect()); });
          el.append(btn);
        }
      }
      text.textContent = label;
      if (block) {
        block.badge = el;
        el.classList.toggle("hot", block === this.hot);
      }
    });
  }

  /** The block's first row is the blank padding row above the prompt: the header goes there. */
  private drawHeader(start: IMarker, ctx: PromptContext) {
    this.term.registerDecoration({ marker: start, x: 0, width: this.term.cols, height: 1, layer: "top" })?.onRender((el) => {
      el.classList.add("relay-head");
      if (el.firstChild) return; // built once; onRender fires on every redraw
      if (ctx.cwd) {
        const path = document.createElement("span");
        path.className = "head-path";
        path.textContent = ctx.cwd;
        el.append(path);
      }
      if (ctx.branch) {
        const branch = document.createElement("span");
        branch.className = "head-branch";
        branch.innerHTML = BRANCH; // a constant, never shell text
        branch.append(document.createTextNode(ctx.branch));
        el.append(branch);
      }
    });
  }

  dispose() {
    this.subs.forEach((s) => s.dispose());
  }
}
