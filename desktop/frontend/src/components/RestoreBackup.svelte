<script lang="ts">
  import Modal from "./Modal.svelte";
  import { ToolsService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

  /** Restores a chosen backup: asks for its passphrase (when it is encrypted) and confirms the replacement in one step. */
  let { path, encrypted, onrestored, onclose }: { path: string; encrypted: boolean; onrestored: () => void; onclose: () => void } = $props();

  let pass = $state("");
  let error = $state("");
  let busy = $state(false);
  let field = $state<HTMLInputElement | null>(null);
  const file = $derived(path.split(/[\\/]/).pop() ?? path);

  async function restore(e: Event) {
    e.preventDefault();
    if (busy || (encrypted && !pass)) return;
    busy = true;
    error = "";
    const typed = pass;
    pass = "";
    try {
      await ToolsService.Restore(path, typed);
      onrestored();
    } catch (err) {
      const m = (err as Error)?.message ?? String(err);
      error = /incorrect passphrase/i.test(m) ? "That passphrase does not open this backup." : m;
      field?.focus();
    } finally {
      busy = false;
    }
  }
</script>

<Modal title="Restore backup" subtitle={file} size="sm" role="alertdialog" dismissOnScrim={false} {onclose}>
  <form id="restore-form" onsubmit={restore}>
    {#if encrypted}
      <label class="field">
        <span>Backup passphrase</span>
        <input type="password" bind:this={field} bind:value={pass} autocomplete="off" data-autofocus />
        <small>The passphrase you chose when you made this backup. Relay never stored it.</small>
      </label>
    {:else}
      <div class="notice warn">This backup is not encrypted, so nothing proves it is yours. Only restore a file you made or trust. Connections that could not be written safely into your ssh config are refused.</div>
    {/if}
    <p class="hint">Restoring replaces your current connections and adopted keys with the ones in this backup.</p>
  </form>
  {#if error}<div class="notice bad" role="alert">{error}</div>{/if}
  {#snippet footer()}
    <button type="button" class="btn ghost" onclick={onclose}>Cancel</button>
    <button type="submit" form="restore-form" class="btn danger" disabled={busy || (encrypted && !pass)}>{busy ? "Restoring…" : encrypted ? "Restore" : "Restore anyway"}</button>
  {/snippet}
</Modal>

<style>
  form { display: grid; gap: var(--s3); }
  .hint { margin: 0; }
  .notice.bad { margin-top: var(--s3); }
</style>
