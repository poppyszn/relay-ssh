// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { describe, expect, it, vi } from "vitest";
import VaultPanel from "./VaultPanel.svelte";
import { vault } from "../state/vault.svelte";
import { VaultService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

describe("VaultPanel component", () => {
  it("renders Create mode when mode='create' or vault does not exist", () => {
    vault.exists = false;
    vault.unlocked = false;
    const target = document.createElement("div");
    const c = mount(VaultPanel, { target, props: { mode: "create", onclose: () => {} } });
    flushSync();

    expect(target.querySelector("h2")?.textContent).toBe("Create vault");
    expect(target.textContent).toContain("the only way back in is to reset the vault");
    expect(target.textContent).not.toContain("cannot be recovered");
    expect(target.querySelector('input[aria-label="Master password"]')).not.toBeNull();
    expect(target.querySelector('input[aria-label="Confirm master password"]')).not.toBeNull();
    unmount(c);
  });

  it("renders Unlock mode when vault exists and is locked", () => {
    vault.exists = true;
    vault.unlocked = false;
    const target = document.createElement("div");
    const c = mount(VaultPanel, { target, props: { mode: "unlock", onclose: () => {} } });
    flushSync();

    expect(target.querySelector("h2")?.textContent).toBe("Unlock vault");
    expect(target.querySelector('input[aria-label="Master password"]')).not.toBeNull();
    expect(target.querySelector("button[type=submit]")?.textContent).toBe("Unlock");
    unmount(c);
  });

  it("renders unlock-connect mode with target connection name and Skip button", () => {
    vault.exists = true;
    vault.unlocked = false;
    let skipped = false;
    const target = document.createElement("div");
    const c = mount(VaultPanel, {
      target,
      props: {
        mode: "unlock-connect",
        connName: "prod-db",
        onclose: () => {},
        onskip: () => { skipped = true; },
      },
    });
    flushSync();

    expect(target.querySelector("h2")?.textContent).toBe("Unlock vault for prod-db");
    expect(target.textContent).toContain("prod-db");
    const skipBtn = target.querySelector<HTMLButtonElement>('button[aria-label="Skip unlocking vault"]');
    expect(skipBtn).not.toBeNull();
    skipBtn?.click();
    expect(skipped).toBe(true);
    unmount(c);
  });

  it("auto mode creates when there is no vault and unlocks when there is one", () => {
    vault.exists = false;
    vault.unlocked = false;
    let target = document.createElement("div");
    let c = mount(VaultPanel, { target, props: { mode: "auto", onclose: () => {} } });
    flushSync();
    expect(target.querySelector("h2")?.textContent).toBe("Create vault");
    unmount(c);

    vault.exists = true;
    target = document.createElement("div");
    c = mount(VaultPanel, { target, props: { mode: "auto", onclose: () => {} } });
    flushSync();
    expect(target.querySelector("h2")?.textContent).toBe("Unlock vault");
    unmount(c);
  });

  it("create refuses mismatched passwords without calling the service", async () => {
    vault.exists = false;
    const create = vi.spyOn(VaultService, "Create").mockResolvedValue(undefined as never);
    const target = document.createElement("div");
    const c = mount(VaultPanel, { target, props: { mode: "create", onclose: () => {} } });
    flushSync();
    const type = (label: string, value: string) => {
      const el = target.querySelector<HTMLInputElement>(`input[aria-label="${label}"]`)!;
      el.value = value;
      el.dispatchEvent(new Event("input", { bubbles: true }));
    };
    type("Master password", "one");
    type("Confirm master password", "two");
    flushSync();
    target.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true }));
    flushSync();
    expect(target.textContent).toContain("Passwords do not match");
    expect(create).not.toHaveBeenCalled();
    unmount(c);
    vi.restoreAllMocks();
  });

  it("clears input on unlock attempt", async () => {
    vault.exists = true;
    vault.unlocked = false;
    vi.spyOn(VaultService, "Unlock").mockRejectedValue(new Error("wrong password"));

    const target = document.createElement("div");
    const c = mount(VaultPanel, { target, props: { mode: "unlock", onclose: () => {} } });
    flushSync();

    const input = target.querySelector<HTMLInputElement>('input[aria-label="Master password"]')!;
    input.value = "secret123";
    input.dispatchEvent(new Event("input", { bubbles: true }));
    flushSync();

    const form = target.querySelector("form")!;
    form.dispatchEvent(new Event("submit", { bubbles: true }));
    flushSync();
    await new Promise((r) => setTimeout(r, 50));
    flushSync();

    // Password input must be cleared immediately, never left in DOM
    expect(input.value).toBe("");
    expect(target.textContent).toContain("Wrong master password or corrupt vault.");

    unmount(c);
    vi.restoreAllMocks();
  });
});
