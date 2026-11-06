import { describe, expect, it } from "vitest";
import { nextExtent, rowsInUse, type AnchorBuffer } from "./terminal_anchor";

function buf(lines: string[], cursorY: number, over: Partial<AnchorBuffer> = {}): AnchorBuffer {
  return { type: "normal", baseY: 0, cursorY, getLine: (y: number) => (y < lines.length ? { translateToString: () => lines[y] } : undefined), ...over };
}
const ROWS = 24;

describe("rows in use", () => {
  it("is the last row with text, or the cursor's row if that is lower", () => {
    expect(rowsInUse(buf(["$ ls", "a b c", ""], 2), ROWS)).toBe(3);
    expect(rowsInUse(buf(["a", "b", "c", "d"], 0), ROWS)).toBe(4); // the cursor moved up, the text did not go away
    expect(rowsInUse(buf([], 0), ROWS)).toBe(1);
  });
});

// The screen is shifted down by the unused rows so content sits at the bottom. It used to follow the cursor, so a
// program that redraws in place (docker pull, pip, npm: move the cursor up N lines, rewrite, move back) made the
// whole screen jump up and down with every update.
describe("the anchored extent", () => {
  it("grows as output does", () => {
    let e = 0;
    e = nextExtent(e, buf(["$ docker pull x"], 1), ROWS);
    expect(e).toBe(2);
    e = nextExtent(e, buf(["$ docker pull x", "layer 1", "layer 2", "layer 3"], 4), ROWS);
    expect(e).toBe(5);
  });

  it("does not shrink when a progress display moves the cursor back up to redraw", () => {
    const lines = ["$ docker pull x", "a: Downloading 10%", "b: Downloading 20%", "c: Downloading 30%", "d: Downloading 40%"];
    let e = nextExtent(0, buf(lines, 5), ROWS);
    const seen = [e];
    for (const cursor of [1, 2, 3, 4, 5, 1, 5, 2, 5]) { // up, down, up, down, as the progress lines are rewritten
      e = nextExtent(e, buf(lines, cursor), ROWS);
      seen.push(e);
    }
    expect(new Set(seen)).toEqual(new Set([6]));
  });

  it("does not shrink while a redrawn line is momentarily erased", () => {
    const lines = ["$ docker pull x", "a: Downloading", "b: Downloading", ""]; // the last line was just erased
    expect(nextExtent(5, buf(lines, 3), ROWS)).toBe(5);
  });

  it("starts over when the screen is cleared", () => {
    expect(nextExtent(14, buf(["$ "], 0), ROWS)).toBe(1);
  });

  it("is capped at the number of rows", () => {
    expect(nextExtent(30, buf(["x"], 0, { cursorY: 5 }), ROWS)).toBe(ROWS);
  });
});
