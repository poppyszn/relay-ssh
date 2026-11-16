import { describe, expect, it } from "vitest";

// vitest empties stylesheet imports, so read theme.css directly (matching block_scroll_css.test.ts).
// @ts-ignore
const { readFileSync } = (await import("node:fs")) as { readFileSync: (p: URL, enc: string) => string };
const css = readFileSync(new URL("../theme.css", import.meta.url), "utf8");

describe("theme.css preflight reset", () => {
  it("does not import tailwindcss", () => {
    expect(css).not.toMatch(/tailwindcss/);
  });

  it("includes base reset rules for box model, headings, lists, media, forms, and hidden", () => {
    expect(css).toContain("@layer base");
    expect(css).toContain("border: 0 solid;");
    expect(css).toMatch(/h1,\s*h2,\s*h3,\s*h4,\s*h5,\s*h6/);
    expect(css).toMatch(/ol,\s*ul,\s*menu/);
    expect(css).toContain("background-color: transparent;");
    expect(css).toContain("[hidden]:where(:not([hidden='until-found']))");
    expect(css).toContain(".shrink-0 { flex-shrink: 0; }");
  });
});
