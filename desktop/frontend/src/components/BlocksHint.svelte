<script lang="ts">
  import { onMount } from "svelte";
  import Icon from "./Icon.svelte";
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
    <button type="button" class="btn sm primary" onclick={enable}>Enable</button>
    <button type="button" class="icon-btn dense" aria-label="Don't show this again" title="Don't show this again" onclick={dismiss}><Icon name="x" size={14} /></button>
  </div>
{/if}

<style>
  .hint { position: absolute; right: var(--s4); bottom: var(--s4); z-index: 3; display: flex; align-items: center; gap: var(--s3); padding: var(--s3) var(--s2) var(--s3) var(--s4); max-width: 460px; background: var(--active); border: 1px solid var(--floatline); border-radius: var(--r-md); box-shadow: var(--e-pop); }
  .hint div { display: grid; gap: 2px; min-width: 0; }
  strong { font-size: var(--t-md); color: var(--strong); }
  span { font-size: var(--t-sm); color: var(--muted); }
</style>
