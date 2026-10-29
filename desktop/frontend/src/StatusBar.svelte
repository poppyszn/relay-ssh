<script lang="ts">
  import { ws } from "./workspace.svelte";
  import { vault } from "./vault.svelte";
  import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

  let { profiles, themeName, onvault }: { profiles: Profile[]; themeName: string; onvault?: () => void } = $props();
  const info = $derived(ws.panes[ws.focusedPane]);
  const prof = $derived(info?.target ? profiles.find((p) => p.name === info.target) : undefined);
  const size = $derived.by(() => {
    const t = ws.session(ws.focusedPane)?.term;
    return t ? `${t.cols}×${t.rows}` : "";
  });
</script>

<footer>
  {#if info}
    <span class="seg"><span class="dot" class:ok={!info.ended && !info.connecting} class:warn={info.connecting} class:err={info.ended}></span>{info.ended ? "Disconnected" : info.connecting ? "Connecting…" : "Connected"}</span>
    <span class="seg mono">{prof ? `${prof.user}@${prof.host}:${prof.port}` : "local shell"}</span>
    {#if prof?.jump}<span class="seg">via {prof.jump}</span>{/if}
    {#if info.filled}<span class="seg ok" title="Password auto-filled from Relay's vault">Password filled</span>{/if}
    {#if info.blocks}<span class="seg ok">Blocks</span>
    {:else if info.target && !info.connecting && !info.ended}
      <button class="seg act" title="Splits this session's output into command blocks. Types a short setup line into the remote bash or zsh." onclick={() => ws.enableBlocks(ws.focusedPane)}>Enable command blocks</button>
    {/if}
  {:else}
    <span class="seg">No session</span>
  {/if}
  <span class="spacer"></span>
  <button
    type="button"
    class="seg act vault-btn"
    onclick={onvault}
    title={vault.exists ? (vault.unlocked ? "Vault is unlocked" : "Vault is locked") : "Set up vault"}
    aria-label={vault.exists ? (vault.unlocked ? "Vault unlocked" : "Vault locked") : "Vault setup"}
  >
    {#if !vault.exists}
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>
      <span>Vault</span>
    {:else if vault.unlocked}
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 9.9-1" /></svg>
      <span class="ok">Vault</span>
    {:else}
      <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>
      <span>Vault (locked)</span>
    {/if}
  </button>
  <span class="seg mono">UTF-8 {size}</span>
  <span class="seg">{themeName}</span>
</footer>

<style>
  footer { height: 28px; flex: none; display: flex; align-items: stretch; background: var(--bar); border-top: 1px solid var(--line); font-size: 11.5px; color: var(--muted); }
  .seg { display: flex; align-items: center; gap: 6px; padding: 0 14px; border-right: 1px solid var(--line); }
  .spacer { flex: 1; }
  .act { all: unset; box-sizing: border-box; display: flex; align-items: center; padding: 0 14px; border-right: 1px solid var(--line); color: var(--accent-text); cursor: pointer; }
  .act:hover { background: var(--hover); color: var(--strong); }
  .ok { color: var(--ok); }
  .seg:last-child { border-right: 0; border-left: 1px solid var(--line); }
  .spacer + .seg { border-left: 1px solid var(--line); }
  .dot { width: 7px; height: 7px; }
</style>
