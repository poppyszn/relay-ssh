<script lang="ts">
  import { ws } from "./workspace.svelte";
  import { leaves } from "./panes";

  let { onsidebar, onsettings }: { onsidebar: () => void; onsettings: () => void } = $props();
  const mac = navigator.userAgent.includes("Mac");

  function status(t: (typeof ws.tabs)[number]) {
    const ids = leaves(t.root);
    return ids.every((id) => ws.panes[id]?.ended) ? "err" : "ok";
  }
</script>

<div class="strip drag" class:mac>
  <div class="tabs nodrag" role="tablist">
    {#each ws.tabs as t (t.id)}
      <div class="tab" class:active={t.id === ws.activeId} role="presentation">
        <button class="main" role="tab" aria-selected={t.id === ws.activeId} onclick={() => ws.select(t.id)}>
          <span class="dot {status(t)}"></span>
          <span class="title">{ws.tabTitle(t)}</span>
          {#if leaves(t.root).length > 1}<span class="chip">{leaves(t.root).length}</span>{/if}
        </button>
        <button class="x" aria-label="Close tab" onclick={() => ws.closeTab(t.id)}>&times;</button>
      </div>
    {/each}
    <button class="plus" aria-label="New local tab" title="New local tab" onclick={() => ws.openTab(null)}>+</button>
  </div>
  <div class="tools nodrag">
    <button aria-label="Toggle connections" title="Toggle connections" onclick={onsidebar}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="4" width="18" height="16" rx="2" /><path d="M9 4v16" /></svg>
    </button>
    <button aria-label="Settings" title="Settings" onclick={onsettings}>
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="12" cy="12" r="3" /><path d="M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M4.9 19.1L7 17M17 7l2.1-2.1" /></svg>
    </button>
  </div>
</div>

<style>
  .strip { height: 40px; flex: none; display: flex; background: var(--strip); border-bottom: 1px solid var(--line); }
  .strip.mac { padding-left: 84px; }
  .tabs { display: flex; min-width: 0; overflow: hidden; }
  .tab { display: flex; align-items: center; height: 40px; min-width: 140px; max-width: 240px; border-right: 1px solid var(--line); color: var(--muted); }
  .tab.active { background: var(--bg); color: var(--text); box-shadow: inset 0 2px var(--accent); margin-bottom: -1px; height: 41px; }
  .tab:not(.active):hover { background: #141d2a; }
  .main { all: unset; flex: 1; min-width: 0; display: flex; align-items: center; gap: 8px; padding: 0 0 0 16px; height: 100%; cursor: pointer; font-weight: 500; }
  .title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .x { all: unset; width: 28px; height: 100%; display: grid; place-items: center; font-size: 16px; cursor: pointer; opacity: 0.6; }
  .x:hover { opacity: 1; color: var(--text); }
  .plus { all: unset; width: 40px; height: 40px; display: grid; place-items: center; font-size: 18px; color: var(--muted); cursor: pointer; border-right: 1px solid var(--line); }
  .plus:hover { color: var(--text); background: #141d2a; }
  .tools { margin-left: auto; display: flex; }
  .tools button { all: unset; width: 40px; height: 40px; display: grid; place-items: center; color: var(--muted); border-left: 1px solid var(--line); cursor: pointer; }
  .tools button:hover { color: var(--text); background: #141d2a; }
</style>
