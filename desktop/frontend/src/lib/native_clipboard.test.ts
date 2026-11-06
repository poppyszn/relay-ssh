// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const clip = vi.hoisted(() => ({ Text: vi.fn(), SetText: vi.fn() }));
vi.mock("@wailsio/runtime", () => ({ Clipboard: clip }));

import { useNativeClipboard } from "./native_clipboard";

// Minimal stand-in: happy-dom has no ClipboardItem.
class FakeItem {
  constructor(private parts: Record<string, Blob>) {}
  get types() { return Object.keys(this.parts); }
  getType(t: string) { return Promise.resolve(this.parts[t]); }
}

let restore: (() => void) | null = null;
beforeEach(() => {
  (globalThis as any).ClipboardItem = FakeItem;
  clip.Text.mockReset(); clip.SetText.mockReset();
});
afterEach(() => { restore?.(); restore = null; });

describe("the native clipboard behind navigator.clipboard", () => {
  it("reads the Mac clipboard as text, with no browser permission prompt", async () => {
    clip.Text.mockResolvedValue("copied on the mac");
    restore = useNativeClipboard();
    const items = await navigator.clipboard.read();
    expect(items).toHaveLength(1);
    expect(items[0].types).toEqual(["text/plain"]);
    expect(await (await items[0].getType("text/plain")).text()).toBe("copied on the mac");
    expect(await navigator.clipboard.readText()).toBe("copied on the mac");
  });

  it("reports an empty clipboard as no items", async () => {
    clip.Text.mockResolvedValue("");
    restore = useNativeClipboard();
    expect(await navigator.clipboard.read()).toEqual([]);
  });

  it("writes text that comes back from the remote desktop to the Mac clipboard", async () => {
    restore = useNativeClipboard();
    await navigator.clipboard.write([new FakeItem({ "text/plain": new Blob(["from windows"], { type: "text/plain" }) }) as unknown as ClipboardItem]);
    expect(clip.SetText).toHaveBeenCalledWith("from windows");
    await navigator.clipboard.writeText("plain");
    expect(clip.SetText).toHaveBeenLastCalledWith("plain");
  });

  it("ignores items with no text instead of failing the sync", async () => {
    restore = useNativeClipboard();
    await navigator.clipboard.write([new FakeItem({ "image/png": new Blob(["x"], { type: "image/png" }) }) as unknown as ClipboardItem]);
    expect(clip.SetText).not.toHaveBeenCalled();
  });

  it("puts the browser's own clipboard back when the last user lets go", async () => {
    const before = Object.getOwnPropertyDescriptor(navigator, "clipboard");
    const a = useNativeClipboard();
    const b = useNativeClipboard(); // two RDP tabs
    a();
    expect(Object.getOwnPropertyDescriptor(navigator, "clipboard")?.value).toBeDefined(); // still in use by the second
    b();
    expect(Object.getOwnPropertyDescriptor(navigator, "clipboard")).toEqual(before);
  });
});
