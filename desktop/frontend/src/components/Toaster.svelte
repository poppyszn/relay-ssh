<script lang="ts">
  import { pop } from "../lib/motion";
  import { toasts, toast, type ToastItem } from "../state/toasts.svelte";

  let {
    theme = "dark",
    position = "bottom-center",
  }: {
    theme?: "dark" | "light";
    position?: "bottom-center";
  } = $props();

  function runAction(item: ToastItem) {
    try {
      item.action?.onClick();
    } finally {
      toast.dismiss(item.id);
    }
  }
</script>

<div class="toaster" data-theme={theme} data-position={position}>
  {#each toasts.items as item (item.id)}
    <div
      class="toast nodrag {item.kind}"
      role="status"
      aria-live="polite"
      in:pop|global={{ duration: 190, y: 8, scale: 0.96 }}
      out:pop|global={{ duration: 140, y: 6, scale: 0.97 }}
    >
      <span class="toast-icon" aria-hidden="true">
        {#if item.kind === "success"}
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9" /><path d="m9 12 2 2 4-4" /></svg>
        {:else if item.kind === "error"}
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9" /><path d="M12 8v4" /><path d="M12 16h.01" /></svg>
        {:else}
          <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="9" /><path d="M12 16v-4" /><path d="M12 8h.01" /></svg>
        {/if}
      </span>

      <div class="toast-copy">
        <div class="toast-title">{item.message}</div>
        {#if item.description}
          <div class="toast-desc">{item.description}</div>
        {/if}
      </div>

      {#if item.action}
        <button type="button" class="toast-action" onclick={() => runAction(item)}>
          {item.action.label}
        </button>
      {/if}

      <button
        type="button"
        class="toast-close"
        aria-label="Dismiss notification"
        onclick={() => toast.dismiss(item.id)}
      >
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M18 6 6 18" /><path d="m6 6 12 12" /></svg>
      </button>
    </div>
  {/each}
</div>

<style>
  .toaster {
    position: fixed;
    left: 50%;
    bottom: 24px;
    transform: translateX(-50%);
    z-index: var(--z-toast);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    pointer-events: none;
    max-width: calc(100vw - 32px);
  }

  .toast {
    pointer-events: auto;
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 240px;
    max-width: 380px;
    padding: 11px 12px;
    border-radius: var(--r-md);
    background: var(--float);
    border: 1px solid var(--floatline);
    color: var(--text);
    box-shadow: var(--e-pop);
    font-size: var(--t-sm);
    line-height: 1.4;
  }

  .toast-icon {
    display: inline-flex;
    flex: none;
    color: var(--accent-text);
  }
  .toast.success .toast-icon { color: var(--ok); }
  .toast.error .toast-icon { color: var(--err); }

  .toast-copy {
    flex: 1;
    min-width: 0;
    display: grid;
    gap: 2px;
  }

  .toast-title {
    font-weight: 500;
    color: var(--strong);
  }

  .toast-desc {
    font-size: var(--t-xs);
    color: var(--muted);
  }

  .toast-action {
    flex: none;
    height: 26px;
    padding: 0 10px;
    border: 0;
    border-radius: var(--r-sm);
    background: var(--cta-bg);
    color: var(--cta-fg);
    font-size: var(--t-xs);
    font-weight: 600;
    white-space: nowrap;
    transition: background var(--dur-1) var(--ease);
  }
  .toast-action:hover { background: var(--cta-hover); }

  .toast-close {
    flex: none;
    width: 22px;
    height: 22px;
    padding: 0;
    border: 0;
    border-radius: var(--r-xs);
    display: inline-grid;
    place-items: center;
    color: var(--muted);
    background: transparent;
    transition: background var(--dur-1) var(--ease), color var(--dur-1) var(--ease);
  }
  .toast-close:hover {
    background: var(--hover);
    color: var(--strong);
  }
  .toast-close svg { pointer-events: none; }
</style>
