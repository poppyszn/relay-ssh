// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from "vitest";
import { clampSidebarWidth } from "./ui.svelte";

const KEY = "relay.ui.v1";

afterEach(() => {
  localStorage.clear();
  vi.resetModules();
});

describe("ui persistent workspace state", () => {
  it("clamps sidebar widths between 220px and 400px", () => {
    expect(clampSidebarWidth(120)).toBe(220);
    expect(clampSidebarWidth(268)).toBe(268);
    expect(clampSidebarWidth(520)).toBe(400);
    expect(clampSidebarWidth(285.6)).toBe(286);
  });

  it("recovers defaults cleanly from corrupt localStorage", async () => {
    localStorage.setItem(KEY, "{not-json");
    vi.resetModules();
    const { ui } = await import("./ui.svelte");
    expect(ui.sidebarOpen).toBe(true);
    expect(ui.sidebarWidth).toBe(268);
    expect(ui.selectedFolder).toBe("");
    expect(ui.recentIds).toEqual([]);
    expect(ui.sessionTabs).toEqual([]);
  });

  it("persists sidebar width, open state, selected folder, recents, and session tabs", async () => {
    vi.resetModules();
    const first = await import("./ui.svelte");
    first.ui.setSidebarOpen(false);
    first.ui.setSidebarWidth(340);
    first.ui.setSelectedFolder("prod/eu");
    first.ui.recordConnection("host-a");
    first.ui.recordConnection("host-b");
    first.ui.recordConnection("host-a");
    first.ui.saveSession([{ target: null }, { target: "host-a" }], 1);

    vi.resetModules();
    const second = await import("./ui.svelte");
    expect(second.ui.sidebarOpen).toBe(false);
    expect(second.ui.sidebarWidth).toBe(340);
    expect(second.ui.selectedFolder).toBe("prod/eu");
    expect(second.ui.recentIds).toEqual(["host-a", "host-b"]);
    expect(second.ui.sessionTabs).toEqual([{ target: null }, { target: "host-a" }]);
    expect(second.ui.sessionActive).toBe(1);
  });

  it("caps recent connections at 8 unique ids", async () => {
    vi.resetModules();
    const { ui } = await import("./ui.svelte");
    for (let i = 1; i <= 12; i++) ui.recordConnection(`id-${i}`);
    expect(ui.recentIds).toEqual(["id-12", "id-11", "id-10", "id-9", "id-8", "id-7", "id-6", "id-5"]);
  });
});
