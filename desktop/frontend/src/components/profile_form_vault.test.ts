// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { describe, expect, it, vi } from "vitest";
import ProfileForm from "./ProfileForm.svelte";
import { blankProfile } from "../lib/lib";
import { vault } from "../state/vault.svelte";
import { ProfilesService, ToolsService, VaultService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

describe("ProfileForm vault password integration", () => {
  vi.spyOn(ProfilesService, "Preview").mockResolvedValue({ config: "", problem: "" });
  vi.spyOn(ToolsService, "Keys").mockResolvedValue([]);
  vi.spyOn(VaultService, "HasSecret").mockResolvedValue(false);

  it("shows optional password input when auth='password' and no saved password", () => {
    vault.remembered = [];
    const prof = { ...blankProfile(), name: "srv1", auth: "password" as const };
    const target = document.createElement("div");
    const c = mount(ProfileForm, {
      target,
      props: {
        initial: prof,
        oldName: "",
        profiles: [],
        folders: [],
        onsave: async () => {},
        ondelete: () => {},
        oncancel: () => {},
      },
    });
    flushSync();

    expect(target.querySelector('input[type="password"][aria-label="Password"]')).not.toBeNull();
    expect(target.textContent).toContain("Stored encrypted in Relay's vault, unlocked with your master password.");
    expect(target.textContent).not.toContain("It is never stored.");
    unmount(c);
  });

  it("shows 'A password is saved' badge with Change and Forget buttons when saved", () => {
    vault.remembered = ["srv-saved"];
    const prof = { ...blankProfile(), name: "srv-saved", auth: "password" as const };
    const target = document.createElement("div");
    const c = mount(ProfileForm, {
      target,
      props: {
        initial: prof,
        oldName: "srv-saved",
        profiles: [],
        folders: [],
        onsave: async () => {},
        ondelete: () => {},
        oncancel: () => {},
      },
    });
    flushSync();

    expect(target.textContent).toContain("A password is saved");
    expect(target.querySelector('input[type="password"]')).toBeNull();

    const changeBtn = Array.from(target.querySelectorAll("button")).find((b) => b.textContent === "Change");
    expect(changeBtn).toBeDefined();
    changeBtn?.click();
    flushSync();

    // Now password field should appear with Cancel button
    expect(target.querySelector('input[type="password"][aria-label="Password"]')).not.toBeNull();
    unmount(c);
  });

  it("clicking Forget removes the remembered state", async () => {
    vault.remembered = ["srv-forget"];
    const forgetSpy = vi.spyOn(VaultService, "ForgetSecret").mockResolvedValue();
    const prof = { ...blankProfile(), name: "srv-forget", auth: "password" as const };
    const target = document.createElement("div");
    const c = mount(ProfileForm, {
      target,
      props: {
        initial: prof,
        oldName: "srv-forget",
        profiles: [],
        folders: [],
        onsave: async () => {},
        ondelete: () => {},
        oncancel: () => {},
      },
    });
    flushSync();

    const forgetBtn = Array.from(target.querySelectorAll("button")).find((b) => b.textContent === "Forget");
    expect(forgetBtn).toBeDefined();
    forgetBtn?.click();
    flushSync();
    await new Promise((r) => setTimeout(r, 20));
    flushSync();

    expect(vault.hasRemembered("srv-forget")).toBe(false);
    expect(target.textContent).not.toContain("A password is saved");
    expect(target.querySelector('input[type="password"][aria-label="Password"]')).not.toBeNull();
    unmount(c);
    forgetSpy.mockRestore();
  });

  // The form hands the typed password to onsave so the backend saves and binds it in one step.
  async function submitted(opts: { oldName: string; name: string; unlocked: boolean; exists?: boolean; typed?: string; remembered?: string[] }) {
    vault.remembered = opts.remembered ?? [];
    vi.spyOn(vault, "refresh").mockResolvedValue({ exists: opts.exists ?? true, unlocked: opts.unlocked });
    const onsave = vi.fn(async () => {});
    const target = document.createElement("div");
    document.body.append(target);
    const prof = { ...blankProfile(), name: opts.name, host: "h", user: "u", auth: "password" as const };
    const c = mount(ProfileForm, { target, props: { initial: prof, oldName: opts.oldName, profiles: [], folders: [], onsave, ondelete: () => {}, oncancel: () => {} } });
    flushSync();
    if (opts.typed) {
      const input = target.querySelector<HTMLInputElement>('input[type="password"][aria-label="Password"]')!;
      input.value = opts.typed;
      input.dispatchEvent(new Event("input", { bubbles: true }));
      flushSync();
    }
    target.querySelector("form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    await new Promise((r) => setTimeout(r, 0));
    flushSync();
    return { onsave, target, done: () => { unmount(c); target.remove(); } };
  }

  it("passes a typed password to onsave when the vault is open", async () => {
    const r = await submitted({ oldName: "", name: "srv", unlocked: true, typed: "s3cret-pass" });
    expect(r.onsave).toHaveBeenCalledTimes(1);
    expect((r.onsave.mock.calls[0] as unknown[])[1]).toBe("s3cret-pass");
    r.done();
  });

  it("asks to unlock instead of saving when a typed password has nowhere to go", async () => {
    const r = await submitted({ oldName: "", name: "srv", unlocked: false, typed: "s3cret-pass" });
    expect(r.onsave).not.toHaveBeenCalled();
    expect(document.body.textContent).toContain("Unlock Vault");
    r.done();
  });

  it("asks to unlock before renaming a connection whose password is saved", async () => {
    const r = await submitted({ oldName: "old", name: "new", unlocked: false, remembered: ["old"] });
    expect(r.onsave).not.toHaveBeenCalled();
    expect(document.body.textContent).toContain("Unlock Vault");
    r.done();
  });

  it("saves without a password when nothing needs the vault", async () => {
    const r = await submitted({ oldName: "srv", name: "srv", unlocked: false, remembered: ["srv"] });
    expect(r.onsave).toHaveBeenCalledTimes(1);
    expect((r.onsave.mock.calls[0] as unknown[])[1]).toBe("");
    r.done();
  });
});
