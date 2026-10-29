<script lang="ts">
  import { ws } from "./workspace.svelte";
  import { leaves } from "./panes";
  import WindowControls from "./WindowControls.svelte";
  import { Window } from "@wailsio/runtime";
  import { connSearch } from "./connsearch.svelte";
  import { slotAt, paneEdge } from "./lib";
  import { sb } from "./foldertree.svelte";

  let { custom, onsidebar, onsearch, onpalette, onsplit, onsettings, ontools, onfiles, onforwards, onsnippets }: { custom: boolean; onsidebar: () => void; onsearch: () => void; onpalette: () => void; onsplit: (dir: "row" | "col") => void; onsettings: () => void; ontools: () => void; onfiles: () => void; onforwards: () => void; onsnippets: () => void } = $props();
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
        <button class="main" role="tab" aria-selected={t.id === ws.activeId} onclick={() => !justDragged && ws.select(t.id)} onpointerdown={(e) => press(e, t.id)}>
          <span class="dot {status(t)}"></span>
          <span class="title">{ws.tabTitle(t)}</span>
          {#if leaves(t.root).length > 1}<span class="chip">{leaves(t.root).length}</span>{/if}
        </button>
        <button class="x" aria-label="Close tab" onclick={() => ws.closeTab(t.id)}>&times;</button>
      </div>
    {/each}
    {#if drag && drag.slot >= 0}<div class="slot" style="left: {drag.x}px"></div>{/if}
    <button class="plus" data-newtab class:drop={sb.tabDrop === "new"} aria-label="New local tab" title="New local tab" onclick={() => ws.openTab(null)}>+</button>
  </div>
  <div class="search nodrag" role="search">
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="M20 20l-3.5-3.5" /></svg>
    <input bind:value={connSearch.query} placeholder="Search connections" aria-label="Search connections" oninput={onsearch}
           onkeydown={(e) => { if (e.key === "Escape") { connSearch.query = ""; e.stopPropagation(); } }} />
    {#if connSearch.query}
      <button class="clear" aria-label="Clear search" onclick={() => (connSearch.query = "")}>&times;</button>
    {:else}
      <button class="kbd" onclick={onpalette} title="Quick open">{mac ? "⌘K" : "Ctrl+K"}</button>
    {/if}
  </div>
  <div class="tools nodrag">
    <button aria-label="Toggle connections" title="Toggle connections" onclick={onsidebar}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="4" width="18" height="16" rx="2" /><path d="M9 4v16" /></svg>
    </button>
    <button aria-label="Split right" title={mac ? "Split pane right (⌘D)" : "Split pane right (Ctrl+Shift+D)"} onclick={() => onsplit("row")}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="4" width="18" height="16" rx="2" /><path d="M12 4v16" /></svg>
    </button>
    <button aria-label="Split down" title={mac ? "Split pane down (⇧⌘D)" : "Split pane down (Ctrl+Alt+D)"} onclick={() => onsplit("col")}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="4" width="18" height="16" rx="2" /><path d="M3 12h18" /></svg>
    </button>
    <button aria-label="Files" title="Browse files on this connection" onclick={onfiles}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" /></svg>
    </button>
    <button aria-label="Port forwards" title="Port forwards on this connection" onclick={onforwards}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 8h13M13 4l4 4-4 4M20 16H7M11 12l-4 4 4 4" /></svg>
    </button>
    <button aria-label="Snippets" title={mac ? "Snippets (⇧⌘S)" : "Snippets (Ctrl+Shift+S)"} onclick={onsnippets}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M8 4H5v16h3M16 4h3v16h-3M9 9l2 2-2 2M13 13h3" /></svg>
    </button>
    <button aria-label="Tools" title="Tools: health, backup, import, keys" onclick={ontools}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M14.7 6.3a4 4 0 0 0-5.4 5.2L3 17.8V21h3.2l6.3-6.3a4 4 0 0 0 5.2-5.4l-2.6 2.6-2.4-.6-.6-2.4z" /></svg>
    </button>
    <button aria-label="Settings" title="Settings" onclick={onsettings}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="3" /><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M4.9 19.1L7 17M17 7l2.1-2.1" /></svg>
    </button>
  </div>
</div>

<style>
  .strip { height: 40px; flex: none; display: flex; background: var(--strip); border-bottom: 1px solid var(--line); }
  .strip.mac { padding-left: 84px; }
  .tabs { position: relative; display: flex; min-width: 0; overflow: hidden; }
  .tab.dragging { opacity: 0.45; }
  .tab.drop, .plus.drop { background: var(--accent-soft); box-shadow: inset 0 0 0 1px var(--accent); }
  .slot { position: absolute; top: 4px; bottom: 4px; width: 2px; margin-left: -1px; background: var(--accent); z-index: 2; pointer-events: none; }
  .tab { display: flex; align-items: center; height: 40px; flex: 0 1 200px; min-width: 56px; max-width: 240px; border-right: 1px solid var(--line); color: var(--muted); }
  .tab.active { background: var(--bg); color: var(--text); box-shadow: inset 0 2px var(--accent); margin-bottom: -1px; height: 41px; }
  .tab:not(.active):hover { background: var(--hover); }
  .main { all: unset; flex: 1; min-width: 0; display: flex; align-items: center; gap: 8px; padding: 0 0 0 16px; height: 100%; cursor: pointer; font-weight: 500; }
  .title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .x { all: unset; width: 28px; height: 100%; display: grid; place-items: center; font-size: 16px; cursor: pointer; opacity: 0.6; }
  .x:hover { opacity: 1; color: var(--text); }
  .plus { all: unset; width: 40px; height: 40px; display: grid; place-items: center; font-size: 18px; color: var(--muted); cursor: pointer; border-right: 1px solid var(--line); }
  .plus:hover { color: var(--text); background: var(--hover); }
  .search { margin: 6px 12px 6px auto; flex: none; width: 220px; display: flex; align-items: center; gap: 8px; padding: 0 6px 0 10px; background: var(--field); border: 1px solid var(--line); border-radius: 4px; color: var(--muted); }
  .search:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .search input { flex: 1; min-width: 0; height: 100%; border: 0; background: none; padding: 0; box-shadow: none; }
  .search input:focus { box-shadow: none; }
  .search .kbd, .search .clear { all: unset; cursor: pointer; font-size: 11px; color: var(--dim); padding: 0 4px; }
  .search .clear { font-size: 15px; }
  .search .clear:hover { color: var(--text); }
  .tools { display: flex; }
  .tools button { all: unset; width: 40px; height: 40px; display: grid; place-items: center; color: var(--muted); border-left: 1px solid var(--line); cursor: pointer; }
  .tools button:hover { color: var(--text); background: var(--hover); }
</style>
