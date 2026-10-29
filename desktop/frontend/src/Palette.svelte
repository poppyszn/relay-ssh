<script lang="ts">
  import { tick } from "svelte";
  import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { matches } from "./lib";

  export interface Action { id: string; label: string; hint: string }
  let { profiles, active, actions, onconnect, onaction, onclose }: {
    profiles: Profile[];
    active: Set<string>;
    actions: Action[];
    onconnect: (p: Profile, split: boolean) => void;
    onaction: (id: string) => void;
    onclose: () => void;
  } = $props();

  let query = $state("");
  let sel = $state(0);
  let input: HTMLInputElement;

  const conns = $derived(profiles.filter((p) => matches(p, query)).slice(0, 8));
  const acts = $derived(query.trim() ? actions.filter((a) => a.label.toLowerCase().includes(query.trim().toLowerCase())) : actions);
  const total = $derived(conns.length + acts.length);
  $effect(() => { query; sel = 0; });
  $effect(() => { tick().then(() => input?.focus()); });

  function pick(i: number, split = false) {
    if (i < conns.length) onconnect(conns[i], split);
    else if (acts[i - conns.length]) onaction(acts[i - conns.length].id);
  }
  function key(e: KeyboardEvent) {
    if (e.key === "Escape") onclose();
    else if (e.key === "ArrowDown") { sel = Math.min(sel + 1, total - 1); e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = Math.max(sel - 1, 0); e.preventDefault(); }
    else if (e.key === "Enter") { pick(sel, e.metaKey || e.ctrlKey); e.preventDefault(); }
  }
</script>

<div class="scrim" onclick={onclose} role="presentation">
  <div class="box" role="dialog" aria-label="Quick open" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={key}>
    <div class="field">
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="M20 20l-3.5-3.5" /></svg>
      <input bind:this={input} bind:value={query} placeholder="Connect to…" aria-label="Connect to" />
      <span class="hint">Search name, host, folder or #tag</span>
    </div>
    <div class="list" role="listbox">
      {#if conns.length}<div class="label sect">Connections</div>{/if}
      {#each conns as p, i (p.name)}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div class="row" role="option" aria-selected={i === sel} class:sel={i === sel} onmouseenter={() => (sel = i)} onclick={() => pick(i)}>
          <span class="dot" class:ok={active.has(p.name)}></span>
          <span class="col"><span class="name">{p.name}</span><span class="sub mono">{p.folder ? p.folder + " · " : ""}{p.user}@{p.host}{p.port !== 22 ? ":" + p.port : ""}{p.jump ? " · via " + p.jump : ""}</span></span>
          <span class="tags">{#each p.tags ?? [] as t}<span class="chip">{t}</span>{/each}{#if i === sel}<span class="kbd">↵</span>{/if}</span>
        </div>
      {/each}
      {#if acts.length}<div class="label sect">Actions</div>{/if}
      {#each acts as a, j (a.id)}
        {@const i = conns.length + j}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <div class="row act" role="option" aria-selected={i === sel} class:sel={i === sel} onmouseenter={() => (sel = i)} onclick={() => pick(i)}>
          <span class="col"><span>{a.label}</span></span><span class="kbd">{a.hint}</span>
        </div>
      {/each}
      {#if !total}<div class="none">No matches</div>{/if}
    </div>
    <div class="foot"><span><b>↑↓</b> navigate</span><span><b>↵</b> connect in new tab</span><span><b>⌘↵</b> split current tab</span><span class="right"><b>esc</b> close</span></div>
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; background: var(--scrim); display: flex; justify-content: center; padding-top: 110px; z-index: 20; }
  .box { width: min(640px, 92vw); max-height: 70vh; align-self: flex-start; display: flex; flex-direction: column; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 24px 70px var(--shadow); overflow: hidden; }
  .field { display: flex; align-items: center; gap: 12px; padding: 0 18px; height: 52px; border-bottom: 1px solid var(--line2); color: var(--muted); }
  .field input { flex: 1; height: 100%; border: 0; background: none; padding: 0; font-size: 16px; color: var(--strong); box-shadow: none; }
  .field input:focus { box-shadow: none; }
  .hint { font-size: 11.5px; }
  .list { overflow: auto; padding: 6px 0 4px; }
  .sect { padding: 8px 18px 6px; }
  .row { display: flex; align-items: center; gap: 12px; padding: 10px 18px; cursor: pointer; }
  .row.sel { background: var(--accent-hl); box-shadow: inset 3px 0 var(--accent); }
  .col { display: grid; gap: 1px; min-width: 0; }
  .name { font-weight: 600; color: var(--strong); }
  .sub { font-size: 11.5px; color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .tags { margin-left: auto; display: flex; gap: 6px; align-items: center; }
  .act .kbd { margin-left: auto; }
  .none { padding: 14px 18px; color: var(--muted); }
  .foot { display: flex; gap: 16px; padding: 10px 18px; border-top: 1px solid var(--line2); font-size: 11.5px; color: var(--muted); background: var(--foot); }
  .foot b { font-weight: 400; }
  .right { margin-left: auto; }
</style>
