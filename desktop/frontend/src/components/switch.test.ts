// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it } from "vitest";
import Switch from "./Switch.svelte";
import { settings } from "../state/settings.svelte";

let c: ReturnType<typeof mount> | null = null;
afterEach(() => {
  if (c) unmount(c);
  c = null;
  document.body.innerHTML = "";
});

function show(initial = false, disabled = false) {
  const target = document.createElement("div");
  document.body.append(target);
  settings.copyOnSelect = initial;
  c = mount(Switch, {
    target,
    props: {
      class: "switch",
      disabled,
      "aria-labelledby": "label-id",
      get checked() { return settings.copyOnSelect; },
      set checked(v: boolean) { settings.copyOnSelect = v; },
    },
  });
  flushSync();
  return {
    btn: target.querySelector<HTMLButtonElement>('button[role="switch"]')!,
  };
}

describe("Switch", () => {
  it("renders a switch button with aria-checked, aria-labelledby, and switch class", () => {
    const { btn } = show(true);
    expect(btn).not.toBeNull();
    expect(btn.type).toBe("button");
    expect(btn.getAttribute("aria-checked")).toBe("true");
    expect(btn.getAttribute("aria-labelledby")).toBe("label-id");
    expect(btn.classList.contains("switch")).toBe(true);
  });

  it("toggles by click and keeps bind:checked and aria-checked in sync", () => {
    const { btn } = show(false);
    expect(btn.getAttribute("aria-checked")).toBe("false");

    btn.click();
    flushSync();
    expect(settings.copyOnSelect).toBe(true);
    expect(btn.getAttribute("aria-checked")).toBe("true");

    btn.click();
    flushSync();
    expect(settings.copyOnSelect).toBe(false);
    expect(btn.getAttribute("aria-checked")).toBe("false");
  });

  it("toggles with Space and Enter keys", () => {
    const { btn } = show(false);

    const space = new KeyboardEvent("keydown", { key: " ", bubbles: true, cancelable: true });
    btn.dispatchEvent(space);
    flushSync();
    expect(space.defaultPrevented).toBe(true);
    expect(settings.copyOnSelect).toBe(true);
    expect(btn.getAttribute("aria-checked")).toBe("true");

    const enter = new KeyboardEvent("keydown", { key: "Enter", bubbles: true, cancelable: true });
    btn.dispatchEvent(enter);
    flushSync();
    expect(enter.defaultPrevented).toBe(true);
    expect(settings.copyOnSelect).toBe(false);
    expect(btn.getAttribute("aria-checked")).toBe("false");
  });

  it("does not toggle when disabled", () => {
    const { btn } = show(false, true);
    btn.click();
    btn.dispatchEvent(new KeyboardEvent("keydown", { key: " ", bubbles: true, cancelable: true }));
    flushSync();
    expect(settings.copyOnSelect).toBe(false);
    expect(btn.getAttribute("aria-checked")).toBe("false");
  });
});
