// @vitest-environment happy-dom
import { mount, unmount } from "svelte";
import { describe, expect, it } from "vitest";
import ConnRow from "./ConnRow.svelte";
import { blankProfile } from "./lib";

// The sidebar's "Open now" list renders rows at depth -1 (flush left, no indent guides).
// Array(-1) throws, which broke the whole sidebar the moment a connection was opened.
describe("ConnRow", () => {
  const props = (depth: number) => ({ profile: { ...blankProfile(), name: "web", host: "h", user: "u" }, depth, live: true, current: false, onconnect: () => {}, onedit: () => {} });

  it.each([-1, 0, 2])("renders at depth %i with one guide per level", (depth) => {
    const target = document.createElement("div");
    const c = mount(ConnRow, { target, props: props(depth) });
    expect(target.querySelectorAll(".guide")).toHaveLength(Math.max(0, depth));
    expect(target.textContent).toContain("web");
    unmount(c);
  });
});
