import { readFileSync } from "node:fs";
import { describe, expect, it } from "vitest";

// xterm hides a decoration scrolled out of view with inline `display: none`. Block headers and badges force
// `display: flex !important`, so a rule that wins back for the hidden state is what keeps them from sitting at the top.
describe("block overlays that scroll out of view", () => {
  const css = readFileSync(new URL("../theme.css", import.meta.url), "utf8");
  it("stay hidden when xterm hides them", () => {
    for (const cls of ["relay-head", "relay-badge", "relay-block"]) {
      expect(css).toContain(`.xterm .${cls}[style*="display: none"]`);
    }
    expect(css).toMatch(/\[style\*="display: none"\][^{]*\{ display: none !important; \}/);
  });
});
