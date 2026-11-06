// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { RdpService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

const ui = vi.hoisted(() => ({ setEnableClipboard: vi.fn(), setEnableAutoClipboard: vi.fn(), setVisibility: vi.fn(), resize: vi.fn(), ctrlAltDel: vi.fn(), shutdown: vi.fn(), connect: vi.fn() }));
vi.mock("../lib/rdp_client", async (orig) => ({
  ...(await orig<typeof import("../lib/rdp_client")>()),
  loadRdpClient: async () => ({ Backend: {}, displayControl: () => ({}), init: async () => {} }),
  mountClient: async () => ui,
  buildConfig: () => ({}),
}));
vi.mock("../lib/native_clipboard", () => ({ useNativeClipboard: () => () => {} }));

import RdpView from "./RdpView.svelte";

// happy-dom has no layout: give the desktop and the toolbar fixed sizes.
const sizes = (el: Element) => (el.classList.contains("bar") ? { width: 300, height: 30 } : { width: 1000, height: 640 });
beforeEach(() => {
  try { localStorage.clear(); } catch { /* none */ }
  vi.spyOn(Element.prototype, "getBoundingClientRect").mockImplementation(function (this: Element) {
    const { width, height } = sizes(this);
    return { x: 0, y: 0, left: 0, top: 0, right: width, bottom: height, width, height, toJSON() {} } as DOMRect;
  });
});
let c: ReturnType<typeof mount> | null = null;
afterEach(() => { if (c) unmount(c); c = null; document.body.innerHTML = ""; vi.restoreAllMocks(); vi.clearAllMocks(); });

async function connected() {
  vi.spyOn(RdpService, "Open").mockResolvedValue({ sessionId: "s1", wsUrl: "ws://127.0.0.1:1/rdp", token: "t", host: "h", user: "u", domain: "", port: 3389, hasSavedPassword: true });
  vi.spyOn(RdpService, "Credentials").mockResolvedValue("pw");
  vi.spyOn(RdpService, "Close").mockResolvedValue(undefined as never);
  vi.spyOn(RdpService, "Note").mockResolvedValue(undefined as never);
  ui.connect.mockResolvedValue({ run: () => new Promise(() => {}) });
  const target = document.createElement("div");
  document.body.append(target);
  c = mount(RdpView, { target, props: { target: "win", prof: { id: "win", name: "win", protocol: "rdp", host: "h", user: "u", port: 3389, auth: "", key: "", folder: "", tags: [], jump: "", forwardAgent: false, blocks: "", tunnels: [] } } });
  flushSync();
  await new Promise((r) => setTimeout(r, 30));
  flushSync();
  return target;
}
const grip = (t: HTMLElement) => t.querySelector<HTMLElement>("[data-grip]")!;
const bar = (t: HTMLElement) => t.querySelector<HTMLElement>(".bar")!;
const key = (el: Element, k: string) => { el.dispatchEvent(new KeyboardEvent("keydown", { key: k, bubbles: true, cancelable: true })); flushSync(); };

describe("the remote desktop toolbar can be moved", () => {
  it("has a grip, and starts at the top right", async () => {
    const t = await connected();
    expect(grip(t).getAttribute("aria-label")).toMatch(/move/i);
    expect(bar(t).style.transform).toMatch(/translate\(0px, 0px\)|^$/);
  });

  it("moves with the arrow keys, and stays inside the desktop", async () => {
    const t = await connected();
    key(grip(t), "ArrowLeft");
    key(grip(t), "ArrowDown");
    expect(bar(t).style.transform).toBe("translate(-16px, 16px)");
    key(grip(t), "ArrowRight"); key(grip(t), "ArrowRight"); key(grip(t), "ArrowUp"); key(grip(t), "ArrowUp");
    expect(bar(t).style.transform).toBe("translate(0px, 0px)"); // cannot go past the top right corner
    for (let i = 0; i < 100; i++) key(grip(t), "ArrowLeft");
    expect(bar(t).style.transform).toBe(`translate(${-(1000 - 300 - 16)}px, 0px)`);
  });

  it("follows a drag", async () => {
    const t = await connected();
    const g = grip(t);
    const ev = (type: string, x: number, y: number) => g.dispatchEvent(new MouseEvent(type, { clientX: x, clientY: y, bubbles: true, button: 0 }));
    ev("pointerdown", 900, 20); ev("pointermove", 700, 120); ev("pointerup", 700, 120);
    flushSync();
    expect(bar(t).style.transform).toBe("translate(-200px, 100px)");
  });

  it("remembers where it was left, and Home puts it back", async () => {
    let t = await connected();
    key(grip(t), "ArrowLeft"); key(grip(t), "ArrowDown");
    unmount(c!); c = null; document.body.innerHTML = "";
    t = await connected();
    expect(bar(t).style.transform).toBe("translate(-16px, 16px)");
    key(grip(t), "Home");
    expect(bar(t).style.transform).toBe("translate(0px, 0px)");
  });
});
