// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it } from "vitest";
import Icon, { type IconName } from "./Icon.svelte";

// @ts-ignore
const { readFileSync } = (await import("node:fs")) as { readFileSync: (p: string, enc: string) => string };
// @ts-ignore
const { resolve } = (await import("node:path")) as { resolve: (...p: string[]) => string };
const src = readFileSync(resolve("src/components/Icon.svelte"), "utf8");

const ALL_ICONS: IconName[] = [
  "check",
  "x",
  "plus",
  "search",
  "chevron-right",
  "chevron-down",
  "chevron-up",
  "arrow-right",
  "arrow-up",
  "refresh",
  "more",
  "grip",
  "folder",
  "folder-open",
  "folder-plus",
  "terminal",
  "pencil",
  "trash",
  "copy",
  "lines",
  "copy-all",
  "import",
  "key",
  "lock",
  "unlock",
  "sidebar",
  "split-right",
  "split-down",
  "forwards",
  "snippets",
  "tools",
  "settings",
  "files",
  "warning",
  "info",
  "vault",
  "rdp",
  "server",
  "zap",
  "command",
];

let mounted: Array<ReturnType<typeof mount>> = [];
afterEach(() => {
  mounted.forEach((c) => unmount(c));
  mounted = [];
  document.body.innerHTML = "";
});

describe("Icon", () => {
  it("does not depend on @lucide/svelte", () => {
    expect(src).not.toContain("@lucide/svelte");
  });

  it("renders all 40 icons with accessible SVG attributes and geometry", () => {
    for (const name of ALL_ICONS) {
      const target = document.createElement("div");
      document.body.append(target);
      mounted.push(mount(Icon, { target, props: { name, size: 14 } }));
      flushSync();
      const svg = target.querySelector("svg");
      expect(svg, `missing svg for ${name}`).not.toBeNull();
      expect(svg?.getAttribute("width")).toBe("14");
      expect(svg?.getAttribute("height")).toBe("14");
      expect(svg?.getAttribute("viewBox")).toBe("0 0 24 24");
      expect(svg?.getAttribute("aria-hidden")).toBe("true");
      expect(svg?.classList.contains("shrink-0")).toBe(true);
      expect(svg?.querySelectorAll("path, rect, circle, line").length ?? 0).toBeGreaterThan(0);
    }
  });
});
