// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import UpdatesSection from "./UpdatesSection.svelte";
import { UpdateService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
import { updates } from "../state/updates.svelte";

let c: ReturnType<typeof mount> | null = null;
const info = (o: Record<string, unknown> = {}) =>
  ({ current: "1.0.0", latest: "", available: false, notes: "", enabled: true, ...o }) as never;
const tick = async () => { await Promise.resolve(); await Promise.resolve(); flushSync(); };
const button = (t: HTMLElement, text: string) => [...t.querySelectorAll("button")].find((b) => b.textContent?.trim() === text) as HTMLButtonElement | undefined;

async function show() {
  const target = document.createElement("div");
  document.body.append(target);
  c = mount(UpdatesSection, { target });
  flushSync();
  await tick();
  return target;
}

beforeEach(() => {
  updates.phase = "idle";
  updates.latest = updates.notes = updates.error = "";
});
afterEach(() => {
  if (c) unmount(c);
  c = null;
  document.body.innerHTML = "";
  vi.restoreAllMocks();
});

describe("UpdatesSection", () => {
  it("shows the version and says a development build does not update itself", async () => {
    vi.spyOn(UpdateService, "Info").mockResolvedValue(info({ current: "dev", enabled: false }));
    const t = await show();
    expect(t.textContent).toContain("dev");
    expect(t.textContent).toContain("development build");
    expect(button(t, "Check for updates")?.disabled).toBe(true);
  });

  it("checks, then offers the new version with its notes", async () => {
    vi.spyOn(UpdateService, "Info").mockResolvedValue(info());
    vi.spyOn(UpdateService, "Check").mockResolvedValue(info({ available: true, latest: "1.1.0", notes: "Faster tunnels." }));
    const t = await show();
    button(t, "Check for updates")!.click();
    await tick();
    expect(t.textContent).toContain("1.1.0");
    expect(t.textContent).toContain("Faster tunnels.");
    expect(button(t, "Download and install")).toBeTruthy();
  });

  it("says when Relay is up to date", async () => {
    vi.spyOn(UpdateService, "Info").mockResolvedValue(info());
    vi.spyOn(UpdateService, "Check").mockResolvedValue(info());
    const t = await show();
    button(t, "Check for updates")!.click();
    await tick();
    expect(t.textContent).toContain("up to date");
  });

  it("after installing, asks for a restart instead of restarting by itself", async () => {
    vi.spyOn(UpdateService, "Info").mockResolvedValue(info());
    vi.spyOn(UpdateService, "Check").mockResolvedValue(info({ available: true, latest: "1.1.0" }));
    vi.spyOn(UpdateService, "Install").mockResolvedValue(undefined as never);
    const restart = vi.spyOn(UpdateService, "Restart").mockResolvedValue(undefined as never);
    const t = await show();
    button(t, "Check for updates")!.click();
    await tick();
    button(t, "Download and install")!.click();
    await tick();
    expect(restart).not.toHaveBeenCalled();
    button(t, "Restart Relay")!.click();
    await tick();
    expect(restart).toHaveBeenCalledOnce();
  });

  it("shows a failed check as an error", async () => {
    vi.spyOn(UpdateService, "Info").mockResolvedValue(info());
    vi.spyOn(UpdateService, "Check").mockRejectedValue(new Error("offline"));
    const t = await show();
    button(t, "Check for updates")!.click();
    await tick();
    expect(t.querySelector('[role="alert"]')?.textContent).toContain("offline");
  });
});
