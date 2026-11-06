// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import { RdpService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

const handlers = vi.hoisted(() => ({ cert: null as null | ((e: { data: unknown }) => void) }));
vi.mock("@wailsio/runtime", async (orig) => ({
  ...(await orig<typeof import("@wailsio/runtime")>()),
  Events: { On: (name: string, cb: (e: { data: unknown }) => void) => { if (name === "rdp:cert") handlers.cert = cb; return () => {}; } },
}));
const copied = vi.hoisted(() => vi.fn(async () => true));
vi.mock("../lib/clipboard", () => ({ copyText: copied }));
vi.mock("../lib/rdp_client", async (orig) => ({ ...(await orig<typeof import("../lib/rdp_client")>()), loadRdpClient: async () => null }));

import RdpView from "./RdpView.svelte";

let c: ReturnType<typeof mount> | null = null;
afterEach(() => { if (c) unmount(c); c = null; document.body.innerHTML = ""; vi.clearAllMocks(); handlers.cert = null; });

async function promptFor(event: Record<string, unknown>) {
  // Open never resolves: the tab is waiting on the gateway, which is where the certificate prompt arrives.
  vi.spyOn(RdpService, "Open").mockResolvedValue({ sessionId: "s1", wsUrl: "ws://127.0.0.1:1/rdp", token: "t", host: "h", user: "u", domain: "", port: 3389, hasSavedPassword: false });
  vi.spyOn(RdpService, "Close").mockResolvedValue(undefined as never);
  vi.spyOn(RdpService, "Note").mockResolvedValue(undefined as never);
  const target = document.createElement("div");
  document.body.append(target);
  c = mount(RdpView, { target, props: { target: "win", prof: { id: "win", name: "win", protocol: "rdp", host: "h", user: "u", port: 3389, auth: "", key: "", folder: "", tags: [], jump: "", forwardAgent: false, blocks: "", tunnels: [] } } });
  flushSync();
  await new Promise((r) => setTimeout(r, 20));
  handlers.cert!({ data: { sessionId: "s1", endpoint: "10.0.0.5:3389", fingerprint: "SHA256:AA:BB:CC", storedFingerprint: "", isMismatch: false, ...event } });
  flushSync();
  return target;
}
const dialog = () => document.querySelector<HTMLElement>('[role="dialog"], [role="alertdialog"]')!;

describe("the certificate prompt", () => {
  it("explains a first connection: what Relay will do, and when to check the fingerprint first", async () => {
    await promptFor({});
    const text = dialog().textContent!;
    expect(text).toContain("10.0.0.5:3389");
    expect(text).toContain("SHA256:AA:BB:CC");
    expect(text).toMatch(/has not connected to this server before/i);
    expect(text).toMatch(/warns you if it ever changes/i);
    expect(text).toMatch(/not a network you trust/i);
  });

  it("lets the fingerprint be copied to compare with the server", async () => {
    await promptFor({});
    const btn = [...dialog().querySelectorAll("button")].find((b) => /copy fingerprint/i.test(b.textContent ?? ""))!;
    expect(btn).toBeTruthy();
    btn.click();
    expect(copied).toHaveBeenCalledWith("SHA256:AA:BB:CC");
  });

  it("treats a changed certificate as a warning, not a routine question", async () => {
    await promptFor({ isMismatch: true, storedFingerprint: "SHA256:11:22:33" });
    const text = dialog().textContent!;
    expect(text).toContain("SHA256:11:22:33");
    expect(text).toContain("SHA256:AA:BB:CC");
    expect(text).toMatch(/unless you know/i);
  });
});
