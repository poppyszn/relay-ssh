<script lang="ts">
  import { ws } from "../state/workspace.svelte";
  import { leaves } from "../lib/panes";
  import WindowControls from "./WindowControls.svelte";
  import { Window } from "@wailsio/runtime";
  import { slotAt, paneEdge } from "../lib/lib";
  import { sb } from "../state/foldertree.svelte";
  import Icon from "./Icon.svelte";

  let {
    custom,
    connected,
    remoteDesktop = false,
    onDeck = false,
    activeForwardCount = 0,
    sidebarOpen: _sidebarOpen = false,
    ondeck,
    onselecttab,
    onsidebar: _onsidebar,
    onsearch,
    onpalette,
    onsplit,
    onsettings: _onsettings,
    ontools: _ontools,
    onfiles: _onfiles,
    onforwards,
    onsnippets: _onsnippets,
  }: {
    custom: boolean;
    connected: boolean;
    remoteDesktop?: boolean;
    onDeck?: boolean;
    activeForwardCount?: number;
    sidebarOpen?: boolean;
    ondeck?: () => void;
    onselecttab?: (id: string) => void;
    onsidebar?: () => void;
    onsearch?: () => void;
    onpalette: () => void;
    onsplit: (dir: "row" | "col") => void;
    onsettings?: () => void;
    ontools?: () => void;
    onfiles?: () => void;
    onforwards?: () => void;
    onsnippets?: () => void;
  } = $props();

  const mac = navigator.userAgent.includes("Mac");
  const mod = mac ? "⌘" : "⌃⇧";

  let tabsEl: HTMLElement;
  let drag = $state<{ id: string; slot: number; x: number } | null>(null);
  let justDragged = false;

  function press(e: PointerEvent, id: string) {
    if (e.button !== 0) return;
    const sx = e.clientX;
    let active = false;
    const tabs = () => [...tabsEl.querySelectorAll<HTMLElement>("[data-tab]")].map((t) => t.getBoundingClientRect());
    const move = (ev: PointerEvent) => {
      if (!active && Math.abs(ev.clientX - sx) < 6) return;
      active = true;
      const pane = document.elementFromPoint(ev.clientX, ev.clientY)?.closest<HTMLElement>("[data-pane]");
      const own = ws.tabs.find((t) => t.id === id);
      if (pane && own && !leaves(own.root).includes(pane.dataset.pane ?? "")) {
        sb.paneDrop = { id: pane.dataset.pane!, edge: paneEdge(pane.getBoundingClientRect(), ev.clientX, ev.clientY, false), verb: "Add tab" };
        drag = { id, slot: -1, x: 0 };
        return;
      }
      sb.paneDrop = null;
      const rects = tabs();
      const slot = slotAt(rects.map((r) => r.left + r.width / 2), ev.clientX);
      const edge = slot < rects.length ? rects[slot].left : (rects[rects.length - 1]?.right ?? 0);
      drag = { id, slot, x: edge - tabsEl.getBoundingClientRect().left };
    };
    const end = (commit: boolean) => {
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", up);
      window.removeEventListener("keydown", key);
      const onto = sb.paneDrop;
      sb.paneDrop = null;
      if (active) {
        if (commit && onto && drag?.slot === -1 && onto.edge !== "center") ws.mergeTab(id, onto.id, onto.edge);
        else if (commit && drag && drag.slot >= 0) ws.moveTab(id, drag.slot);
        justDragged = true;
        setTimeout(() => (justDragged = false), 0);
      }
      drag = null;
    };
    const up = () => end(true);
    const key = (ev: KeyboardEvent) => { if (ev.key === "Escape") end(false); };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
    window.addEventListener("keydown", key);
  }

  function tabStatus(t: (typeof ws.tabs)[number]): "err" | "warn" | "rdp" | "ok" {
    const ids = leaves(t.root);
    if (ids.every((id) => ws.panes[id]?.ended)) return "err";
    if (ids.some((id) => ws.panes[id]?.connecting)) return "warn";
    if (ids.some((id) => ws.isRdp(ws.panes[id]?.target ?? null))) return "rdp";
    return "ok";
  }

  function activateTab(id: string) {
    if (justDragged) return;
    if (onselecttab) onselecttab(id);
    else ws.select(id);
  }

  async function openLocal() {
    await ws.openTab(null);
    if (onselecttab && ws.activeId) onselecttab(ws.activeId);
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
  class="strip drag"
  class:mac-inset={!custom && mac}
  ondblclick={(e) => custom && e.target === e.currentTarget && Window.ToggleMaximise()}
>
  {#if custom}
    <div class="win-controls nodrag">
      <WindowControls />
    </div>
  {/if}

  <div class="tabs nodrag" role="tablist" bind:this={tabsEl}>
    <button
      type="button"
      class="deck-pill"
      class:active={onDeck}
      data-deck-tab
      role="tab"
      aria-selected={onDeck}
      title="Return to Relay Deck Spotlight Home"
      onclick={() => ondeck?.()}
    >
      Deck
    </button>

    {#each ws.tabs as t (t.id)}
      {@const paneCount = leaves(t.root).length}
      {@const isCurrent = !onDeck && t.id === ws.activeId}
      {@const st = tabStatus(t)}
      <div
        class="tab"
        data-tab={t.id}
        class:active={isCurrent}
        class:dragging={drag?.id === t.id}
        class:drop={sb.tabDrop === t.id && !leaves(t.root).includes(sb.drag?.id ?? "")}
        role="presentation"
      >
        <button
          type="button"
          class="main"
          role="tab"
          aria-selected={isCurrent}
          onclick={() => activateTab(t.id)}
          onpointerdown={(e) => press(e, t.id)}
        >
          <span class="dot {st}" aria-hidden="true"></span>
          <span class="title">{ws.tabTitle(t)}</span>
          {#if paneCount > 1}
            <span class="pane-count mono">{paneCount}</span>
          {/if}
        </button>
        <button
          type="button"
          class="close-btn"
          aria-label="Close tab"
          onclick={() => ws.closeTab(t.id)}
        >
          <Icon name="x" size={12} />
        </button>
      </div>
    {/each}

    {#if drag && drag.slot >= 0}
      <div class="slot" style="left: {drag.x}px"></div>
    {/if}

    <button
      type="button"
      class="plus-pill"
      data-newtab
      class:drop={sb.tabDrop === "new"}
      aria-label="New local tab"
      title={mac ? "New local shell tab (⌘T)" : "New local shell tab (Ctrl+Shift+T)"}
      onclick={openLocal}
    >
      <Icon name="plus" size={14} />
    </button>
  </div>

  <div class="tools controls nodrag">
    {#if !onDeck && ws.active && !remoteDesktop}
      <div class="session-tools">
        {#if onsearch}
          <button
            type="button"
            class="strip-pill ghost"
            aria-label="Find in scrollback"
            title={mac ? "Find in scrollback (⌘F)" : "Find in scrollback (Ctrl+Shift+F)"}
            onclick={onsearch}
          >
            <Icon name="search" size={14} />
            <span>Find</span>
          </button>
        {/if}
        <button
          type="button"
          class="strip-icon"
          aria-label="Split right"
          title={mac ? "Split pane right (⌘D)" : "Split pane right (Ctrl+Shift+D)"}
          onclick={() => onsplit("row")}
        >
          <Icon name="split-right" size={15} />
        </button>
        <button
          type="button"
          class="strip-icon"
          aria-label="Split down"
          title={mac ? "Split pane down (⇧⌘D)" : "Split pane down (Ctrl+Alt+D)"}
          onclick={() => onsplit("col")}
        >
          <Icon name="split-down" size={15} />
        </button>
      </div>
    {:else if connected}
      <!-- Subtle no-op marker ignored visually while preserving layout rhythm -->
    {/if}

    {#if activeForwardCount > 0}
      <button
        type="button"
        class="telemetry-pill"
        aria-label="Active port forwards"
        title="View active port forwards"
        onclick={() => onforwards?.()}
      >
        <span class="tele-dot" aria-hidden="true"></span>
        <span class="mono">{activeForwardCount} {activeForwardCount === 1 ? "tunnel" : "tunnels"}</span>
      </button>
    {/if}

    <button
      type="button"
      class="cmd-trigger"
      onclick={onpalette}
      aria-label="Quick open"
      title={mac ? "Quick open & commands (⌘K)" : "Quick open & commands (Ctrl+Shift+K)"}
    >
      <span class="cmd-label">Quick Open</span>
      <span class="kbd mono">{mod}K</span>
    </button>
  </div>
</div>

<style>
  .strip {
    height: 44px;
    flex: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 0 14px;
    background: var(--strip);
    border-bottom: 1px solid var(--hairline);
    user-select: none;
  }

  .strip.mac-inset {
    padding-left: 76px;
  }

  .win-controls {
    display: flex;
    align-items: center;
    flex: none;
    padding-right: 4px;
  }

  .tabs {
    position: relative;
    flex: 1 1 0;
    align-self: stretch;
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
    overflow-x: auto;
    scrollbar-width: none;
  }

  .tabs::-webkit-scrollbar {
    display: none;
  }

  .deck-pill {
    height: 28px;
    padding: 0 12px;
    border-radius: 999px;
    font-size: 12px;
    font-weight: 500;
    color: var(--muted);
    background: transparent;
    flex: none;
    transition: background var(--dur-1) var(--ease), color var(--dur-1) var(--ease);
  }

  .deck-pill:hover {
    background: var(--hover);
    color: var(--strong);
  }

  .deck-pill.active {
    background: var(--active);
    color: var(--strong);
  }

  .tab {
    position: relative;
    display: flex;
    align-items: center;
    height: 28px;
    flex: 0 1 208px;
    min-width: 78px;
    max-width: 230px;
    padding-right: 4px;
    border-radius: 999px;
    color: var(--muted);
    border: 1px solid transparent;
    transition: background var(--dur-1) var(--ease), color var(--dur-1) var(--ease), border-color var(--dur-1) var(--ease);
  }

  .tab:not(.active):hover {
    background: var(--hover);
    color: var(--text);
  }

  .tab.active {
    background: var(--active);
    color: var(--strong);
    border-color: var(--hairline);
  }

  .tab.dragging {
    opacity: 0.45;
  }

  .tab.drop,
  .plus-pill.drop {
    background: var(--accent-soft);
    border-color: var(--accent-edge);
  }

  .slot {
    position: absolute;
    top: 8px;
    bottom: 8px;
    width: 2px;
    margin-left: -1px;
    background: var(--accent);
    z-index: 2;
    pointer-events: none;
  }

  .main {
    flex: 1;
    min-width: 0;
    height: 100%;
    display: flex;
    align-items: center;
    gap: 7px;
    padding: 0 2px 0 11px;
    border-radius: inherit;
    background: none;
    border: 0;
    color: inherit;
    font-size: 12px;
    font-weight: 500;
    text-align: left;
  }

  .dot {
    width: 6px;
    height: 6px;
    border-radius: 999px;
    flex: none;
    background: var(--dim);
  }

  .dot.ok { background: var(--ok); }
  .dot.warn { background: var(--warn); }
  .dot.err { background: var(--err); }
  .dot.rdp { background: var(--accent-text); }

  .title {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .pane-count {
    font-size: 11px;
    color: var(--dim);
    flex: none;
  }

  .close-btn {
    width: 20px;
    height: 20px;
    border-radius: 999px;
    display: grid;
    place-items: center;
    color: var(--muted);
    opacity: 0;
    flex: none;
    transition: opacity var(--dur-1) var(--ease), background var(--dur-1) var(--ease), color var(--dur-1) var(--ease);
  }

  .tab:hover .close-btn,
  .tab.active .close-btn,
  .close-btn:focus-visible {
    opacity: 1;
  }

  .close-btn:hover {
    background: var(--hover);
    color: var(--strong);
  }

  .plus-pill {
    width: 28px;
    height: 28px;
    border-radius: 999px;
    display: grid;
    place-items: center;
    color: var(--muted);
    flex: none;
    transition: background var(--dur-1) var(--ease), color var(--dur-1) var(--ease);
  }

  .plus-pill:hover {
    background: var(--hover);
    color: var(--strong);
  }

  .tools {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: none;
  }

  .session-tools {
    display: flex;
    align-items: center;
    gap: 4px;
    padding-right: 8px;
    margin-right: 2px;
    border-right: 1px solid var(--hairline);
  }

  .strip-pill {
    height: 28px;
    padding: 0 9px;
    border-radius: 8px;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    font-size: 12px;
    color: var(--muted);
  }

  .strip-pill:hover {
    background: var(--hover);
    color: var(--strong);
  }

  .strip-icon {
    width: 28px;
    height: 28px;
    border-radius: 8px;
    display: grid;
    place-items: center;
    color: var(--muted);
  }

  .strip-icon:hover {
    background: var(--hover);
    color: var(--strong);
  }

  .telemetry-pill {
    height: 28px;
    padding: 0 10px;
    border-radius: 999px;
    background: var(--hover);
    display: inline-flex;
    align-items: center;
    gap: 7px;
    font-size: 12px;
    color: var(--text2);
  }

  .telemetry-pill:hover {
    background: var(--active);
    color: var(--strong);
  }

  .tele-dot {
    width: 6px;
    height: 6px;
    border-radius: 999px;
    background: var(--ok);
  }

  .cmd-trigger {
    height: 28px;
    padding: 0 11px;
    border-radius: 999px;
    background: var(--field);
    border: 1px solid var(--hairline);
    color: var(--text);
    font-size: 12px;
    font-weight: 500;
    display: inline-flex;
    align-items: center;
    gap: 8px;
    transition: background var(--dur-1) var(--ease), border-color var(--dur-1) var(--ease), color var(--dur-1) var(--ease);
  }

  .cmd-trigger:hover {
    background: var(--active);
    border-color: var(--floatline);
    color: var(--strong);
  }

  .kbd {
    color: var(--muted);
    font-size: 11px;
  }
</style>
