// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it } from "vitest";
import TabBar from "./TabBar.svelte";

let c: ReturnType<typeof mount> | null = null;
afterEach(() => { if (c) unmount(c); c = null; document.body.innerHTML = ""; });
function show(props: { connected: boolean; remoteDesktop?: boolean }) {
  const target = document.createElement("div");
  document.body.append(target);
  const noop = () => {};
  c = mount(TabBar, { target, props: { custom: false, onsidebar: noop, onsearch: noop, onpalette: noop, onsplit: noop, onsettings: noop, ontools: noop, onfiles: noop, onforwards: noop, onsnippets: noop, ...props } });
  flushSync();
  const btn = (label: string) => target.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!;
  return { files: btn("Files"), forwards: btn("Port forwards") };
}

describe("Files and Port forwards on a remote desktop tab", () => {
  it("are available on an SSH connection", () => {
    const { files, forwards } = show({ connected: true });
    expect(files.disabled).toBe(false);
    expect(forwards.disabled).toBe(false);
  });

  it("are off on an RDP connection, and say why instead of pointing at an SSH connection", () => {
    // They browse and forward over ssh, which a remote desktop connection does not have.
    const { files, forwards } = show({ connected: false, remoteDesktop: true });
    expect(files.disabled).toBe(true);
    expect(forwards.disabled).toBe(true);
    expect(files.title).toMatch(/remote desktop/i);
    expect(files.title).not.toMatch(/saved connection/i);
    expect(forwards.title).toMatch(/remote desktop/i);
  });

  it("still ask for a saved connection when nothing is open", () => {
    const { files } = show({ connected: false });
    expect(files.disabled).toBe(true);
    expect(files.title).toMatch(/saved connection/i);
  });
});
