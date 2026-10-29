// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { describe, expect, it, vi } from "vitest";
import ProfileForm from "./ProfileForm.svelte";
import { blankProfile } from "./lib";
import { vault } from "./vault.svelte";
import { ProfilesService, ToolsService, VaultService } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

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
});
