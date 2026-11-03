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

describe("restoring a backup", () => {
  const tick = () => new Promise((r) => setTimeout(r, 0));
  async function pick(encrypted: boolean) {
    const t = show("backup");
    vi.spyOn(ToolsService, "PickRestorePath").mockResolvedValue("/tmp/relay-backup.tar.gz.age" as never);
    vi.spyOn(ToolsService, "NeedsPassphrase").mockResolvedValue(encrypted as never);
    [...t.querySelectorAll("button")].find((b) => b.textContent?.includes("Choose backup"))!.click();
    await tick(); await tick(); flushSync();
    return [...document.querySelectorAll<HTMLElement>('[role="dialog"], [role="alertdialog"]')].find((d) => document.getElementById(d.getAttribute("aria-labelledby") ?? "")?.textContent === "Restore backup") ?? null;
  }
  const button = (d: HTMLElement, text: string) => [...d.querySelectorAll("button")].find((b) => b.textContent?.trim() === text)!;

  it("asks for the backup passphrase in its own dialog as soon as an encrypted backup is chosen", async () => {
    const d = (await pick(true))!;
    expect(d).not.toBeNull();
    const field = d.querySelector<HTMLInputElement>('input[type="password"]')!;
    expect(field.closest("label")?.textContent).toContain("Backup passphrase");
    expect(button(d, "Restore").disabled).toBe(true);
  });

  it("says plainly when the passphrase is wrong, and stays open to try again", async () => {
    const d = (await pick(true))!;
    const restore = vi.spyOn(ToolsService, "Restore").mockRejectedValueOnce(new Error("incorrect passphrase (or the file is corrupted)") as never).mockResolvedValueOnce(undefined as never);
    const field = d.querySelector<HTMLInputElement>('input[type="password"]')!;
    field.value = "wrong"; field.dispatchEvent(new Event("input", { bubbles: true })); flushSync();
    button(d, "Restore").click();
    await tick(); flushSync();
    expect(d.textContent).toContain("That passphrase does not open this backup.");
    expect(field.value).toBe("");
    field.value = "right"; field.dispatchEvent(new Event("input", { bubbles: true })); flushSync();
    button(d, "Restore").click();
    await tick(); flushSync();
    expect(restore).toHaveBeenLastCalledWith("/tmp/relay-backup.tar.gz.age", "right");
  });

  it("asks before restoring an unencrypted backup, with no passphrase field", async () => {
    const d = (await pick(false))!;
    expect(d.querySelector('input[type="password"]')).toBeNull();
    expect(d.textContent).toContain("not encrypted");
  });
});
