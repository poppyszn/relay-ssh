// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import TabBar from "./TabBar.svelte";
import DockRail from "./DockRail.svelte";
import { ws } from "../state/workspace.svelte";

let c: ReturnType<typeof mount> | null = null;
afterEach(() => {
  if (c) unmount(c);
  c = null;
  ws.tabs = [];
  ws.activeId = "";
  document.body.innerHTML = "";
});

function showTabBar(props: Record<string, unknown> = {}) {
  const target = document.createElement("div");
  document.body.append(target);
  const noop = () => {};
  c = mount(TabBar, {
    target,
    props: {
      custom: false,
      connected: false,
      remoteDesktop: false,
      onDeck: false,
      activeForwardCount: 0,
      ondeck: noop,
      onselecttab: noop,
      onsidebar: noop,
      onsearch: noop,
      onpalette: noop,
      onsplit: noop,
      onsettings: noop,
      ontools: noop,
      onfiles: noop,
      onforwards: noop,
      onsnippets: noop,
      ...props,
    },
  });
  flushSync();
  return {
    target,
    deck: target.querySelector<HTMLButtonElement>("[data-deck-tab]")!,
    quickOpen: target.querySelector<HTMLButtonElement>('button[aria-label="Quick open"]')!,
    splitRight: () => target.querySelector<HTMLButtonElement>('button[aria-label="Split right"]'),
    splitDown: () => target.querySelector<HTMLButtonElement>('button[aria-label="Split down"]'),
    find: () => target.querySelector<HTMLButtonElement>('button[aria-label="Find in scrollback"]'),
    telemetry: () => target.querySelector<HTMLButtonElement>('button[aria-label="Active port forwards"]'),
  };
}

function showDock(props: Record<string, unknown> = {}) {
  const target = document.createElement("div");
  document.body.append(target);
  const noop = () => {};
  c = mount(DockRail, {
    target,
    props: {
      drawer: "none" as const,
      onDeck: true,
      activeForwardCount: 0,
      vaultExists: false,
      vaultUnlocked: false,
      ondeck: noop,
      ontoggledrawer: noop,
      onvault: noop,
      onsettings: noop,
      ...props,
    },
  });
  flushSync();
  return {
    target,
    btn: (label: string) => target.querySelector<HTMLButtonElement>(`button[aria-label="${label}"]`)!,
    vault: target.querySelector<HTMLButtonElement>("button.vault-btn")!,
  };
}

describe("Command Deck TabBar", () => {
  it("invokes onpalette from Quick open and reflects Deck tab selection", () => {
    const onpalette = vi.fn();
    const ondeck = vi.fn();
    const { deck, quickOpen } = showTabBar({ onDeck: true, onpalette, ondeck });
    expect(deck.getAttribute("aria-selected")).toBe("true");
    quickOpen.click();
    expect(onpalette).toHaveBeenCalledTimes(1);
    deck.click();
    expect(ondeck).toHaveBeenCalledTimes(1);
  });

  it("shows Split and Find session controls only on non-RDP live tabs", () => {
    ws.tabs = [{ id: "t1", root: { id: "p1" }, focus: "p1" }];
    ws.activeId = "t1";

    const ssh = showTabBar({ connected: true, remoteDesktop: false, onDeck: false });
    expect(ssh.splitRight()).not.toBeNull();
    expect(ssh.splitDown()).not.toBeNull();
    expect(ssh.find()).not.toBeNull();
    unmount(c!);
    c = null;
    document.body.innerHTML = "";

    const rdp = showTabBar({ connected: false, remoteDesktop: true, onDeck: false });
    expect(rdp.splitRight()).toBeNull();
    expect(rdp.splitDown()).toBeNull();
    expect(rdp.find()).toBeNull();
  });

  it("shows live tunnel telemetry pill when activeForwardCount > 0", () => {
    const onforwards = vi.fn();
    const { telemetry } = showTabBar({ activeForwardCount: 2, onforwards });
    expect(telemetry()?.textContent).toContain("2 tunnels");
    telemetry()?.click();
    expect(onforwards).toHaveBeenCalledTimes(1);
  });

  it("renders close, minimise and zoom window buttons when custom chrome is active", () => {
    const { target } = showTabBar({ custom: true });
    expect(target.querySelector('button[aria-label="Close window"]')).not.toBeNull();
    expect(target.querySelector('button[aria-label="Minimise window"]')).not.toBeNull();
    expect(target.querySelector('button[aria-label="Zoom window"]')).not.toBeNull();
  });
});

describe("Command Deck DockRail", () => {
  it("highlights active drawer and dispatches drawer toggles", () => {
    const ontoggledrawer = vi.fn();
    const { btn, target } = showDock({ drawer: "forwards", activeForwardCount: 1, ontoggledrawer });
    expect(btn("Port Forwards").getAttribute("aria-pressed")).toBe("true");
    expect(btn("Saved Hosts").getAttribute("aria-pressed")).toBe("false");
    expect(target.querySelector(".pulse-dot")).not.toBeNull();

    btn("Saved Hosts").click();
    expect(ontoggledrawer).toHaveBeenCalledWith("hosts");
    btn("Remote Files (SFTP)").click();
    expect(ontoggledrawer).toHaveBeenCalledWith("files");
  });

  it("triggers vault management from the vault status control", () => {
    const onvault = vi.fn();
    const { vault } = showDock({ vaultExists: true, vaultUnlocked: true, onvault });
    expect(vault.getAttribute("aria-label")).toMatch(/unlocked/i);
    vault.click();
    expect(onvault).toHaveBeenCalledTimes(1);
  });
});
