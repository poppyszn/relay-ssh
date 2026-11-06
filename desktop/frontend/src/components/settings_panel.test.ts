// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it } from "vitest";
import SettingsPanel from "./SettingsPanel.svelte";
import { vault } from "../state/vault.svelte";
import { settings } from "../state/settings.svelte";

let c: ReturnType<typeof mount> | null = null;
afterEach(() => {
  if (c) unmount(c);
  c = null;
  document.body.innerHTML = "";
});
function show(props: Record<string, unknown> = {}) {
  const target = document.createElement("div");
  document.body.append(target);
  c = mount(SettingsPanel, { target, props: { onclose: () => {}, ...props } });
  flushSync();
  return target;
}
const click = (el: Element | null | undefined) => { (el as HTMLElement).click(); flushSync(); };
const button = (t: HTMLElement, text: string) => [...t.querySelectorAll("button")].find((b) => b.textContent?.trim() === text);

describe("SettingsPanel", () => {
  it("is a Settings dialog with Appearance, Terminal, Remote desktop and Vault sections", () => {
    vault.exists = false;
    vault.unlocked = false;
    const t = show();
    expect(t.querySelector("h2")?.textContent).toBe("Settings");
    const labels = [...t.querySelectorAll("h3.label")].map((h) => h.textContent);
    expect(labels).toEqual(["Appearance", "Terminal", "Remote desktop", "Vault"]);
    expect(t.querySelector("#settings-vault .chip")?.textContent).toBe("Not set up");
    expect(button(t, "Create vault")).toBeTruthy();
  });

  it("shows the vault controls that match its state", () => {
    vault.exists = true;
    vault.unlocked = true;
    vault.remembered = ["a", "b"];
    let t = show();
    expect(t.querySelector("#settings-vault .chip")?.textContent).toBe("Unlocked");
    expect(t.textContent).toContain("2 passwords saved");
    for (const label of ["Lock now", "Change master password", "Reset vault…"]) expect(button(t, label)).toBeTruthy();
    unmount(c!);
    c = null;
    document.body.innerHTML = "";

    vault.unlocked = false;
    t = show();
    expect(t.querySelector("#settings-vault .chip")?.textContent).toBe("Locked");
    expect(button(t, "Unlock vault")).toBeTruthy();
    expect(button(t, "Lock now")).toBeFalsy();
  });

  it("reset needs RESET typed before the confirming button enables", () => {
    vault.exists = true;
    vault.unlocked = true;
    const t = show();
    click(button(t, "Reset vault…"));
    const confirmBtn = button(t, "Reset vault") as HTMLButtonElement;
    expect(confirmBtn.className).toContain("danger");
    expect(confirmBtn.disabled).toBe(true);
    const input = t.querySelector<HTMLInputElement>('input[placeholder="RESET"]')!;
    input.value = "reset";
    input.dispatchEvent(new Event("input", { bubbles: true }));
    flushSync();
    expect(confirmBtn.disabled).toBe(true);
    input.value = "RESET";
    input.dispatchEvent(new Event("input", { bubbles: true }));
    flushSync();
    expect(confirmBtn.disabled).toBe(false);
  });

  it("on/off settings are switches", () => {
    const t = show();
    const sw = t.querySelector<HTMLElement>('[role="switch"][aria-labelledby="set-copy"]')!;
    const before = sw.getAttribute("aria-checked");
    click(sw);
    expect(sw.getAttribute("aria-checked")).not.toBe(before);
    click(sw);
  });

  it("lets the clipboard sharing with remote desktops be switched off, and says what it does", () => {
    settings.rdpClipboard = true;
    const t = show();
    const sw = t.querySelector<HTMLElement>('[aria-labelledby="set-rdpclip"]')!;
    expect(sw.getAttribute("aria-checked")).toBe("true");
    expect(t.querySelector("#set-rdpclip")?.textContent).toBe("Share the clipboard with remote desktops");
    expect(t.textContent).toMatch(/replaces your clipboard/i);
    click(sw);
    expect(settings.rdpClipboard).toBe(false);
    expect(sw.getAttribute("aria-checked")).toBe("false");
  });

  it("lets click-to-move-cursor be switched off, and says when it works", () => {
    settings.clickToMove = true;
    const t = show();
    const sw = t.querySelector<HTMLElement>('[aria-labelledby="set-click"]')!;
    expect(sw.getAttribute("aria-checked")).toBe("true");
    expect(t.querySelector("#set-click")?.textContent).toBe("Click to move the cursor");
    expect(t.textContent).toMatch(/command blocks/i);
    click(sw);
    expect(settings.clickToMove).toBe(false);
  });
});
