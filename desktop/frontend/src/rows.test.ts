// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { describe, expect, it } from "vitest";
import Branch from "./Branch.svelte";
import ConnRow from "./ConnRow.svelte";
import { sb } from "./foldertree.svelte";
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

// (MouseEvent stands in for PointerEvent, whose button and coordinates happy-dom ignores.)
// A real mouse presses the row's inner button, not the row element, so dragging has to start from there.
describe("dragging from the buttons a mouse actually hits", () => {
  const press = (el: Element) => el.dispatchEvent(new MouseEvent("pointerdown", { bubbles: true, button: 0, clientX: 50, clientY: 20 }));
  const pull = () => window.dispatchEvent(new MouseEvent("pointermove", { clientX: 90, clientY: 60 }));
  const release = () => window.dispatchEvent(new MouseEvent("pointerup", { clientX: 90, clientY: 60 }));

  it("starts from a connection row's name button, not from its edit button", () => {
    const target = document.createElement("div");
    const c = mount(ConnRow, { target, props: { profile: { ...blankProfile(), name: "web", host: "h", user: "u" }, depth: 0, live: false, current: false, onconnect: () => {}, onedit: () => {} } });
    flushSync(); // actions attach in an effect
    press(target.querySelector(".edit")!); pull();
    expect(sb.drag).toBeNull();
    release();
    press(target.querySelector(".main")!); pull();
    expect(sb.drag).toMatchObject({ kind: "profile", id: "web" });
    release();
    sb.cancelDrag();
    unmount(c);
  });

  it("starts from a folder's name button, not from its caret or menu button", () => {
    const target = document.createElement("div");
    const node = { name: "prod", path: "prod", count: 0, children: [], profiles: [] } as never;
    const c = mount(Branch, { target, props: { node, depth: 0, searching: false, open: new Set<string>(), current: "", onconnect: () => {}, onedit: () => {} } });
    flushSync();
    for (const sel of [".caret", ".more"]) { press(target.querySelector(sel)!); pull(); expect(sb.drag).toBeNull(); release(); }
    press(target.querySelector(".name")!); pull();
    expect(sb.drag).toMatchObject({ kind: "folder", id: "prod" });
    release();
    sb.cancelDrag();
    unmount(c);
  });
});
