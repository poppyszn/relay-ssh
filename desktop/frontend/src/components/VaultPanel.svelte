<script lang="ts">
  import { VaultService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { vault } from "../state/vault.svelte";
  import Modal from "./Modal.svelte";

  type Mode = "auto" | "create" | "unlock" | "unlock-connect";

  let {
    mode = "auto",
    connName = "",
    onclose,
    onsuccess,
    onskip,
  }: {
    mode?: Mode;
    connName?: string;
    onclose: () => void;
    onsuccess?: () => void;
    onskip?: () => void;
  } = $props();

  // The prompt keeps the mode it opened with; "auto" means create when there is no vault, else unlock.
  // svelte-ignore state_referenced_locally
  const current: Exclude<Mode, "auto"> = mode === "auto" ? (vault.exists ? "unlock" : "create") : mode;
  const creating = current === "create";

  let masterInput = $state("");
  let confirmInput = $state("");
  let error = $state("");
  let busy = $state(false);

  const formId = `vault-prompt-${Math.random().toString(36).slice(2, 8)}`;
  const title = $derived(creating ? "Create vault" : current === "unlock-connect" && connName ? `Unlock vault for ${connName}` : "Unlock vault");

  function clearInputs() {
    masterInput = "";
    confirmInput = "";
  }

  async function handleCreate(e: Event) {
    e.preventDefault();
    error = "";
    if (!masterInput) {
      error = "Enter a master password.";
      return;
    }
    if (masterInput !== confirmInput) {
      error = "Passwords do not match. Type the same password in both fields.";
      return;
    }
    const secret = masterInput;
    clearInputs();
    busy = true;
    try {
      await VaultService.Create(secret);
      await vault.refresh();
      onsuccess?.();
      onclose();
    } catch (err) {
      error = (err as Error)?.message ?? String(err);
    } finally {
      clearInputs();
      busy = false;
    }
  }

  async function handleUnlock(e: Event) {
    e.preventDefault();
    error = "";
    if (!masterInput) {
      error = "Enter your master password.";
      return;
    }
    const secret = masterInput;
    clearInputs();
    busy = true;
    try {
      await VaultService.Unlock(secret);
      await vault.refresh();
      onsuccess?.();
      onclose();
    } catch {
      error = "Wrong master password or corrupt vault.";
    } finally {
      clearInputs();
      busy = false;
    }
  }
</script>

<Modal {title} size="sm" {onclose}>
  <form id={formId} onsubmit={creating ? handleCreate : handleUnlock}>
    {#if creating}
      <p class="desc">Saved connection passwords are encrypted with your master password and filled in when ssh asks.</p>
      <div class="notice warn" role="note">If you forget the master password the only way back in is to reset the vault, which deletes the saved passwords.</div>
    {:else if current === "unlock-connect"}
      <p class="desc">Enter your master password to fill in the password for <strong>{connName}</strong>.</p>
    {:else}
      <p class="desc">Enter your master password to unlock your saved SSH passwords.</p>
    {/if}
    {#if error}<div class="notice bad" role="alert">{error}</div>{/if}
    <label class="field">
      <span>Master password</span>
      <input type="password" bind:value={masterInput} required placeholder={creating ? "Choose a master password" : "Master password"} aria-label="Master password" autocomplete={creating ? "new-password" : "current-password"} />
    </label>
    {#if creating}
      <label class="field">
        <span>Confirm master password</span>
        <input type="password" bind:value={confirmInput} required placeholder="Repeat master password" aria-label="Confirm master password" autocomplete="new-password" />
      </label>
    {:else}
      <p class="hint">Forgot it? Reset the vault from Settings.</p>
    {/if}
  </form>
  {#snippet footer()}
    <button type="button" class="btn ghost" onclick={onclose}>Cancel</button>
    {#if current === "unlock-connect"}
      <button type="button" class="btn" onclick={onskip} aria-label="Skip unlocking vault">Skip</button>
    {/if}
    <button type="submit" form={formId} class="btn primary" disabled={busy}>{creating ? (busy ? "Creating…" : "Create vault") : busy ? "Unlocking…" : "Unlock"}</button>
  {/snippet}
</Modal>

<style>
  form { display: grid; gap: var(--s4); }
  .desc, .hint { margin: 0; }
  .desc { font-size: var(--t-md); color: var(--text2); line-height: 1.5; }
  .desc strong { color: var(--strong); font-weight: 600; }
</style>
