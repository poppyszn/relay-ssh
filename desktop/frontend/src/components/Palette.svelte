<script lang="ts">
  import { tick } from "svelte";
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { matches, parseTarget } from "../lib/lib";
  import { pop, veil } from "../lib/motion";
  import EmptyState from "./EmptyState.svelte";
  import Icon, { type IconName } from "./Icon.svelte";

  export interface Action { id: string; label: string; hint: string; icon?: IconName; group?: string }
  export interface PaletteTab { id: string; label: string; current: boolean }

  let { profiles, active, actions, tabs = [], onconnect, onselecttab, onnew, onaction, onclose }: {
    profiles: Profile[];
    active: Set<string>;
    actions: Action[];
    tabs?: PaletteTab[];
    onconnect: (p: Profile, split: boolean) => void;
    onselecttab?: (id: string) => void;
    /** Opens the new-connection form prefilled with a pasted `ssh user@host`. */
    onnew: (t: { user: string; host: string; port: number }) => void;
    onaction: (id: string) => void;
    onclose: () => void;
  } = $props();

  let query = $state("");
  let sel = $state(0);
  let input: HTMLInputElement;
  let list = $state<HTMLElement | null>(null);

  const q = $derived(query.trim().toLowerCase());
  const conns = $derived(profiles.filter((p) => matches(p, query)).slice(0, 8));
  const openTabs = $derived(
    onselecttab
      ? tabs.filter((t) => !q || t.label.toLowerCase().includes(q)).slice(0, 5)
      : [],
  );
  const acts = $derived(
    q
      ? actions.filter((a) => a.label.toLowerCase().includes(q) || a.group?.toLowerCase().includes(q))
      : actions,
  );
  const target = $derived(parseTarget(query));
  // A pasted target comes first: it is what the user typed, and it never connects without the form.
  const off = $derived(target ? 1 : 0);
  const total = $derived(off + conns.length + openTabs.length + acts.length);
  $effect(() => { query; sel = 0; });
  $effect(() => { tick().then(() => input?.focus()); });
  // Keep the highlighted row in view as the arrows move through a long list.
  $effect(() => { sel; tick().then(() => list?.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: "nearest" })); });

  function pick(i: number, split = false) {
    if (target && i === 0) return onnew(target);
    i -= off;
    if (i < conns.length) return onconnect(conns[i], split);
    i -= conns.length;
    if (i < openTabs.length) return onselecttab?.(openTabs[i].id);
    i -= openTabs.length;
    if (acts[i]) onaction(acts[i].id);
  }
  function key(e: KeyboardEvent) {
    // Stop here so a modal underneath does not also close on the same press.
    if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); onclose(); }
    else if (e.key === "ArrowDown") { sel = Math.min(sel + 1, Math.max(0, total - 1)); e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = Math.max(sel - 1, 0); e.preventDefault(); }
    else if (e.key === "Enter" && total > 0) { pick(sel, e.metaKey || e.ctrlKey); e.preventDefault(); }
  }
</script>

<div class="scrim nodrag" onclick={onclose} role="presentation" in:veil|global={{ duration: 180 }} out:veil|global={{ duration: 150 }}>
  <div class="box nodrag" role="dialog" aria-modal="true" aria-label="Quick open" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={key} in:pop|global={{ duration: 220, y: -10, scale: 0.965 }} out:pop|global={{ duration: 150, y: -6, scale: 0.975 }}>
    <div class="query">
      <Icon name="search" size={20} />
      <input bind:this={input} bind:value={query} placeholder="Jump to connection, tab, or command…" aria-label="Connect to" />
      {#if !query}<span class="hint">or paste ssh user@host</span>{/if}
    </div>
    <div class="list" role="listbox" aria-label="Results" bind:this={list}>
      {#if target}
        <!-- svelte-ignore a11y_click_events_have_key_events (keys are handled on the dialog) -->
        <div class="row" role="option" tabindex="-1" aria-selected={sel === 0} class:row-hl={sel === 0} onmouseenter={() => (sel = 0)} onclick={() => pick(0)}>
          <span class="lead"><Icon name="plus" size={14} /></span>
          <span class="col"><span class="name">New connection</span><span class="sub mono">{target.user ? target.user + "@" : ""}{target.host}{target.port !== 22 ? ":" + target.port : ""}</span></span>
          {#if sel === 0}<span class="kbd push">↵</span>{/if}
        </div>
      {/if}
      {#if conns.length}<div class="label sect">Connections</div>{/if}
      {#each conns as p, k (p.id)}
        {@const i = off + k}
        <!-- svelte-ignore a11y_click_events_have_key_events (keys are handled on the dialog) -->
        <div class="row" role="option" tabindex="-1" aria-selected={i === sel} class:row-hl={i === sel} onmouseenter={() => (sel = i)} onclick={() => pick(i)}>
          <span class="dot" class:ok={active.has(p.id)}></span>
          <span class="col">
            <span class="name">
              {#if p.protocol === "rdp"}<span class="rdp-icon" title="RDP connection"><Icon name="rdp" size={14} /></span>{/if}
              {p.name}
            </span>
            <span class="sub mono">{p.folder ? p.folder + " · " : ""}{p.protocol === "rdp" ? "rdp · " : ""}{p.user}@{p.host}{p.port !== 22 && p.port !== 3389 ? ":" + p.port : ""}{p.jump ? " · via " + (profiles.find((x) => x.id === p.jump)?.name ?? p.jump) : ""}</span>
          </span>
          <span class="tags">{#each p.tags ?? [] as t}<span class="chip">{t}</span>{/each}{#if i === sel}<span class="kbd">↵</span>{/if}</span>
        </div>
      {/each}
      {#if openTabs.length}<div class="label sect">Open tabs</div>{/if}
      {#each openTabs as t, k (t.id)}
        {@const i = off + conns.length + k}
        <!-- svelte-ignore a11y_click_events_have_key_events (keys are handled on the dialog) -->
        <div class="row" role="option" tabindex="-1" aria-selected={i === sel} class:row-hl={i === sel} onmouseenter={() => (sel = i)} onclick={() => pick(i)}>
          <span class="dot" class:ok={t.current}></span>
          <span class="col">
            <span class="name">{t.label}</span>
            <span class="sub">{t.current ? "Current tab" : "Switch to tab"}</span>
          </span>
          {#if i === sel}<span class="kbd push">↵</span>{/if}
        </div>
      {/each}
      {#if acts.length}<div class="label sect">Commands</div>{/if}
      {#each acts as a, j (a.id)}
        {@const i = off + conns.length + openTabs.length + j}
        <!-- svelte-ignore a11y_click_events_have_key_events (keys are handled on the dialog) -->
        <div class="row" role="option" tabindex="-1" aria-selected={i === sel} class:row-hl={i === sel} onmouseenter={() => (sel = i)} onclick={() => pick(i)}>
          <span class="act-ico"><Icon name={a.icon ?? "terminal"} size={14} /></span>
          <span class="col"><span class="act-label">{a.label}</span></span>
          {#if a.hint}<span class="kbd push">{a.hint}</span>{/if}
        </div>
      {/each}
      {#if !total}<EmptyState title="No matches" text="Try a connection name, host, tab, or command." />{/if}
    </div>
    <div class="foot"><span><span class="kbd">↑↓</span> navigate</span><span><span class="kbd">↵</span> open</span><span><span class="kbd">⌘↵</span> split pane</span><span class="push"><span class="kbd">esc</span> close</span></div>
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; z-index: var(--z-modal); display: flex; justify-content: center; padding-top: 96px; background: var(--scrim); backdrop-filter: blur(8px); }
  .box { width: min(650px, 92vw); max-height: 72vh; align-self: flex-start; display: flex; flex-direction: column; background: var(--float); border: 1px solid var(--float-edge); border-radius: 16px; box-shadow: var(--e-modal); outline: none; overflow: hidden; }
  .query { display: flex; align-items: center; gap: var(--s3); padding: 0 var(--s5); height: 54px; border-bottom: 1px solid var(--hairline); color: var(--accent-text); }
  .query input, .query input:focus { flex: 1; height: 100%; padding: 0; border: 0; background: none; box-shadow: none; font-size: var(--t-lg); color: var(--strong); }
  .hint { font-size: var(--t-xs); color: var(--dim); white-space: nowrap; }
  .list { overflow: auto; padding: var(--s2); }
  .sect { padding: var(--s2) var(--s3) var(--s1); }
  .row { display: flex; align-items: center; gap: var(--s3); height: 44px; padding: 0 var(--s3); border-radius: 10px; cursor: pointer; outline: none; font-size: var(--t-md); }
  .col { display: grid; min-width: 0; }
  .name { font-weight: 500; font-size: var(--t-md); color: var(--strong); display: flex; align-items: center; gap: 6px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .act-ico { display: inline-flex; align-items: center; justify-content: center; width: 16px; color: var(--muted); flex-shrink: 0; }
  .act-label { font-size: var(--t-md); color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .row-hl .act-label { color: var(--strong); }
  .row-hl .act-ico { color: var(--accent-text); }
  .rdp-icon { display: inline-flex; align-items: center; color: var(--muted); }
  .lead { display: inline-grid; place-items: center; width: 16px; color: var(--accent-text); }
  .sub { font-size: var(--t-xs); color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .tags { margin-left: auto; display: flex; gap: var(--s2); align-items: center; }
  .push { margin-left: auto; }
  .foot { display: flex; gap: var(--s4); padding: var(--s3) var(--s4); border-top: 1px solid var(--hairline); font-size: var(--t-xs); color: var(--muted); background: var(--foot); }
  .foot span { display: inline-flex; align-items: center; gap: var(--s1); }
</style>
