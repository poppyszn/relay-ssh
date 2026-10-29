<script lang="ts">
  import { ws } from "./workspace.svelte";
  import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";

  let { profiles, themeName }: { profiles: Profile[]; themeName: string } = $props();
  const info = $derived(ws.panes[ws.focusedPane]);
  const prof = $derived(info?.target ? profiles.find((p) => p.name === info.target) : undefined);
  const size = $derived.by(() => {
    const t = ws.session(ws.focusedPane)?.term;
    return t ? `${t.cols}×${t.rows}` : "";
  });
</script>

<footer>
  {#if info}
    <span class="seg"><span class="dot" class:ok={!info.ended} class:err={info.ended}></span>{info.ended ? "Disconnected" : "Connected"}</span>
    <span class="seg mono">{prof ? `${prof.user}@${prof.host}:${prof.port}` : "local shell"}</span>
    {#if prof?.jump}<span class="seg">via {prof.jump}</span>{/if}
  {:else}
    <span class="seg">No session</span>
  {/if}
  <span class="spacer"></span>
  <span class="seg mono">UTF-8 {size}</span>
  <span class="seg">{themeName}</span>
</footer>

<style>
  footer { height: 28px; flex: none; display: flex; align-items: stretch; background: var(--bar); border-top: 1px solid var(--line); font-size: 11.5px; color: var(--muted); }
  .seg { display: flex; align-items: center; gap: 6px; padding: 0 14px; border-right: 1px solid var(--line); }
  .spacer { flex: 1; }
  .seg:last-child { border-right: 0; border-left: 1px solid var(--line); }
  .spacer + .seg { border-left: 1px solid var(--line); }
  .dot { width: 7px; height: 7px; }
</style>
