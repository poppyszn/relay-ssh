<script lang="ts">
  import { onMount } from "svelte";
  import { ws } from "../state/workspace.svelte";

  let { id }: { id: string } = $props();

  const KEY = "relay.blockshint.off.v1";
  const off = () => {
    try {
      return localStorage.getItem(KEY) === "1";
    } catch {
      return false;
    }
  };

  // Shown a moment after the login, once, per window (until enabled or dismissed for good).
  let shown = $state(false);
  let hidden = $state(off());
  onMount(() => {
    const t = setTimeout(() => (shown = true), 2500);
    return () => clearTimeout(t);
  });
  function dismiss() {
    hidden = true;
    try {
      localStorage.setItem(KEY, "1");
    } catch {
      // best effort
    }
  }
  function enable() {
    hidden = true;
    ws.enableBlocks(id);
  }
</script>

{#if shown && !hidden}
  <div class="hint" role="status">
    <div>
      <strong>Command blocks are off in this SSH session</strong>
      <span>Needs a short setup line typed into the remote bash or zsh.</span>
    </div>
    <button class="btn primary" onclick={enable}>Enable</button>
    <button class="x" aria-label="Don't show this again" title="Don't show this again" onclick={dismiss}>&times;</button>
  </div>
{/if}

<style>
  .hint { position: absolute; right: 16px; bottom: 16px; z-index: 3; display: flex; align-items: center; gap: 12px; padding: 10px 8px 10px 14px; max-width: 460px; background: var(--active); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 10px 30px var(--shadow); }
  .hint div { display: grid; gap: 1px; min-width: 0; }
  strong { font-size: 12.5px; color: var(--strong); }
  span { font-size: 11.5px; color: var(--muted); }
  .x { all: unset; width: 24px; height: 24px; display: grid; place-items: center; font-size: 16px; color: var(--dim); cursor: pointer; }
  .x:hover { color: var(--text); }
</style>
