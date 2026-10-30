<script lang="ts">
  import { onDestroy } from "svelte";
  import Modal from "./Modal.svelte";
  import { VaultService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { copyText } from "../lib/clipboard";

  let { connection, onclose }: { connection: string; onclose: () => void } = $props();

  /** How long a revealed password stays on screen. */
  const SHOW_FOR = 20;

  let master = $state("");
  let shown = $state("");
  let left = $state(0);
  let error = $state("");
  let busy = $state(false);
  let copied = $state(false);
  let timer: ReturnType<typeof setInterval> | undefined;

  function hide() {
    clearInterval(timer);
    shown = "";
    left = 0;
    copied = false;
  }
  onDestroy(hide);

  async function reveal(e: Event) {
    e.preventDefault();
    if (!master || busy) return;
    busy = true;
    error = "";
    const typed = master;
    master = "";
    try {
      shown = await VaultService.RevealSecret(connection, typed);
      left = SHOW_FOR;
      timer = setInterval(() => { if (--left <= 0) hide(); }, 1000);
    } catch (err) {
      const m = (err as Error)?.message ?? String(err);
      error = /incorrect master/i.test(m) ? "That master password is not right." : /no password is saved/i.test(m) ? "No password is saved for this connection." : m;
    } finally {
      busy = false;
    }
  }

  async function copy() {
    copied = await copyText(shown);
  }
</script>

<Modal title="Show saved password" subtitle={connection} size="sm" {onclose}>
  {#if shown}
    <div class="field">
      <span>Password for {connection}</span>
      <input class="mono" readonly value={shown} aria-label="Saved password" data-autofocus onfocus={(e) => e.currentTarget.select()} />
    </div>
    <p class="hint">Hides itself in {left} seconds. Copying it puts it on your clipboard until you replace it.</p>
  {:else}
    <form id="reveal-form" onsubmit={reveal}>
      <label class="field">
        <span>Master password</span>
        <input type="password" bind:value={master} autocomplete="current-password" data-autofocus />
        <small>Needed every time, even while the vault is unlocked.</small>
      </label>
    </form>
    {#if error}<div class="notice bad" role="alert">{error}</div>{/if}
  {/if}
  {#snippet footer()}
    {#if shown}
      <button type="button" class="btn" onclick={copy}>{copied ? "Copied" : "Copy"}</button>
      <button type="button" class="btn primary" onclick={hide}>Hide</button>
    {:else}
      <button type="button" class="btn ghost" onclick={onclose}>Cancel</button>
      <button type="submit" form="reveal-form" class="btn primary" disabled={busy || !master}>{busy ? "Checking…" : "Show password"}</button>
    {/if}
  {/snippet}
</Modal>

<style>
  form { display: grid; gap: var(--s3); }
  .hint { margin: var(--s3) 0 0; }
  .notice { margin-top: var(--s3); }
</style>
