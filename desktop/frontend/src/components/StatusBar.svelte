<script lang="ts">
  import { ws } from "../state/workspace.svelte";
  import { vault } from "../state/vault.svelte";
  import Icon from "./Icon.svelte";
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

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
    <span class="cell"><span class="dot" class:ok={!info.ended && !info.connecting} class:warn={info.connecting} class:err={info.ended}></span>{info.ended ? "Disconnected" : info.connecting ? "Connecting…" : "Connected"}</span>
    <span class="cell mono">{prof ? `${prof.user}@${prof.host}:${prof.port}` : "local shell"}</span>
    {#if prof?.jump}<span class="cell">via {prof.jump}</span>{/if}
    {#if info.filled}<span class="cell ok" title="Password auto-filled from Relay's vault">Password filled</span>{/if}
    {#if info.blocks}<span class="cell ok">Blocks</span>
    {:else if info.target && !info.connecting && !info.ended}
      <button type="button" class="cell act" title="Splits this session's output into command blocks. Types a short setup line into the remote bash or zsh." onclick={() => ws.enableBlocks(ws.focusedPane)}>Enable command blocks</button>
    {/if}
  {:else}
    <span class="cell">No session</span>
  {/if}
  <span class="spacer"></span>
  <button
    type="button"
    class="cell act vault-btn"
    onclick={onvault}
    title={vault.exists ? (vault.unlocked ? "Vault is unlocked. Open the vault settings." : "Vault is locked. Open the vault settings.") : "Set up the password vault"}
    aria-label={vault.exists ? (vault.unlocked ? "Vault unlocked" : "Vault locked") : "Vault setup"}
  >
    <Icon name={vault.unlocked ? "unlock" : "lock"} size={14} />
    <span class:ok={vault.unlocked}>{!vault.exists ? "Vault" : vault.unlocked ? "Vault" : "Vault locked"}</span>
  </button>
  <span class="cell mono">UTF-8 {size}</span>
  <span class="cell">{themeName}</span>
</footer>

<style>
  footer { height: 28px; flex: none; display: flex; align-items: center; gap: var(--s1); padding: 0 var(--s2); background: var(--bar); border-top: 1px solid var(--line); font-size: var(--t-sm); color: var(--muted); }
  .cell { display: flex; align-items: center; gap: 6px; padding: 0 var(--s2); height: 100%; }
  .spacer { flex: 1; }
  .act { background: none; border: 0; color: var(--accent-text); font: inherit; cursor: pointer; border-radius: var(--r-xs); transition: background var(--dur-1) var(--ease); }
  .act:hover { background: var(--hover); color: var(--strong); }
  .ok { color: var(--ok); }
  .vault-btn { color: var(--muted); }
</style>
