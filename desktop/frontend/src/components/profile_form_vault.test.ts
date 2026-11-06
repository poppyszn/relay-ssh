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
        oldId: "",
        profiles: [],
        folders: [],
        onsave: async () => "id",
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
        oldId: "srv-saved",
        profiles: [],
        folders: [],
        onsave: async () => "id",
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
        oldId: "srv-forget",
        profiles: [],
        folders: [],
        onsave: async () => "id",
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
  async function submitted(opts: { oldId: string; name: string; unlocked: boolean; exists?: boolean; typed?: string; remembered?: string[]; newHost?: string }) {
    vault.remembered = opts.remembered ?? [];
    vi.spyOn(vault, "refresh").mockResolvedValue({ exists: opts.exists ?? true, unlocked: opts.unlocked });
    const onsave = vi.fn(async () => "id");
    const target = document.createElement("div");
    document.body.append(target);
    const prof = { ...blankProfile(), name: opts.name, host: "h", user: "u", auth: "password" as const };
    const c = mount(ProfileForm, { target, props: { initial: prof, oldId: opts.oldId, profiles: [], folders: [], onsave, ondelete: () => {}, oncancel: () => {} } });
    flushSync();
    if (opts.newHost) {
      const host = [...target.querySelectorAll<HTMLInputElement>("input.mono")].find((i) => i.value === "h")!;
      host.value = opts.newHost;
      host.dispatchEvent(new Event("input", { bubbles: true }));
      flushSync();
    }
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
    const r = await submitted({ oldId: "", name: "srv", unlocked: true, typed: "s3cret-pass" });
    expect(r.onsave).toHaveBeenCalledTimes(1);
    expect((r.onsave.mock.calls[0] as unknown[])[1]).toBe("s3cret-pass");
    r.done();
  });

  it("asks to unlock instead of saving when a typed password has nowhere to go", async () => {
    const r = await submitted({ oldId: "", name: "srv", unlocked: false, typed: "s3cret-pass" });
    expect(r.onsave).not.toHaveBeenCalled();
    expect(document.body.textContent).toContain("Unlock vault");
    r.done();
  });

  it("asks to unlock before changing the host of a connection whose password is saved", async () => {
    const r = await submitted({ oldId: "old", name: "new", unlocked: false, remembered: ["old"], newHost: "other-host" });
    expect(r.onsave).not.toHaveBeenCalled();
    expect(document.body.textContent).toContain("Unlock vault");
    r.done();
  });

  it("does not ask to unlock for a rename alone: the password follows the connection's id", async () => {
    const r = await submitted({ oldId: "old", name: "new", unlocked: false, remembered: ["old"] });
    expect(r.onsave).toHaveBeenCalledTimes(1);
    r.done();
  });

  it("saves without a password when nothing needs the vault", async () => {
    const r = await submitted({ oldId: "srv", name: "srv", unlocked: false, remembered: ["srv"] });
    expect(r.onsave).toHaveBeenCalledTimes(1);
    expect((r.onsave.mock.calls[0] as unknown[])[1]).toBe("");
    r.done();
  });

  it("shows a saved password only after the master password is supplied", async () => {
    vault.remembered = ["srv-saved"];
    const reveal = vi.spyOn(VaultService, "RevealSecret").mockImplementation((_c, master) =>
      (master === "right-master-pass" ? Promise.resolve("s3cret-pass") : Promise.reject(new Error("incorrect master password"))) as never);
    const target = document.createElement("div");
    document.body.append(target);
    const prof = { ...blankProfile(), name: "srv-saved", host: "h", user: "u", auth: "password" as const };
    const c = mount(ProfileForm, { target, props: { initial: prof, oldId: "srv-saved", profiles: [], folders: [], onsave: async () => "id", ondelete: () => {}, oncancel: () => {} } });
    flushSync();
    const show = Array.from(target.querySelectorAll("button")).find((b) => b.textContent === "Show")!;
    show.click();
    flushSync();
    const ask = (master: string) => {
      const input = document.body.querySelector<HTMLInputElement>('#reveal-form input[type="password"]')!;
      input.value = master;
      input.dispatchEvent(new Event("input", { bubbles: true }));
      flushSync();
      document.body.querySelector<HTMLFormElement>("#reveal-form")!.dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
    };
    expect(document.body.textContent).toContain("Show saved password");
    expect(document.body.querySelector('[aria-label="Saved password"]')).toBeNull();

    ask("wrong-master-pass");
    await new Promise((r) => setTimeout(r, 0));
    flushSync();
    expect(document.body.textContent).toContain("That master password is not right.");
    expect(document.body.querySelector('[aria-label="Saved password"]')).toBeNull();

    ask("right-master-pass");
    await new Promise((r) => setTimeout(r, 0));
    flushSync();
    expect(document.body.querySelector<HTMLInputElement>('[aria-label="Saved password"]')?.value).toBe("s3cret-pass");
    expect(reveal).toHaveBeenCalledWith("srv-saved", "right-master-pass");
    unmount(c);
    target.remove();
  });

  it("supports RDP protocol with domain and password vault", async () => {
    vault.remembered = [];
    const prof = { ...blankProfile(), name: "win1", protocol: "rdp", port: 3389, domain: "CORP" };
    const target = document.createElement("div");
    const c = mount(ProfileForm, {
      target,
      props: {
        initial: prof,
        oldId: "",
        profiles: [],
        folders: [],
        onsave: async () => "id",
        ondelete: () => {},
        oncancel: () => {},
      },
    });
    flushSync();

    expect(target.querySelector('input[placeholder="e.g. CORP or WORKGROUP"]')).not.toBeNull();
    expect(target.querySelector('input[type="password"][aria-label="Password"]')).not.toBeNull();
    // Key and Agent options are hidden for RDP
    expect(target.textContent).not.toContain("Key file");
    expect(target.textContent).not.toContain("Forward SSH agent");
    expect(target.textContent).not.toContain("Port forwarding");
    unmount(c);
  });
});
