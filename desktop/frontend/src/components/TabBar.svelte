<script lang="ts">
  import { ws } from "../state/workspace.svelte";
  import { leaves } from "../lib/panes";
  import WindowControls from "./WindowControls.svelte";
  import { Window } from "@wailsio/runtime";
  import { connSearch } from "../state/connsearch.svelte";
  import { slotAt, paneEdge } from "../lib/lib";
  import { sb } from "../state/foldertree.svelte";
  import Icon from "./Icon.svelte";

  let { custom, connected, remoteDesktop = false, onsidebar, onsearch, onpalette, onsplit, onsettings, ontools, onfiles, onforwards, onsnippets }: { custom: boolean; /** The focused pane is a saved connection, so Files and Forwards have something to act on. */ connected: boolean; /** The focused pane is a remote desktop (RDP) connection: no SSH file browser or port forwards. */ remoteDesktop?: boolean; onsidebar: () => void; onsearch: () => void; onpalette: () => void; onsplit: (dir: "row" | "col") => void; onsettings: () => void; ontools: () => void; onfiles: () => void; onforwards: () => void; onsnippets: () => void } = $props();
  const mac = navigator.userAgent.includes("Mac");

  // Reordering tabs by dragging (pointer events, like the sidebar: native drag and drop is avoided).
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
      // Over a terminal that is not this tab's own: dropping adds the whole tab to that pane's side.
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

  function status(t: (typeof ws.tabs)[number]) {
    const ids = leaves(t.root);
    if (ids.every((id) => ws.panes[id]?.ended)) return "err";
    return ids.some((id) => ws.panes[id]?.connecting) ? "warn" : "ok";
  }
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="strip drag" class:mac ondblclick={(e) => custom && e.target === e.currentTarget && Window.ToggleMaximise()}>
  {#if custom}<WindowControls />{/if}
  <div class="tabs nodrag" role="tablist" bind:this={tabsEl}>
    {#each ws.tabs as t (t.id)}
      <div class="tab" data-tab={t.id} class:active={t.id === ws.activeId} class:dragging={drag?.id === t.id} class:drop={sb.tabDrop === t.id && !leaves(t.root).includes(sb.drag?.id ?? "")} role="presentation">
        <button type="button" class="main" role="tab" aria-selected={t.id === ws.activeId} onclick={() => !justDragged && ws.select(t.id)} onpointerdown={(e) => press(e, t.id)}>
          <span class="dot {status(t)}"></span>
          <span class="title">{ws.tabTitle(t)}</span>
          {#if leaves(t.root).length > 1}<span class="chip">{leaves(t.root).length}</span>{/if}
        </button>
        <button type="button" class="icon-btn dense x" aria-label="Close tab" onclick={() => ws.closeTab(t.id)}><Icon name="x" size={14} /></button>
      </div>
    {/each}
    {#if drag && drag.slot >= 0}<div class="slot" style="left: {drag.x}px"></div>{/if}
    <button type="button" class="icon-btn plus" data-newtab class:drop={sb.tabDrop === "new"} aria-label="New local tab" title="New local tab" onclick={() => ws.openTab(null)}><Icon name="plus" /></button>
  </div>
  <div class="search nodrag" role="search">
    <Icon name="search" size={14} />
    <input bind:value={connSearch.query} placeholder="Search connections" aria-label="Search connections" oninput={onsearch}
           onkeydown={(e) => { if (e.key === "Escape") { connSearch.query = ""; e.stopPropagation(); } }} />
    {#if connSearch.query}
      <button type="button" class="icon-btn dense" aria-label="Clear search" onclick={() => (connSearch.query = "")}><Icon name="x" size={14} /></button>
    {:else}
      <button type="button" class="kbd" onclick={onpalette} title="Quick open">{mac ? "⌘K" : "Ctrl+K"}</button>
    {/if}
  </div>
  <div class="tools nodrag">
    <button type="button" class="icon-btn" aria-label="Toggle connections" title="Toggle connections" onclick={onsidebar}><Icon name="sidebar" /></button>
    <button type="button" class="icon-btn" aria-label="Split right" title={mac ? "Split pane right (⌘D)" : "Split pane right (Ctrl+Shift+D)"} onclick={() => onsplit("row")}><Icon name="split-right" /></button>
    <button type="button" class="icon-btn" aria-label="Split down" title={mac ? "Split pane down (⇧⌘D)" : "Split pane down (Ctrl+Alt+D)"} onclick={() => onsplit("col")}><Icon name="split-down" /></button>
    <span class="rule" aria-hidden="true"></span>
    <button type="button" class="icon-btn" aria-label="Files" title={remoteDesktop ? "Not available on remote desktop connections (no SSH file browser)" : connected ? "Browse files on this connection" : "Open a saved connection to browse its files"} disabled={!connected} onclick={onfiles}><Icon name="files" /></button>
    <button type="button" class="icon-btn" aria-label="Port forwards" title={remoteDesktop ? "Not available on remote desktop connections (port forwards are an SSH feature)" : connected ? "Port forwards on this connection" : "Open a saved connection to manage its port forwards"} disabled={!connected} onclick={onforwards}><Icon name="forwards" /></button>
    <button type="button" class="icon-btn" aria-label="Snippets" title={mac ? "Snippets (⇧⌘S)" : "Snippets (Ctrl+Shift+S)"} onclick={onsnippets}><Icon name="snippets" /></button>
    <span class="rule" aria-hidden="true"></span>
    <button type="button" class="icon-btn" aria-label="Tools" title="Tools: health, backup, import, keys" onclick={ontools}><Icon name="tools" /></button>
    <button type="button" class="icon-btn" aria-label="Settings" title="Settings" onclick={onsettings}><Icon name="settings" /></button>
  </div>
</div>

<style>
  .strip { height: 40px; flex: none; display: flex; align-items: center; background: var(--strip); border-bottom: 1px solid var(--line); }
  .strip.mac { padding-left: 84px; }
  .tabs { position: relative; flex: 1 1 0; align-self: stretch; display: flex; align-items: center; gap: 2px; min-width: 0; overflow: hidden; padding-left: var(--s2); }
  /* Tabs are rounded pills in the strip; the active one is a raised surface with a hairline edge. */
  .tab { position: relative; display: flex; align-items: center; height: var(--h-sm); flex: 0 1 176px; min-width: 72px; max-width: 200px; padding-right: 2px; border-radius: var(--r-sm); color: var(--muted); transition: background var(--dur-1) var(--ease), color var(--dur-1) var(--ease), box-shadow var(--dur-1) var(--ease); }
  .tab:not(.active):hover { background: var(--hover); color: var(--text); }
  .tab.active { background: var(--active); color: var(--strong); box-shadow: inset 0 0 0 1px var(--line2), 0 1px 2px var(--shadow); }
  .tab.dragging { opacity: 0.45; }
  .tab.drop, .plus.drop { background: var(--accent-soft); box-shadow: inset 0 0 0 1px var(--accent); }
  .slot { position: absolute; top: var(--s2); bottom: var(--s2); width: 2px; margin-left: -1px; background: var(--accent); z-index: 2; pointer-events: none; }
  .main { flex: 1; min-width: 0; height: 100%; display: flex; align-items: center; gap: var(--s2); padding: 0 0 0 var(--s3); border-radius: inherit; background: none; border: 0; color: inherit; font-weight: 500; text-align: left; }
  .title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .x { opacity: 0; }
  .tab:hover .x, .tab.active .x, .x:focus-visible { opacity: 1; }
  .plus { margin-left: var(--s1); }
  .search { margin: 0 var(--s3); flex: none; width: 220px; height: var(--h-sm); display: flex; align-items: center; gap: var(--s2); padding: 0 var(--s1) 0 var(--s3); background: var(--field); border: 1px solid var(--line); border-radius: var(--r-sm); color: var(--muted); }
  .search:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .search input { flex: 1; min-width: 0; height: 100%; border: 0; background: none; padding: 0; box-shadow: none; font-size: var(--t-sm); }
  .search input:focus { box-shadow: none; }
  .search .kbd { background: none; cursor: pointer; margin-right: var(--s1); }
  .tools { display: flex; align-items: center; gap: 2px; padding-right: var(--s3); }
  .rule { width: 1px; height: 16px; margin: 0 var(--s2); background: var(--line2); }
</style>
