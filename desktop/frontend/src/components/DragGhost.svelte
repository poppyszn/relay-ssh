<script lang="ts">
  import { sb } from "../state/foldertree.svelte";
  import Icon from "./Icon.svelte";
</script>

{#if sb.drag}
  <div class="ghost" style="left: {sb.drag.x + 14}px; top: {sb.drag.y + 10}px" class:bad={sb.dropTarget !== null && !sb.dropOk}>
    <Icon name="grip" size={14} />
    <b>{sb.drag.label}</b>
    {#if sb.dropTarget !== null && !sb.dropOk}<span>{sb.dropReason}</span>{/if}
  </div>
{/if}

<style>
  .ghost { position: fixed; z-index: var(--z-drag); pointer-events: none; display: flex; align-items: center; gap: var(--s2); padding: var(--s2) var(--s3) var(--s2) var(--s2); background: var(--active); border: 1px solid var(--accent); border-radius: var(--r-sm); box-shadow: var(--e-pop); color: var(--muted); }
  .ghost b { color: var(--strong); font-weight: 600; }
  .ghost.bad { border-color: var(--err); }
  .ghost span { font-size: var(--t-xs); color: var(--err-text); }
  :global(body.dragging) { cursor: grabbing; user-select: none; }
</style>
