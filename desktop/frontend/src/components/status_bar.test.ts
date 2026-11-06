// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import StatusBar from "./StatusBar.svelte";
import { ToolsService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

let c: ReturnType<typeof mount> | null = null;
afterEach(() => { if (c) unmount(c); c = null; document.body.innerHTML = ""; vi.restoreAllMocks(); });

describe("the status bar's build version", () => {
  it("shows which build is running, so a tester can tell which copy of the app they have", async () => {
    vi.spyOn(ToolsService, "Version").mockResolvedValue("ecea3c5-dirty" as never);
    const target = document.createElement("div");
    document.body.append(target);
    c = mount(StatusBar, { target, props: { profiles: [], themeName: "Midnight" } });
    await new Promise((r) => setTimeout(r, 0));
    flushSync();
    const cell = target.querySelector<HTMLElement>("[data-version]");
    expect(cell?.textContent).toBe("ecea3c5-dirty");
    expect(cell?.title).toContain("ecea3c5-dirty");
  });

  it("shows nothing rather than a wrong version when it cannot be read", async () => {
    vi.spyOn(ToolsService, "Version").mockRejectedValue(new Error("no backend") as never);
    const target = document.createElement("div");
    document.body.append(target);
    c = mount(StatusBar, { target, props: { profiles: [], themeName: "Midnight" } });
    await new Promise((r) => setTimeout(r, 0));
    flushSync();
    expect(target.querySelector("[data-version]")).toBeNull();
  });
});
