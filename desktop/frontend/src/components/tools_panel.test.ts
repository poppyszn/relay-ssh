// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import ToolsPanel from "./ToolsPanel.svelte";
import { ToolsService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

let c: ReturnType<typeof mount> | null = null;
afterEach(() => {
  if (c) unmount(c);
  c = null;
  document.body.innerHTML = "";
  vi.restoreAllMocks();
});
function show(initial = "health") {
  vi.spyOn(ToolsService, "Health").mockResolvedValue([] as never);
  vi.spyOn(ToolsService, "Keys").mockResolvedValue([] as never);
  const target = document.createElement("div");
  document.body.append(target);
  c = mount(ToolsPanel, { target, props: { initial: initial as never, onclose: () => {}, onchanged: () => {}, ongenerate: () => {} } });
  flushSync();
  return target;
}
const tabs = (t: HTMLElement) => [...t.querySelectorAll<HTMLElement>('[role="tab"]')];

describe("ToolsPanel", () => {
  it("has Health, Backup & restore, Import and Keys tabs, and no Vault tab", () => {
    const t = show();
    expect(tabs(t).map((b) => b.textContent)).toEqual(["Health", "Backup & restore", "Import", "Keys"]);
    expect(t.querySelector('[role="tablist"]')).not.toBeNull();
  });

  it("marks the selected tab and moves with the arrow keys", () => {
    const t = show();
    expect(tabs(t)[0].getAttribute("aria-selected")).toBe("true");
    tabs(t)[0].dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
    flushSync();
    expect(tabs(t)[1].getAttribute("aria-selected")).toBe("true");
    expect(tabs(t)[0].getAttribute("aria-selected")).toBe("false");
    tabs(t)[1].dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowUp", bubbles: true, cancelable: true }));
    tabs(t)[0].dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowUp", bubbles: true, cancelable: true }));
    flushSync();
    expect(tabs(t)[3].getAttribute("aria-selected")).toBe("true");
  });

  it("renders no error notice while there is no message", () => {
    const t = show("backup");
    expect(t.querySelector(".notice")).toBeNull();
  });
});
