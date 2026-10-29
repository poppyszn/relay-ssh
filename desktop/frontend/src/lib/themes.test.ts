import { describe, expect, it } from "vitest";
import { THEMES, contrast } from "./themes";

// Every theme must keep its text readable on the surfaces it is drawn on. 4.5 is the WCAG AA
// threshold for body text; the dimmest text (dim) only labels things, so 3 is enough there.
describe.each(Object.entries(THEMES))("%s", (_id, t) => {
  const v = t.vars;
  const surfaces = ["bg", "term", "rail", "bar", "float", "hover", "active", "field"];

  it("body, secondary and muted text are readable on every surface", () => {
    for (const s of surfaces) {
      expect(contrast(v.text, v[s]), `text on ${s}`).toBeGreaterThanOrEqual(7);
      expect(contrast(v.text2, v[s]), `text2 on ${s}`).toBeGreaterThanOrEqual(4.5);
      expect(contrast(v.muted, v[s]), `muted on ${s}`).toBeGreaterThanOrEqual(4.5);
      expect(contrast(v.strong, v[s]), `strong on ${s}`).toBeGreaterThanOrEqual(7);
    }
  });

  it("dim labels, accent text, and status text stay legible", () => {
    for (const s of ["bg", "term", "rail", "bar", "float", "field"]) {
      expect(contrast(v.dim, v[s]), `dim on ${s}`).toBeGreaterThanOrEqual(4.5);
      expect(contrast(v["accent-text"], v[s]), `accent-text on ${s}`).toBeGreaterThanOrEqual(4.5);
      expect(contrast(v["err-text"], v[s]), `err-text on ${s}`).toBeGreaterThanOrEqual(4.5);
      expect(contrast(v.ok, v[s]), `ok on ${s}`).toBeGreaterThanOrEqual(3);
    }
  });

  it("the border of a field stands out from the field (3:1)", () => {
    expect(contrast(v.fieldline, v.field)).toBeGreaterThanOrEqual(3);
  });

  it("text on the accent and error buttons is readable", () => {
    expect(contrast(v["on-accent"], v.accent)).toBeGreaterThanOrEqual(4.5);
    expect(contrast(v["on-err"], v.err)).toBeGreaterThanOrEqual(4.5);
  });

  it("the terminal's own colours are readable on its background", () => {
    const bg = t.term.background!;
    expect(contrast(t.term.foreground!, bg)).toBeGreaterThanOrEqual(7);
    for (const c of ["red", "green", "yellow", "blue", "magenta", "cyan", "brightRed", "brightGreen", "brightBlue", "brightCyan"] as const) {
      expect(contrast(t.term[c] as string, bg), `ANSI ${c}`).toBeGreaterThanOrEqual(3);
    }
    expect(t.term.background).toBe(v.term); // the page and the terminal must agree
  });
});
