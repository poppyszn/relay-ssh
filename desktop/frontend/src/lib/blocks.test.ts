// @vitest-environment happy-dom
import { describe, expect, it, vi } from "vitest";
import { BlockTracker, formatDuration, type TermLike } from "./blocks";

// A fake terminal: tracks the cursor row, hands out markers at it, serves buffer rows, and
// records decorations. Decoration elements are real DOM elements (happy-dom), as in xterm.
function fake() {
  let osc: (d: string) => boolean = () => false;
  let onData: (d: string) => void = () => {};
  const row = { baseY: 0, cursorY: 0, cursorX: 0 };
  const lines: string[] = [];
  const elements: HTMLElement[] = [];
  let bufType: "normal" | "alternate" = "normal";
  let onBuffer: () => void = () => {};
  const element = document.createElement("div");
  const decos: { line: number; height: number; anchor?: string; cls: string; text: string }[] = [];
  const term: TermLike = {
    cols: 80,
    rows: 24,
    element,
    buffer: {
      onBufferChange: (cb: () => void) => { onBuffer = cb; return { dispose() {} }; },
      active: {
        get baseY() { return row.baseY; },
        get cursorX() { return row.cursorX; },
        get cursorY() { return row.cursorY; },
        viewportY: 0,
        get length() { return lines.length; },
        getLine: (y: number) => (lines[y] === undefined ? undefined : { isWrapped: false, translateToString: (trim = false, start = 0) => (trim ? lines[y].slice(start).trimEnd() : lines[y].slice(start)) }) as never,
        get type() { return bufType; },
      },
    },
    parser: { registerOscHandler: (_id, cb) => { osc = cb; return { dispose() {} }; } },
    registerMarker: () => ({ line: row.baseY + row.cursorY, isDisposed: false, dispose() {}, onDispose() { return { dispose() {} }; } }) as never,
    registerDecoration: (o) => {
      const d = { line: (o.marker as { line: number }).line, height: o.height ?? 1, anchor: o.anchor, cls: "", text: "" };
      decos.push(d);
      // Real xterm exposes onRender as an event on the returned decoration.
      return {
        dispose() {},
        onRender(cb: (el: HTMLElement) => void) {
          queueMicrotask(() => {
            const el = document.createElement("div");
            el.className = "xterm-decoration"; // xterm's own class, which must survive
            elements.push(el);
            cb(el);
            cb(el); // redraws call onRender again; classes and children must not pile up
            d.cls = el.className;
            d.text = el.querySelector(".badge-text")?.textContent ?? el.textContent ?? "";
          });
          return { dispose() {} };
        },
      } as never;
    },
    onData: (cb) => { onData = cb; return { dispose() {} }; },
  };
  return {
    term, decos, row, lines, elements,
    osc: (d: string) => osc(d),
    setAlt: (on: boolean) => { bufType = on ? "alternate" : "normal"; onBuffer(); },
    type: (s: string) => onData(s),
    flush: () => new Promise((r) => setTimeout(r, 0)),
  };
}

describe("BlockTracker input start", () => {
  // Where typed input begins, but only while a prompt is waiting for it: that is when clicking may move the cursor.
  it("is known between the prompt and Enter, and nowhere else", () => {
    const f = fake();
    const t = new BlockTracker(f.term);
    expect(t.inputStart()).toBeUndefined(); // before any prompt
    f.row.cursorY = 4; f.row.cursorX = 0; f.osc("A");
    expect(t.inputStart()).toBeUndefined(); // the prompt is still being drawn
    f.row.cursorX = 2; f.osc("B");
    expect(t.inputStart()).toEqual({ line: 4, col: 2 });
    f.type("ls\r"); // Enter: the command is running, keys would go to it
    expect(t.inputStart()).toBeUndefined();
    t.dispose();
  });
});

describe("BlockTracker on the alternate screen", () => {
  // Blocks are overlays tied to rows of the normal screen. A full-screen program (vim, less, htop) switches to the
  // alternate screen, and the overlays must not be drawn over it.
  it("hides block decorations while a full-screen program runs, and shows them again after", () => {
    const f = fake();
    const t = new BlockTracker(f.term);
    const el = f.term.element!;
    expect(el.classList.contains("relay-alt")).toBe(false);
    f.setAlt(true);
    expect(el.classList.contains("relay-alt")).toBe(true);
    f.setAlt(false);
    expect(el.classList.contains("relay-alt")).toBe(false);
    t.dispose();
  });

  it("is hidden from the start when created while on the alternate screen", () => {
    const f = fake();
    f.setAlt(true);
    const t = new BlockTracker(f.term);
    expect(f.term.element!.classList.contains("relay-alt")).toBe(true);
    t.dispose();
  });

  it("stops watching and clears the mark when disposed", () => {
    const f = fake();
    const t = new BlockTracker(f.term);
    f.setAlt(true);
    t.dispose();
    expect(f.term.element!.classList.contains("relay-alt")).toBe(false);
  });
});

describe("BlockTracker", () => {
  it("draws a block from prompt to the next prompt, with exit status and badge", async () => {
    const f = fake();
    const seen = vi.fn();
    const t = new BlockTracker(f.term, seen);
    f.row.cursorY = 0; f.osc("A"); f.osc("B");      // prompt on row 0
    f.type("ls\r");                                   // user presses Enter
    f.row.cursorY = 3;                                // 2 lines of output, cursor now on row 3
    f.osc("D;0"); f.osc("A");
    await f.flush();
    expect(seen).toHaveBeenCalledTimes(1);
    const block = f.decos.find((d) => d.cls.includes("relay-block"))!;
    expect(block).toMatchObject({ line: 0, height: 3 });
    expect(block.cls).toContain("ok");
    // xterm's own class carries the absolute positioning; losing it hides the overlay
    expect(block.cls).toContain("xterm-decoration");
    expect(block.cls.split(" ").filter((c) => c === "relay-block")).toHaveLength(1);
    const badge = f.decos.find((d) => d.cls.includes("relay-badge"))!;
    expect(badge.anchor).toBe("right");
    expect(badge.text).not.toContain("exit");
    t.dispose();
  });

  it("draws a header on the block's first row from the prompt marker's branch and directory", async () => {
    const f = fake();
    const t = new BlockTracker(f.term);
    f.row.cursorY = 0; f.osc("A;main;~/code/relay"); f.osc("B");
    f.type("ls\r");
    f.row.cursorY = 3; f.osc("D;0"); f.osc("A;;~");
    await f.flush();
    const head = f.decos.find((d) => d.cls.includes("relay-head"))!;
    expect(head).toMatchObject({ line: 0, height: 1 });
    expect(head.cls).toContain("xterm-decoration");
    expect(head.text).toContain("~/code/relay");
    expect(head.text).toContain("main");
    expect(f.decos.filter((d) => d.cls.includes("relay-head"))).toHaveLength(1);
    // a directory that is only "~" and no branch: no branch part
    f.row.cursorY = 3; f.osc("B"); f.type("pwd\r");
    f.row.cursorY = 6; f.osc("D;0"); f.osc("A");
    await f.flush();
    const second = f.decos.filter((d) => d.cls.includes("relay-head"))[1];
    expect(second.text).toBe("~");
    t.dispose();
  });

  it("keeps a semicolon in the directory, draws no header without context, and renders it as text", async () => {
    const f = fake();
    const t = new BlockTracker(f.term);
    f.row.cursorY = 0; f.osc("A;<img src=x>;~/a;b"); f.osc("B"); f.type("x\r");
    f.row.cursorY = 2; f.osc("D;0"); f.osc("A"); // an old snippet: no branch or directory
    f.osc("B"); f.type("y\r");
    f.row.cursorY = 4; f.osc("D;0"); f.osc("A");
    await f.flush();
    const heads = f.decos.filter((d) => d.cls.includes("relay-head"));
    expect(heads).toHaveLength(1);
    expect(heads[0].text).toContain("~/a;b");
    expect(heads[0].text).toContain("<img src=x>");
    expect(f.elements.some((e) => e.querySelector("img"))).toBe(false);
    t.dispose();
  });

  it("the badge has a menu button that hands the block's command and output to the menu, once", async () => {
    const f = fake();
    f.lines.push("", "❯ ls", "a b", "", "");
    const onMenu = vi.fn();
    new BlockTracker(f.term, undefined, onMenu);
    f.row.cursorY = 0; f.osc("A");
    f.row.cursorY = 1; f.row.cursorX = 2; f.osc("B"); f.type("ls\r"); // the prompt "❯ " is 2 columns wide
    f.row.cursorY = 4; f.osc("D;0"); f.osc("A");
    await f.flush();
    const badge = f.elements.find((e) => e.classList.contains("relay-badge"))!;
    expect(badge.querySelectorAll("button")).toHaveLength(1); // not duplicated by the second render
    badge.querySelector("button")!.click();
    const ref = onMenu.mock.calls[0][0];
    expect(ref.command()).toBe("ls");
    expect(ref.output()).toBe("a b");
    expect(ref.all()).toBe("ls\na b");
  });

  it("marks failures and shows the exit code", async () => {
    const f = fake();
    new BlockTracker(f.term);
    f.osc("A"); f.osc("B"); f.type("false\r");
    f.row.cursorY = 2;
    f.osc("D;5"); f.osc("A");
    await f.flush();
    expect(f.decos.find((d) => d.cls.includes("relay-block"))!.cls).toContain("err");
    expect(f.decos.find((d) => d.cls.includes("relay-badge"))!.text).toContain("exit 5");
  });

  it("handles a command typed ahead before its prompt appeared", async () => {
    const f = fake();
    new BlockTracker(f.term);
    f.osc("A"); f.osc("B"); f.type("sleep 1\r");
    f.type("false\r");                 // typed while the first command still runs
    f.row.cursorY = 1; f.osc("D;0"); f.osc("A"); f.osc("B");
    f.row.cursorY = 3; f.osc("D;1"); f.osc("A");
    await f.flush();
    const blocks = f.decos.filter((d) => d.cls.includes("relay-block"));
    expect(blocks.map((b) => b.cls.includes("err"))).toEqual([false, true]);
  });

  it("ignores prompts where nothing was run, and a D with no prompt before it", async () => {
    const f = fake();
    new BlockTracker(f.term);
    f.osc("D;0");                       // stray first D
    f.osc("A"); f.osc("B");             // prompt, user just presses nothing
    f.row.cursorY = 1;
    f.osc("D;0"); f.osc("A");
    await f.flush();
    expect(f.decos).toHaveLength(0);
  });

  it("does not draw blocks for the alternate screen", async () => {
    const f = fake();
    new BlockTracker(f.term);
    f.osc("A"); f.osc("B"); f.type("vim\r");
    f.setAlt(true);
    f.row.cursorY = 2;
    f.osc("D;0");
    await f.flush();
    expect(f.decos).toHaveLength(0);
  });
});

describe("formatDuration", () => {
  it("is compact at every scale", () => {
    expect(formatDuration(40)).toBe("40ms");
    expect(formatDuration(310)).toBe("0.31s");
    expect(formatDuration(12_340)).toBe("12.3s");
    expect(formatDuration(95_000)).toBe("1m 35s");
  });
});
