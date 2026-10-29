<script lang="ts">
  import { sb } from "./foldertree.svelte";
</script>

{#if sb.drag}
  <div class="ghost" style="left: {sb.drag.x + 14}px; top: {sb.drag.y + 10}px" class:bad={sb.dropTarget !== null && !sb.dropOk}>
    <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><circle cx="9" cy="6" r="1" /><circle cx="15" cy="6" r="1" /><circle cx="9" cy="12" r="1" /><circle cx="15" cy="12" r="1" /><circle cx="9" cy="18" r="1" /><circle cx="15" cy="18" r="1" /></svg>
    <b>{sb.drag.label}</b>
    {#if sb.dropTarget !== null && !sb.dropOk}<span>{sb.dropReason}</span>{/if}
  </div>
{/if}

<style>
  .ghost { position: fixed; z-index: 50; pointer-events: none; display: flex; align-items: center; gap: 8px; padding: 6px 12px 6px 8px; background: var(--active); border: 1px solid var(--accent); border-radius: 6px; box-shadow: 0 12px 30px var(--shadow); transform: rotate(-1.5deg); color: var(--muted); }
  .ghost b { color: var(--strong); font-weight: 600; }
  .ghost.bad { border-color: var(--err); }
  .ghost span { font-size: 11px; color: var(--err-text); }
  :global(body.dragging) { cursor: grabbing; user-select: none; }
</style>
