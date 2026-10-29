<script lang="ts">
  import { onMount } from "svelte";
  import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { connectingHint, formatElapsed } from "./lib";

  let { target, prof, since, oncancel }: { target: string; prof?: Profile; since: number; oncancel: () => void } = $props();

  let now = $state(Date.now());
  onMount(() => {
    const t = setInterval(() => (now = Date.now()), 500);
    return () => clearInterval(t);
  });
  const ms = $derived(Math.max(0, now - since));
  const hint = $derived(connectingHint(ms, !!prof?.jump));
</script>

<div class="connecting" role="status" aria-live="polite">
  <svg class="spin" width="30" height="30" viewBox="0 0 30 30" aria-hidden="true">
    <circle class="track" cx="15" cy="15" r="12" />
    <circle class="arc" cx="15" cy="15" r="12" />
  </svg>
  <h3>Connecting to {target}</h3>
  {#if prof}<p class="mono">{prof.user}@{prof.host}:{prof.port}</p>{/if}
  {#if prof?.jump}<p class="route">via {prof.jump}</p>{/if}
  <p class="time">{formatElapsed(ms)}</p>
  <p class="hint" class:on={!!hint}>{hint}</p>
  <button class="btn" onclick={oncancel}>Cancel</button>
</div>

<style>
  .connecting { position: absolute; inset: 0; z-index: 2; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 4px; background: var(--term); text-align: center; padding: 24px; }
  h3 { margin: 14px 0 0; font-size: 15px; font-weight: 600; color: var(--strong); }
  p { margin: 0; font-size: 12px; color: var(--muted); }
  .mono { font-family: var(--mono); font-size: 11.5px; }
  .route { color: var(--dim); }
  .time { margin-top: 8px; font-variant-numeric: tabular-nums; color: var(--dim); }
  .hint { max-width: 340px; min-height: 34px; line-height: 1.45; color: var(--warn); opacity: 0; transition: opacity 0.3s; }
  .hint.on { opacity: 1; }
  .btn { margin-top: 10px; }
  .spin { animation: turn 1s linear infinite; }
  .track { fill: none; stroke: var(--line2); stroke-width: 3; }
  .arc { fill: none; stroke: var(--accent); stroke-width: 3; stroke-linecap: round; stroke-dasharray: 20 76; }
  @keyframes turn { to { transform: rotate(360deg); } }
  @media (prefers-reduced-motion: reduce) { .spin { animation: none; } .arc { stroke-dasharray: 60 16; } }
</style>
