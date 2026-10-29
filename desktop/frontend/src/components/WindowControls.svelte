<script lang="ts">
  import { Window } from "@wailsio/runtime";

  // Relay's own window buttons for the frameless window, drawn like macOS's and
  // centred in the 40px strip. Glyphs appear while the pointer is over the group.
  let hover = $state(false);
</script>

<div class="controls nodrag" role="group" aria-label="Window controls" onmouseenter={() => (hover = true)} onmouseleave={() => (hover = false)}>
  <button class="close" aria-label="Close window" title="Close" onclick={() => Window.Close()}>
    {#if hover}<svg width="8" height="8" viewBox="0 0 8 8" aria-hidden="true"><path d="M1 1l6 6M7 1L1 7" stroke="#4d0000" stroke-width="1.3" stroke-linecap="round" /></svg>{/if}
  </button>
  <button class="min" aria-label="Minimise window" title="Minimise" onclick={() => Window.Minimise()}>
    {#if hover}<svg width="8" height="8" viewBox="0 0 8 8" aria-hidden="true"><path d="M1 4h6" stroke="#5a3a00" stroke-width="1.3" stroke-linecap="round" /></svg>{/if}
  </button>
  <button class="zoom" aria-label="Zoom window" title="Zoom" onclick={() => Window.ToggleMaximise()}>
    {#if hover}<svg width="8" height="8" viewBox="0 0 8 8" aria-hidden="true"><path d="M1.2 1.2h3.3L1.2 4.5zM6.8 6.8H3.5l3.3-3.3z" fill="#0b3d12" /></svg>{/if}
  </button>
</div>

<style>
  .controls { position: absolute; left: 16px; top: 0; height: 40px; display: flex; align-items: center; gap: 8px; z-index: 2; }
  button { width: 12px; height: 12px; padding: 0; border: 0; border-radius: 50%; display: grid; place-items: center; box-shadow: inset 0 0 0 0.5px rgba(0, 0, 0, 0.25); }
  .close { background: #ff5f57; }
  .min { background: #febc2e; }
  .zoom { background: #28c840; }
  button:active { filter: brightness(0.8); }
</style>
