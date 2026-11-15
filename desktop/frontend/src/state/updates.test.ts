import { beforeEach, describe, expect, it, vi } from "vitest";
import { UpdateService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
import { Updates } from "./updates.svelte";

const info = (o: Record<string, unknown> = {}) =>
  ({ current: "1.0.0", latest: "", available: false, notes: "", enabled: true, ...o }) as never;

beforeEach(() => vi.restoreAllMocks());

describe("Updates", () => {
  it("loads the running version and whether this build can update", async () => {
    vi.spyOn(UpdateService, "Info").mockResolvedValue(info({ enabled: false, current: "dev" }));
    const u = new Updates();
    await u.load();
    expect(u.current).toBe("dev");
    expect(u.enabled).toBe(false);
  });

  it("reports up to date", async () => {
    vi.spyOn(UpdateService, "Check").mockResolvedValue(info());
    const u = new Updates();
    await u.check();
    expect(u.phase).toBe("uptodate");
  });

  it("reports an available release with its notes", async () => {
    vi.spyOn(UpdateService, "Check").mockResolvedValue(info({ available: true, latest: "1.1.0", notes: "Faster." }));
    const u = new Updates();
    await u.check();
    expect(u.phase).toBe("available");
    expect(u.latest).toBe("1.1.0");
    expect(u.notes).toBe("Faster.");
  });

  it("shows a failed check as an error, not as up to date", async () => {
    vi.spyOn(UpdateService, "Check").mockRejectedValue(new Error("offline"));
    const u = new Updates();
    await u.check();
    expect(u.phase).toBe("error");
    expect(u.error).toBe("offline");
  });

  it("installs then waits for the user to restart", async () => {
    vi.spyOn(UpdateService, "Check").mockResolvedValue(info({ available: true, latest: "1.1.0" }));
    const install = vi.spyOn(UpdateService, "Install").mockResolvedValue(undefined as never);
    const restart = vi.spyOn(UpdateService, "Restart").mockResolvedValue(undefined as never);
    const u = new Updates();
    await u.check();
    await u.install();
    expect(install).toHaveBeenCalledOnce();
    expect(u.phase).toBe("ready");
    expect(restart).not.toHaveBeenCalled();
    await u.restart();
    expect(restart).toHaveBeenCalledOnce();
  });

  it("a failed install goes back to an error and can be retried", async () => {
    vi.spyOn(UpdateService, "Check").mockResolvedValue(info({ available: true, latest: "1.1.0" }));
    vi.spyOn(UpdateService, "Install").mockRejectedValue(new Error("signature did not verify"));
    const u = new Updates();
    await u.check();
    await u.install();
    expect(u.phase).toBe("error");
    expect(u.error).toContain("signature");
  });

  it("ignores a second check while one is running", async () => {
    let release!: (v: never) => void;
    const check = vi.spyOn(UpdateService, "Check").mockReturnValue(new Promise((r) => (release = r)) as never);
    const u = new Updates();
    const first = u.check();
    await u.check();
    release(info());
    await first;
    expect(check).toHaveBeenCalledOnce();
  });

  it("the launch check says whether there is something to announce, and never throws", async () => {
    vi.spyOn(UpdateService, "Info").mockResolvedValue(info({ enabled: false }));
    const check = vi.spyOn(UpdateService, "Check");
    const dev = new Updates();
    expect(await dev.checkOnLaunch()).toBe(false);
    expect(check).not.toHaveBeenCalled(); // a development build makes no request

    vi.spyOn(UpdateService, "Info").mockResolvedValue(info());
    check.mockResolvedValue(info({ available: true, latest: "1.1.0" }));
    expect(await new Updates().checkOnLaunch()).toBe(true);

    check.mockRejectedValue(new Error("offline"));
    expect(await new Updates().checkOnLaunch()).toBe(false);
  });
});
