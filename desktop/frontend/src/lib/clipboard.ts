import { Clipboard } from "@wailsio/runtime";

/** Copies text. Prefers the desktop runtime (always allowed), falling back to the web API. */
export async function copyText(text: string): Promise<boolean> {
  try {
    await Clipboard.SetText(text);
    return true;
  } catch {
    try {
      await navigator.clipboard.writeText(text);
      return true;
    } catch {
      return false;
    }
  }
}
