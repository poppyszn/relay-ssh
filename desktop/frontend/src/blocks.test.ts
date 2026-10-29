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
  const decos: { line: number; height: number; anchor?: string; cls: string; text: string }[] = [];
  const term: TermLike = {
    cols: 80,
    rows: 24,
    buffer: {
      active: {
        get baseY() { return row.baseY; },
        get cursorX() { return row.cursorX; },
        get cursorY() { return row.cursorY; },
        viewportY: 0,
        get length() { return lines.length; },
        getLine: (y: number) => (lines[y] === undefined ? undefined : { isWrapped: false, translateToString: (trim = false, start = 0) => (trim ? lines[y].slice(start).trimEnd() : lines[y].slice(start)) }) as never,
        type: "normal",
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
    type: (s: string) => onData(s),
    flush: () => new Promise((r) => setTimeout(r, 0)),
  };
}

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
    (f.term.buffer.active as { type: string }).type = "alternate";
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
