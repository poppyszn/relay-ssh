<script lang="ts">
  import { tick } from "svelte";
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { matches } from "../lib/lib";
  import { pop, veil } from "../lib/motion";
  import EmptyState from "./EmptyState.svelte";
  import Icon from "./Icon.svelte";

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
  let list = $state<HTMLElement | null>(null);

  const conns = $derived(profiles.filter((p) => matches(p, query)).slice(0, 8));
  const acts = $derived(query.trim() ? actions.filter((a) => a.label.toLowerCase().includes(query.trim().toLowerCase())) : actions);
  const total = $derived(conns.length + acts.length);
  $effect(() => { query; sel = 0; });
  $effect(() => { tick().then(() => input?.focus()); });
  // Keep the highlighted row in view as the arrows move through a long list.
  $effect(() => { sel; tick().then(() => list?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: "nearest" })); });

  function pick(i: number, split = false) {
    if (i < conns.length) onconnect(conns[i], split);
    else if (acts[i - conns.length]) onaction(acts[i - conns.length].id);
  }
  function key(e: KeyboardEvent) {
    // Stop here so a modal underneath does not also close on the same press.
    if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); onclose(); }
    else if (e.key === "ArrowDown") { sel = Math.min(sel + 1, total - 1); e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = Math.max(sel - 1, 0); e.preventDefault(); }
    else if (e.key === "Enter") { pick(sel, e.metaKey || e.ctrlKey); e.preventDefault(); }
  }
</script>

<div class="scrim" onclick={onclose} role="presentation" in:veil|global out:veil|global>
  <div class="box" role="dialog" aria-modal="true" aria-label="Quick open" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={key} in:pop|global out:pop|global>
    <div class="query">
      <Icon name="search" size={20} />
      <input bind:this={input} bind:value={query} placeholder="Connect to…" aria-label="Connect to" />
      <span class="hint">Search name, host, folder or #tag</span>
    </div>
    <div class="list" role="listbox" aria-label="Results" bind:this={list}>
      {#if conns.length}<div class="label sect">Connections</div>{/if}
      {#each conns as p, i (p.name)}
        <!-- svelte-ignore a11y_click_events_have_key_events (keys are handled on the dialog) -->
        <div class="row" role="option" tabindex="-1" aria-selected={i === sel} class:row-hl={i === sel} onmouseenter={() => (sel = i)} onclick={() => pick(i)}>
          <span class="dot" class:ok={active.has(p.name)}></span>
          <span class="col">
            <span class="name">
              {#if p.protocol === "rdp"}<span class="rdp-icon" title="RDP connection"><Icon name="rdp" size={14} /></span>{/if}
              {p.name}
            </span>
            <span class="sub mono">{p.folder ? p.folder + " · " : ""}{p.protocol === "rdp" ? "rdp · " : ""}{p.user}@{p.host}{p.port !== 22 && p.port !== 3389 ? ":" + p.port : ""}{p.jump ? " · via " + p.jump : ""}</span>
          </span>
          <span class="tags">{#each p.tags ?? [] as t}<span class="chip">{t}</span>{/each}{#if i === sel}<span class="kbd">↵</span>{/if}</span>
        </div>
      {/each}
      {#if acts.length}<div class="label sect">Actions</div>{/if}
      {#each acts as a, j (a.id)}
        {@const i = conns.length + j}
        <!-- svelte-ignore a11y_click_events_have_key_events (keys are handled on the dialog) -->
        <div class="row" role="option" tabindex="-1" aria-selected={i === sel} class:row-hl={i === sel} onmouseenter={() => (sel = i)} onclick={() => pick(i)}>
          <span class="col"><span>{a.label}</span></span><span class="kbd push">{a.hint}</span>
        </div>
      {/each}
      {#if !total}<EmptyState title="No matches" text="Try a connection name, host or tag." />{/if}
    </div>
    <div class="foot"><span><span class="kbd">↑↓</span> navigate</span><span><span class="kbd">↵</span> connect in new tab</span><span><span class="kbd">⌘↵</span> split current tab</span><span class="push"><span class="kbd">esc</span> close</span></div>
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; z-index: var(--z-modal); display: flex; justify-content: center; padding-top: 110px; background: var(--scrim); }
  .box { width: min(640px, 92vw); max-height: 70vh; align-self: flex-start; display: flex; flex-direction: column; background: var(--float); border: 1px solid var(--floatline); border-radius: var(--r-md); box-shadow: var(--e-modal); outline: none; overflow: hidden; }
  .query { display: flex; align-items: center; gap: var(--s3); padding: 0 var(--s4); height: 52px; border-bottom: 1px solid var(--line); color: var(--muted); }
  .query input, .query input:focus { flex: 1; height: 100%; padding: 0; border: 0; background: none; box-shadow: none; font-size: var(--t-lg); color: var(--strong); }
  .list { overflow: auto; padding: var(--s1) 0; }
  .sect { padding: var(--s2) var(--s4) var(--s1); }
  .row { display: flex; align-items: center; gap: var(--s3); height: 40px; padding: 0 var(--s4); cursor: pointer; outline: none; }
  .col { display: grid; min-width: 0; }
  .name { font-weight: 600; color: var(--strong); display: flex; align-items: center; gap: 4px; }
  .rdp-icon { display: inline-flex; align-items: center; color: var(--muted); }
  .sub { font-size: var(--t-xs); color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .tags { margin-left: auto; display: flex; gap: var(--s2); align-items: center; }
  .push { margin-left: auto; }
  .foot { display: flex; gap: var(--s4); padding: var(--s3) var(--s4); border-top: 1px solid var(--line); font-size: var(--t-xs); color: var(--muted); background: var(--foot); }
  .foot span { display: inline-flex; align-items: center; gap: var(--s1); }
</style>
