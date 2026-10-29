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

    expect(target.querySelector("h2")?.textContent).toBe("Create Vault");
    expect(target.textContent).toContain("cannot be recovered or reset if forgotten");
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

    expect(target.querySelector("h2")?.textContent).toBe("Unlock Vault");
    expect(target.querySelector('input[aria-label="Master password"]')).not.toBeNull();
    expect(target.querySelector("button[type=submit]")?.textContent).toBe("Unlock");
    expect(target.textContent).toContain("Forgot password? Reset");
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

    expect(target.querySelector("h2")?.textContent).toBe("Unlock Vault for prod-db");
    expect(target.textContent).toContain("prod-db");
    const skipBtn = target.querySelector<HTMLButtonElement>('button[aria-label="Skip unlocking vault"]');
    expect(skipBtn).not.toBeNull();
    skipBtn?.click();
    expect(skipped).toBe(true);
    unmount(c);
  });

  it("renders Manage mode when vault is unlocked", () => {
    vault.exists = true;
    vault.unlocked = true;
    vault.remembered = ["web1", "web2"];
    const target = document.createElement("div");
    const c = mount(VaultPanel, { target, props: { mode: "manage", onclose: () => {} } });
    flushSync();

    expect(target.querySelector("h2")?.textContent).toBe("Vault");
    expect(target.textContent).toContain("Vault is unlocked");
    expect(target.textContent).toContain("2 passwords saved");
    expect(target.textContent).toContain("Lock now");
    expect(target.textContent).toContain("Change master password");
    expect(target.textContent).toContain("Reset vault…");
    unmount(c);
  });

  it("reset requires typing RESET before submit button becomes enabled", async () => {
    const target = document.createElement("div");
    const c = mount(VaultPanel, { target, props: { mode: "reset", onclose: () => {} } });
    flushSync();

    expect(target.querySelector("h2")?.textContent).toBe("Reset Vault");
    const submitBtn = target.querySelector<HTMLButtonElement>("button[type=submit]")!;
    const input = target.querySelector<HTMLInputElement>('input[aria-label="Type RESET to confirm"]')!;

    expect(submitBtn.disabled).toBe(true);

    input.value = "reset";
    input.dispatchEvent(new Event("input", { bubbles: true }));
    flushSync();
    expect(submitBtn.disabled).toBe(true);

    input.value = "RESET";
    input.dispatchEvent(new Event("input", { bubbles: true }));
    flushSync();
    expect(submitBtn.disabled).toBe(false);

    unmount(c);
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
