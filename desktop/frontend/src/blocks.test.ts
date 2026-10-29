import { describe, expect, it, vi } from "vitest";
import { BlockTracker, formatDuration, type TermLike } from "./blocks";

// A fake terminal: tracks the cursor row, hands out markers at it, and records decorations.
function fake() {
  let osc: (d: string) => boolean = () => false;
  let onData: (d: string) => void = () => {};
  const row = { baseY: 0, cursorY: 0 };
  const decos: { line: number; height: number; anchor?: string; cls: string; text: string }[] = [];
  const term: TermLike = {
    cols: 80,
    buffer: { active: { get baseY() { return row.baseY; }, get cursorY() { return row.cursorY; }, type: "normal" } },
    parser: { registerOscHandler: (_id, cb) => { osc = cb; return { dispose() {} }; } },
    registerMarker: () => ({ line: row.baseY + row.cursorY, dispose() {}, onDispose() { return { dispose() {} }; } }) as never,
    registerDecoration: (o) => {
      const d = { line: (o.marker as { line: number }).line, height: o.height ?? 1, anchor: o.anchor, cls: "", text: "" };
      decos.push(d);
      // Real xterm exposes onRender as an event on the returned decoration.
      return {
        dispose() {},
        onRender(cb: (el: HTMLElement) => void) {
          queueMicrotask(() => {
            const classes = new Set(["xterm-decoration"]);
            const el = {
              get className() { return [...classes].join(" "); },
              textContent: "",
              classList: { add: (...c: string[]) => c.forEach((x) => classes.add(x)) },
            } as unknown as HTMLElement;
            cb(el);
            cb(el); // redraws call onRender again; classes must not pile up
            d.cls = el.className;
            d.text = el.textContent ?? "";
          });
          return { dispose() {} };
        },
      } as never;
    },
    onData: (cb) => { onData = cb; return { dispose() {} }; },
  };
  return {
    term, decos, row,
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
