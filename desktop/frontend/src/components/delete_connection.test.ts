// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import DeleteConnection from "./DeleteConnection.svelte";
import { ProfilesService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
import { blankProfile } from "../lib/lib";

const mk = (o: object) => ({ ...blankProfile(), host: "h", user: "u", ...o }) as ReturnType<typeof blankProfile>;
const bastion = mk({ name: "bastion" });
const profiles = [bastion, mk({ name: "dev-app-server", jump: "bastion" }), mk({ name: "win-build", protocol: "rdp", jump: "bastion" }), mk({ name: "homelab" })];

let c: ReturnType<typeof mount> | null = null;
afterEach(() => { if (c) unmount(c); c = null; document.body.innerHTML = ""; vi.restoreAllMocks(); });
function show(profile: ReturnType<typeof mk>, ondeleted = () => {}) {
  const target = document.createElement("div");
  document.body.append(target);
  c = mount(DeleteConnection, { target, props: { profile, profiles, ondeleted, onclose: () => {} } });
  flushSync();
  return document.querySelector<HTMLElement>('[role="alertdialog"]')!;
}
const button = (d: HTMLElement, text: string) => [...d.querySelectorAll("button")].find((b) => b.textContent?.trim() === text);

describe("deleting a connection", () => {
  it("says up front which connections use it as their jump host, and does not offer Delete", () => {
    const d = show(bastion);
    expect(d.textContent).toContain("dev-app-server");
    expect(d.textContent).toContain("win-build");
    expect(d.textContent).toContain("jump host");
    expect(button(d, "Delete")).toBeUndefined();
  });

  it("deletes a connection nothing depends on", async () => {
    const del = vi.spyOn(ProfilesService, "Delete").mockResolvedValue(undefined as never);
    const done = vi.fn();
    const d = show(profiles[3], done);
    button(d, "Delete")!.click();
    await new Promise((r) => setTimeout(r, 0));
    expect(del).toHaveBeenCalledWith("homelab");
    expect(done).toHaveBeenCalled();
  });

  it("shows a refusal from the app inside the dialog instead of losing it", async () => {
    vi.spyOn(ProfilesService, "Delete").mockRejectedValue(new Error("writing connections.toml: permission denied") as never);
    const d = show(profiles[3]);
    button(d, "Delete")!.click();
    await new Promise((r) => setTimeout(r, 0));
    flushSync();
    expect(d.querySelector('[role="alert"]')?.textContent).toContain("permission denied");
  });
});
