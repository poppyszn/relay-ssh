// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RdpService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
import { settings } from "../state/settings.svelte";

const ui = vi.hoisted(() => ({
  setEnableClipboard: vi.fn(),
  setEnableAutoClipboard: vi.fn(),
  setVisibility: vi.fn(),
  resize: vi.fn(),
  ctrlAltDel: vi.fn(),
  shutdown: vi.fn(),
  connect: vi.fn(),
}));
vi.mock("../lib/rdp_client", async (orig) => ({
  ...(await orig<typeof import("../lib/rdp_client")>()),
  loadRdpClient: async () => ({ Backend: {}, displayControl: () => ({}), init: async () => {} }),
  mountClient: async () => ui,
  buildConfig: () => ({}),
}));
vi.mock("../lib/native_clipboard", () => ({ useNativeClipboard: () => () => {} }));

import RdpView from "./RdpView.svelte";

let c: ReturnType<typeof mount> | null = null;
afterEach(() => { if (c) unmount(c); c = null; document.body.innerHTML = ""; vi.clearAllMocks(); settings.rdpClipboard = true; });

async function connected() {
  vi.spyOn(RdpService, "Open").mockResolvedValue({ sessionId: "s1", wsUrl: "ws://127.0.0.1:1/rdp", token: "t", host: "h", user: "u", domain: "", port: 3389, hasSavedPassword: true });
  vi.spyOn(RdpService, "Credentials").mockResolvedValue("pw");
  vi.spyOn(RdpService, "Close").mockResolvedValue(undefined as never);
  vi.spyOn(RdpService, "Note").mockResolvedValue(undefined as never);
  ui.connect.mockResolvedValue({ run: () => new Promise(() => {}) }); // a session that stays up
  const target = document.createElement("div");
  document.body.append(target);
  c = mount(RdpView, { target, props: { target: "win", prof: { name: "win", protocol: "rdp", host: "h", user: "u", port: 3389, auth: "", key: "", folder: "", tags: [], jump: "", forwardAgent: false, blocks: "", tunnels: [] } } });
  flushSync();
  await new Promise((r) => setTimeout(r, 30));
  flushSync();
  return target;
}

describe("clipboard sharing in a remote desktop tab", () => {
  it("follows the setting when the session starts, and shows it", async () => {
    settings.rdpClipboard = false;
    const t = await connected();
    expect(ui.setEnableClipboard).toHaveBeenLastCalledWith(false);
    expect(ui.setEnableAutoClipboard).toHaveBeenLastCalledWith(false);
    const btn = t.querySelector<HTMLButtonElement>("[data-clipboard]")!;
    expect(btn.getAttribute("aria-pressed")).toBe("false");
    expect(btn.textContent).toMatch(/off/i);
  });

  it("can be switched for this tab only, without changing the setting", async () => {
    settings.rdpClipboard = true;
    const t = await connected();
    expect(ui.setEnableClipboard).toHaveBeenLastCalledWith(true);
    const btn = t.querySelector<HTMLButtonElement>("[data-clipboard]")!;
    expect(btn.textContent).toMatch(/shared/i);
    btn.click();
    flushSync();
    expect(ui.setEnableClipboard).toHaveBeenLastCalledWith(false);
    expect(ui.setEnableAutoClipboard).toHaveBeenLastCalledWith(false);
    expect(btn.getAttribute("aria-pressed")).toBe("false");
    expect(settings.rdpClipboard).toBe(true);
    btn.click();
    flushSync();
    expect(ui.setEnableClipboard).toHaveBeenLastCalledWith(true);
  });
});
