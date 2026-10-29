// @vitest-environment happy-dom
import { createRawSnippet, flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import Menu, { type MenuItem } from "./Menu.svelte";
import Modal from "./Modal.svelte";

const body = (html: string) => createRawSnippet(() => ({ render: () => `<div>${html}</div>` }));
const press = (el: Element | Window, key: string, init: KeyboardEventInit = {}) => el.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true, ...init }));

let mounted: Array<ReturnType<typeof mount>> = [];
afterEach(() => {
  mounted.forEach((c) => unmount(c));
  mounted = [];
  document.body.innerHTML = "";
});
function show(component: any, props: Record<string, unknown>) {
  const target = document.createElement("div");
  document.body.append(target);
  const c = mount(component, { target, props });
  mounted.push(c);
  flushSync();
  return target;
}

describe("Modal", () => {
  it("focuses the first field, and gives focus back to what opened it", () => {
    const opener = document.createElement("button");
    document.body.append(opener);
    opener.focus();
    const target = document.createElement("div");
    document.body.append(target);
    const c = mount(Modal, { target, props: { title: "Rename", onclose: () => {}, children: body('<input id="f" /><button>Save</button>') } });
    flushSync();
    expect(document.activeElement?.id).toBe("f");
    unmount(c);
    expect(document.activeElement).toBe(opener);
  });

  it("closes on Esc, and only the top modal answers", () => {
    const outer = vi.fn(), inner = vi.fn();
    show(Modal, { title: "Outer", onclose: outer, children: body("a") });
    show(Modal, { title: "Inner", onclose: inner, children: body("b") });
    press(window, "Escape");
    expect(inner).toHaveBeenCalledTimes(1);
    expect(outer).not.toHaveBeenCalled();
  });

  it("is labelled by its title and marked modal", () => {
    show(Modal, { title: "Settings", onclose: () => {}, children: body("x") });
    const dlg = document.querySelector("[role=dialog]")!;
    expect(dlg.getAttribute("aria-modal")).toBe("true");
    expect(document.getElementById(dlg.getAttribute("aria-labelledby")!)?.textContent).toBe("Settings");
  });

  it("keeps Tab inside the panel", () => {
    show(Modal, { title: "T", onclose: () => {}, children: body('<input id="a" /><button id="z">Go</button>') });
    const panel = document.querySelector<HTMLElement>("[role=dialog]")!;
    document.getElementById("z")!.focus();
    press(panel, "Tab");
    expect(document.activeElement).not.toBe(document.getElementById("z"));
    expect(panel.contains(document.activeElement)).toBe(true);
  });

  it("closes on a scrim click unless told not to", () => {
    const close = vi.fn();
    const keep = vi.fn();
    show(Modal, { title: "A", onclose: close, children: body("a") });
    const scrim = document.querySelector<HTMLElement>(".scrim")!;
    scrim.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true }));
    scrim.click();
    expect(close).toHaveBeenCalledTimes(1);

    document.body.innerHTML = "";
    show(Modal, { title: "B", dismissOnScrim: false, onclose: keep, children: body("b") });
    const s2 = document.querySelector<HTMLElement>(".scrim")!;
    s2.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true }));
    s2.click();
    expect(keep).not.toHaveBeenCalled();
  });
});

describe("Menu", () => {
  const items: MenuItem[] = [
    { id: "a", label: "Alpha" },
    { id: "b", label: "Beta", disabled: true },
    { id: "c", label: "Gamma", gapBefore: true },
  ];

  it("moves with the arrows, skips disabled rows and selects with Enter", () => {
    const onselect = vi.fn();
    show(Menu, { items, label: "Test", onselect, onclose: () => {} });
    const menu = document.querySelector<HTMLElement>("[role=menu]")!;
    press(menu, "ArrowDown");
    press(menu, "Enter");
    expect(onselect).toHaveBeenCalledWith("c");
  });

  it("closes on Esc and on a click outside, but not on its own trigger", () => {
    const onclose = vi.fn();
    const trigger = document.createElement("button");
    trigger.dataset.trigger = "";
    document.body.append(trigger);
    show(Menu, { items, label: "Test", onselect: () => {}, onclose, ignore: "[data-trigger]" });
    const menu = document.querySelector<HTMLElement>("[role=menu]")!;
    trigger.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true }));
    expect(onclose).not.toHaveBeenCalled();
    document.body.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true }));
    expect(onclose).toHaveBeenCalledTimes(1);
    press(menu, "Escape");
    expect(onclose).toHaveBeenCalledTimes(2);
  });

  it("does not select a disabled row", () => {
    const onselect = vi.fn();
    show(Menu, { items, label: "Test", onselect, onclose: () => {} });
    document.querySelectorAll<HTMLElement>("[role=menuitem]")[1].click();
    expect(onselect).not.toHaveBeenCalled();
  });
});
