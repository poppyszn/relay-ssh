// The remote desktop client syncs the clipboard through the browser's own navigator.clipboard. In the Mac app's
// web view that needs the user to confirm every read with a "Paste" bubble (which pops up on clicks) and often
// refuses writes, so the sync never worked. While an RDP session is open, navigator.clipboard is backed by the
// app's native clipboard instead, which needs no confirmation. Text only (what the client syncs reliably).
import { Clipboard } from "@wailsio/runtime";

const shim = {
  async read(): Promise<ClipboardItem[]> {
    const text = await Clipboard.Text();
    if (!text) return [];
    return [new ClipboardItem({ "text/plain": new Blob([text], { type: "text/plain" }) })];
  },
  async readText(): Promise<string> {
    return (await Clipboard.Text()) ?? "";
  },
  async write(items: ClipboardItem[]): Promise<void> {
    for (const item of items) {
      if (!item.types.includes("text/plain")) continue;
      await Clipboard.SetText(await (await item.getType("text/plain")).text());
      return;
    }
  },
  async writeText(text: string): Promise<void> {
    await Clipboard.SetText(text);
  },
};

let users = 0;

/** Backs navigator.clipboard with the native clipboard. Call the returned function when done; the browser's own comes back with the last user. */
export function useNativeClipboard(): () => void {
  if (users++ === 0) Object.defineProperty(navigator, "clipboard", { value: shim, configurable: true });
  let released = false;
  return () => {
    if (released) return;
    released = true;
    if (--users === 0) delete (navigator as { clipboard?: unknown }).clipboard;
  };
}
