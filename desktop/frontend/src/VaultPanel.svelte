<script lang="ts">
  import { VaultService } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { vault } from "./vault.svelte";

  type Mode = "auto" | "create" | "unlock" | "unlock-connect" | "manage" | "reset";

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

  function initialMode(): "create" | "unlock" | "unlock-connect" | "manage" | "reset" {
    if (mode === "unlock-connect") return "unlock-connect";
    if (mode === "create") return "create";
    if (mode === "unlock") return "unlock";
    if (mode === "manage") return "manage";
    if (mode === "reset") return "reset";
    if (!vault.exists) return "create";
    if (!vault.unlocked) return "unlock";
    return "manage";
  }

  // svelte-ignore state_referenced_locally
  let currentMode = $state<"create" | "unlock" | "unlock-connect" | "manage" | "reset">(initialMode());

  let masterInput = $state("");
  let confirmInput = $state("");
  let oldMasterInput = $state("");
  let resetConfirmInput = $state("");

  let error = $state("");
  let notice = $state("");
  let busy = $state(false);

  let showChangeMaster = $state(false);
  let showReset = $state(false);

  const title = $derived.by(() => {
    switch (currentMode) {
      case "create": return "Create Vault";
      case "unlock": return "Unlock Vault";
      case "unlock-connect": return connName ? `Unlock Vault for ${connName}` : "Unlock Vault";
      case "reset": return "Reset Vault";
      case "manage": return "Vault";
    }
  });

  function clearInputs() {
    masterInput = "";
    confirmInput = "";
    oldMasterInput = "";
    resetConfirmInput = "";
  }

  async function handleCreate(e: Event) {
    e.preventDefault();
    error = "";
    notice = "";
    if (!masterInput) {
      error = "Enter a master password.";
      return;
    }
    if (masterInput !== confirmInput) {
      error = "Passwords do not match.";
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
    notice = "";
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

  async function handleLock() {
    busy = true;
    error = "";
    notice = "";
    try {
      await vault.lock();
      currentMode = "unlock";
    } catch (err) {
      error = (err as Error)?.message ?? String(err);
    } finally {
      busy = false;
    }
  }

  async function handleChangeMaster(e: Event) {
    e.preventDefault();
    error = "";
    notice = "";
    if (!oldMasterInput) {
      error = "Enter your current master password.";
      return;
    }
    if (!masterInput) {
      error = "Enter a new master password.";
      return;
    }
    if (masterInput !== confirmInput) {
      error = "New passwords do not match.";
      return;
    }
    const oldP = oldMasterInput;
    const newP = masterInput;
    clearInputs();
    busy = true;
    try {
      await VaultService.ChangeMaster(oldP, newP);
      notice = "Master password changed successfully.";
      showChangeMaster = false;
    } catch {
      error = "Wrong current master password or update failed.";
    } finally {
      clearInputs();
      busy = false;
    }
  }

  async function handleReset(e: Event) {
    e.preventDefault();
    error = "";
    notice = "";
    if (resetConfirmInput !== "RESET") {
      error = "Type RESET in all capital letters to confirm.";
      return;
    }
    busy = true;
    try {
      await vault.reset("RESET");
      clearInputs();
      showReset = false;
      currentMode = "create";
      notice = "Vault has been reset.";
    } catch (err) {
      error = (err as Error)?.message ?? String(err);
    } finally {
      clearInputs();
      busy = false;
    }
  }
</script>

<div class="scrim" role="presentation" onclick={onclose} onkeydown={(e) => e.key === "Escape" && onclose()}>
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="panel" role="dialog" aria-label={title} tabindex="-1" onclick={(e) => e.stopPropagation()}>
    <header>
      <h2>{title}</h2>
      <button class="x" aria-label="Close" onclick={onclose}>&times;</button>
    </header>

    {#if currentMode === "create"}
      <form onsubmit={handleCreate}>
        <div class="warn-box" role="note">
          Your master password encrypts all stored SSH passwords. It cannot be recovered or reset if forgotten.
        </div>
        {#if error}<div class="note bad" role="alert">{error}</div>{/if}
        {#if notice}<div class="note good">{notice}</div>{/if}
        <label>
          <span>Master password</span>
          <!-- svelte-ignore a11y_autofocus -->
          <input type="password" bind:value={masterInput} autofocus required placeholder="Choose a master password" aria-label="Master password" autocomplete="new-password" />
        </label>
        <label>
          <span>Confirm master password</span>
          <input type="password" bind:value={confirmInput} required placeholder="Repeat master password" aria-label="Confirm master password" autocomplete="new-password" />
        </label>
        <div class="actions">
          <button type="button" class="btn ghost" onclick={onclose}>Cancel</button>
          <button type="submit" class="btn primary" disabled={busy}>{busy ? "Creating…" : "Create Vault"}</button>
        </div>
      </form>

    {:else if currentMode === "unlock" || currentMode === "unlock-connect"}
      <form onsubmit={handleUnlock}>
        <p class="desc">
          {#if currentMode === "unlock-connect"}
            Enter your master password to auto-fill the password for <strong>{connName}</strong>.
          {:else}
            Enter your master password to unlock your stored SSH passwords.
          {/if}
        </p>
        {#if error}<div class="note bad" role="alert">{error}</div>{/if}
        <label>
          <span>Master password</span>
          <!-- svelte-ignore a11y_autofocus -->
          <input type="password" bind:value={masterInput} autofocus required placeholder="Master password" aria-label="Master password" autocomplete="current-password" />
        </label>
        <div class="actions">
          {#if currentMode === "unlock-connect"}
            <button type="button" class="btn ghost" onclick={onskip} aria-label="Skip unlocking vault">Skip</button>
            <button type="button" class="btn ghost" onclick={onclose}>Cancel</button>
          {:else}
            <button type="button" class="btn text-btn" onclick={() => { error = ""; currentMode = "reset"; }}>Forgot password? Reset</button>
            <button type="button" class="btn ghost" onclick={onclose}>Cancel</button>
          {/if}
          <button type="submit" class="btn primary" disabled={busy}>{busy ? "Unlocking…" : "Unlock"}</button>
        </div>
      </form>

    {:else if currentMode === "reset"}
      <form onsubmit={handleReset}>
        <div class="warn-box" role="note">
          Resetting the vault permanently deletes all stored passwords. This action cannot be undone.
        </div>
        {#if error}<div class="note bad" role="alert">{error}</div>{/if}
        <label>
          <span>Type RESET to confirm</span>
          <!-- svelte-ignore a11y_autofocus -->
          <input type="text" bind:value={resetConfirmInput} autofocus placeholder="RESET" aria-label="Type RESET to confirm" />
        </label>
        <div class="actions">
          <button type="button" class="btn ghost" onclick={() => { error = ""; currentMode = vault.exists && !vault.unlocked ? "unlock" : "manage"; }}>Back</button>
          <button type="submit" class="btn danger" disabled={resetConfirmInput !== "RESET" || busy}>{busy ? "Resetting…" : "Reset vault"}</button>
        </div>
      </form>

    {:else if currentMode === "manage"}
      <div class="manage-body">
        {#if notice}<div class="note good">{notice}</div>{/if}
        {#if error}<div class="note bad" role="alert">{error}</div>{/if}
        <div class="status-row">
          <div class="status-info">
            <span class="dot ok"></span>
            <div>
              <strong>Vault is unlocked</strong>
              <small>{vault.remembered.length} password{vault.remembered.length === 1 ? "" : "s"} saved</small>
            </div>
          </div>
          <button type="button" class="btn ghost" onclick={handleLock} disabled={busy}>Lock now</button>
        </div>

        <div class="section-divider"></div>

        <div class="collapsible">
          <button type="button" class="toggle-btn" onclick={() => { showChangeMaster = !showChangeMaster; showReset = false; error = ""; notice = ""; }}>
            <span>Change master password</span>
            <span>{showChangeMaster ? "▲" : "▼"}</span>
          </button>
          {#if showChangeMaster}
            <form onsubmit={handleChangeMaster} class="nested-form">
              <label>
                <span>Current master password</span>
                <input type="password" bind:value={oldMasterInput} required placeholder="Current password" aria-label="Current master password" autocomplete="current-password" />
              </label>
              <label>
                <span>New master password</span>
                <input type="password" bind:value={masterInput} required placeholder="New master password" aria-label="New master password" autocomplete="new-password" />
              </label>
              <label>
                <span>Confirm new master password</span>
                <input type="password" bind:value={confirmInput} required placeholder="Repeat new master password" aria-label="Confirm new master password" autocomplete="new-password" />
              </label>
              <div class="actions">
                <button type="submit" class="btn primary" disabled={busy}>{busy ? "Saving…" : "Update password"}</button>
              </div>
            </form>
          {/if}
        </div>

        <div class="collapsible">
          <button type="button" class="toggle-btn danger-text" onclick={() => { showReset = !showReset; showChangeMaster = false; error = ""; notice = ""; }}>
            <span>Reset vault…</span>
            <span>{showReset ? "▲" : "▼"}</span>
          </button>
          {#if showReset}
            <form onsubmit={handleReset} class="nested-form">
              <p class="warn-text">Permanently deletes all saved passwords. Type <strong>RESET</strong> to confirm:</p>
              <input type="text" bind:value={resetConfirmInput} placeholder="RESET" aria-label="Type RESET to confirm" />
              <div class="actions">
                <button type="submit" class="btn danger" disabled={resetConfirmInput !== "RESET" || busy}>{busy ? "Resetting…" : "Reset vault"}</button>
              </div>
            </form>
          {/if}
        </div>

        <div class="actions footer-actions">
          <button type="button" class="btn ghost" onclick={onclose}>Close</button>
        </div>
      </div>
    {/if}
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; background: var(--scrim); z-index: 30; display: flex; align-items: center; justify-content: center; padding: 16px; }
  .panel { width: min(420px, 100%); background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 24px 70px var(--shadow); overflow: hidden; }
  header { display: flex; align-items: center; justify-content: space-between; height: 48px; padding: 0 14px 0 18px; border-bottom: 1px solid var(--line2); }
  h2 { margin: 0; font-size: 14px; font-weight: 600; color: var(--strong); }
  .x { width: 30px; height: 30px; border: 0; background: none; border-radius: 4px; color: var(--muted); font-size: 18px; cursor: pointer; }
  .x:hover { background: var(--line); color: var(--strong); }
  form { padding: 16px 18px 18px; display: grid; gap: 12px; }
  .manage-body { padding: 16px 18px 18px; display: grid; gap: 14px; }
  .desc { margin: 0; font-size: 12.5px; color: var(--text2); line-height: 1.4; }
  .desc strong { color: var(--strong); }
  label { display: grid; gap: 5px; font-size: 12px; color: var(--text2); }
  input { height: 34px; padding: 0 10px; background: var(--field); border: 1px solid var(--fieldline); border-radius: 4px; font-size: 13px; color: var(--text); }
  input:focus { outline: none; border-color: var(--accent); }
  .warn-box { padding: 10px 12px; font-size: 12px; border-radius: 4px; color: var(--warn); border: 1px solid var(--line); background: var(--bar); line-height: 1.4; }
  .warn-text { margin: 0 0 8px; font-size: 12px; color: var(--err-text, var(--warn)); line-height: 1.4; }
  .note { padding: 8px 12px; font-size: 12px; border-radius: 4px; line-height: 1.4; }
  .note.bad { color: var(--err-text); border: 1px solid var(--err-edge); background: var(--err-bg); }
  .note.good { color: var(--ok); border: 1px solid var(--ok-edge); background: var(--ok-bg); }
  .actions { display: flex; align-items: center; justify-content: flex-end; gap: 8px; margin-top: 4px; }
  .footer-actions { border-top: 1px solid var(--line2); padding-top: 12px; margin-top: 6px; }
  .btn { height: 32px; padding: 0 14px; border-radius: 4px; font-size: 12.5px; font-weight: 500; cursor: pointer; display: inline-flex; align-items: center; justify-content: center; }
  .btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .btn.primary { background: var(--accent); border: 1px solid var(--accent); color: var(--on-accent); }
  .btn.primary:hover:not(:disabled) { filter: brightness(1.08); }
  .btn.danger { background: var(--err); border: 1px solid var(--err); color: var(--on-err); }
  .btn.danger:hover:not(:disabled) { filter: brightness(1.08); }
  .btn.ghost { background: none; border: 1px solid var(--line); color: var(--text2); }
  .btn.ghost:hover:not(:disabled) { background: var(--hover); color: var(--strong); }
  .btn.text-btn { background: none; border: 0; color: var(--accent-text); padding: 0 6px; font-size: 12px; }
  .btn.text-btn:hover { text-decoration: underline; }
  .status-row { display: flex; align-items: center; justify-content: space-between; }
  .status-info { display: flex; align-items: center; gap: 10px; }
  .status-info strong { display: block; font-size: 13px; color: var(--strong); }
  .status-info small { color: var(--muted); font-size: 11.5px; }
  .dot { width: 8px; height: 8px; border-radius: 50%; }
  .dot.ok { background: var(--ok); }
  .section-divider { height: 1px; background: var(--line2); }
  .collapsible { border: 1px solid var(--line2); border-radius: 4px; overflow: hidden; }
  .toggle-btn { width: 100%; display: flex; align-items: center; justify-content: space-between; padding: 9px 12px; background: var(--bar); border: 0; font-size: 12px; color: var(--text2); cursor: pointer; }
  .toggle-btn:hover { background: var(--hover); color: var(--strong); }
  .danger-text { color: var(--err-text, var(--warn)); }
  .nested-form { padding: 12px; border-top: 1px solid var(--line2); background: var(--float); }
</style>