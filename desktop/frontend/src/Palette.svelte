<script lang="ts">
  import { tick } from "svelte";
  import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
  import { matches } from "./lib";

  let { profiles, onpick, onclose }: { profiles: Profile[]; onpick: (p: Profile) => void; onclose: () => void } = $props();
  let query = $state("");
  let sel = $state(0);
  let input: HTMLInputElement;
  const results = $derived(profiles.filter((p) => matches(p, query)).slice(0, 12));
  $effect(() => { query; sel = 0; });
  $effect(() => { tick().then(() => input?.focus()); });

  function key(e: KeyboardEvent) {
    if (e.key === "Escape") onclose();
    else if (e.key === "ArrowDown") { sel = Math.min(sel + 1, results.length - 1); e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = Math.max(sel - 1, 0); e.preventDefault(); }
    else if (e.key === "Enter" && results[sel]) onpick(results[sel]);
  }
</script>

<div class="scrim" onclick={onclose} role="presentation">
  <div class="box" role="dialog" aria-label="Quick open" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={key}>
    <input bind:this={input} bind:value={query} placeholder="Connect to…" aria-label="Connect to" />
    <ul role="listbox">
      {#each results as p, i (p.name)}
        <!-- svelte-ignore a11y_click_events_have_key_events -->
        <li role="option" aria-selected={i === sel} class:sel={i === sel} onmouseenter={() => (sel = i)} onclick={() => onpick(p)}>
          <span class="name">{p.name}</span>
          <span class="sub">{p.folder ? p.folder + " · " : ""}{p.user}@{p.host}</span>
        </li>
      {:else}
        <li class="none">No matches</li>
      {/each}
    </ul>
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; background: rgba(5, 9, 15, 0.55); display: flex; justify-content: center; padding-top: 12vh; z-index: 10; }
  .box { width: min(560px, 92vw); max-height: 60vh; display: flex; flex-direction: column; background: var(--panel); border: 1px solid var(--border); border-radius: 12px; box-shadow: 0 20px 60px rgba(0,0,0,.5); overflow: hidden; align-self: flex-start; }
  input { margin: 10px; font-size: 15px; padding: 9px 12px; }
  ul { list-style: none; margin: 0; padding: 0 6px 8px; overflow: auto; }
  li { display: flex; justify-content: space-between; gap: 12px; padding: 8px 10px; border-radius: 6px; cursor: pointer; }
  li.sel { background: var(--accent-bg); box-shadow: inset 2px 0 var(--accent); }
  .name { font-weight: 600; }
  .sub { color: var(--muted); font: 12px var(--mono); }
  .none { color: var(--muted); cursor: default; }
</style>
