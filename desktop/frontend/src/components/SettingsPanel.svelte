<script lang="ts">
  import { onMount } from "svelte";
  import { VaultService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { DENSITY, FONTS, SYSTEM, THEMES, settings } from "../state/settings.svelte";
  import { vault } from "../state/vault.svelte";
  import Modal from "./Modal.svelte";
  import VaultPanel from "./VaultPanel.svelte";

  let { onclose, section }: { onclose: () => void; section?: "vault" } = $props();

  const setSize = (e: Event & { currentTarget: HTMLInputElement }) => {
    const n = Math.round(Number(e.currentTarget.value));
    settings.fontSize = Math.min(24, Math.max(10, Number.isFinite(n) && n > 0 ? n : settings.fontSize));
    e.currentTarget.value = String(settings.fontSize);
  };

  // Vault
  let vaultEl: HTMLElement;
  /** The small create / unlock prompt, stacked over Settings. */
  let prompt = $state<"create" | "unlock" | null>(null);
  let form = $state<"master" | "reset" | null>(null);
  let oldInput = $state("");
  let newInput = $state("");
  let confirmInput = $state("");
  let resetInput = $state("");
  let busy = $state(false);
  let error = $state("");
  let done = $state("");

  const status = $derived(vault.exists ? (vault.unlocked ? { text: "Unlocked", tone: "ok" } : { text: "Locked", tone: "warn" }) : { text: "Not set up", tone: "" });
  const saved = $derived(vault.remembered.length);

  onMount(() => {
    if (section !== "vault") return;
    vaultEl.scrollIntoView?.({ block: "start" });
    vaultEl.querySelector<HTMLElement>("select, input, button")?.focus({ preventScroll: true });
  });

  function open(f: "master" | "reset") {
    form = form === f ? null : f;
    error = done = "";
    oldInput = newInput = confirmInput = resetInput = "";
  }

  async function lockNow() {
    busy = true;
    error = done = "";
    try {
      await vault.lock();
    } catch (err) {
      error = (err as Error)?.message ?? String(err);
    } finally {
      busy = false;
    }
  }

  async function changeMaster(e: Event) {
    e.preventDefault();
    error = done = "";
    if (!oldInput) return (error = "Enter your current master password.");
    if (!newInput) return (error = "Enter a new master password.");
    if (newInput !== confirmInput) return (error = "The new passwords do not match. Type the same password in both fields.");
    const [oldP, newP] = [oldInput, newInput];
    oldInput = newInput = confirmInput = "";
    busy = true;
    try {
      await VaultService.ChangeMaster(oldP, newP);
      done = "Master password changed.";
      form = null;
    } catch {
      error = "The current master password is wrong, or the vault could not be updated.";
    } finally {
      busy = false;
    }
  }

  async function resetVault(e: Event) {
    e.preventDefault();
    error = done = "";
    if (resetInput !== "RESET") return (error = "Type RESET in capital letters to confirm.");
    busy = true;
    try {
      await vault.reset("RESET");
      form = null;
      done = "Vault reset. Create a new vault to save passwords again.";
    } catch (err) {
      error = (err as Error)?.message ?? String(err);
    } finally {
      resetInput = "";
      busy = false;
    }
  }
</script>

<Modal title="Settings" size="md" {onclose}>
  <section aria-labelledby="set-appearance">
    <h3 class="label" id="set-appearance">Appearance</h3>
    <div class="themes" role="radiogroup" aria-label="Theme">
      {#each Object.entries(THEMES) as [id, t]}
        <button type="button" role="radio" class="theme" class:row-hl={settings.theme === id} aria-checked={settings.theme === id} onclick={() => (settings.theme = id)}>
          <span class="swatch" style="background: linear-gradient({t.vars.term} 70%, {t.vars.rail} 70%)"><i style="background: {t.vars.accent}"></i></span>
          <span>{t.name}</span>
        </button>
      {/each}
      <button type="button" role="radio" class="theme" class:row-hl={settings.theme === SYSTEM} aria-checked={settings.theme === SYSTEM} onclick={() => (settings.theme = SYSTEM)} title="Follow the macOS light or dark appearance">
        <span class="swatch" style="background: linear-gradient(90deg, {THEMES.midnight.vars.term} 50%, {THEMES.daylight.vars.term} 50%)"><i style="background: {THEMES.daylight.vars.accent}"></i></span>
        <span>System</span>
      </button>
    </div>
    <div class="pair">
      <label class="field">
        <span>Terminal font</span>
        <select bind:value={settings.font}>
          {#each Object.entries(FONTS) as [id, f]}<option value={id}>{f.name}</option>{/each}
        </select>
      </label>
      <label class="field">
        <span>Size (px)</span>
        <input type="number" min="10" max="24" step="1" value={settings.fontSize} onchange={setSize} />
      </label>
    </div>
  </section>

  <section aria-labelledby="set-terminal">
    <h3 class="label" id="set-terminal">Terminal</h3>
    <div class="field">
      <span class="field-label" id="set-spacing">Line spacing</span>
      <div class="seg" role="group" aria-labelledby="set-spacing">
        {#each Object.entries(DENSITY) as [id, d]}
          <button type="button" aria-pressed={settings.density === id} onclick={() => (settings.density = id)}>{d.name}</button>
        {/each}
      </div>
    </div>
    <div class="opt">
      <div class="txt"><span id="set-copy">Copy on select</span><span class="hint">Selecting text copies it to the clipboard.</span></div>
      <button type="button" class="switch" role="switch" aria-checked={settings.copyOnSelect} aria-labelledby="set-copy" onclick={() => (settings.copyOnSelect = !settings.copyOnSelect)}></button>
    </div>
    <div class="opt">
      <div class="txt"><span id="set-blink">Blinking cursor</span></div>
      <button type="button" class="switch" role="switch" aria-checked={settings.cursorBlink} aria-labelledby="set-blink" onclick={() => (settings.cursorBlink = !settings.cursorBlink)}></button>
    </div>
    <div class="opt">
      <div class="txt"><span id="set-blocks">Automatic command blocks over SSH</span><span class="hint">Types a short setup line into the remote bash or zsh after login. Each connection can override this. When off, you are offered it instead.</span></div>
      <button type="button" class="switch" role="switch" aria-checked={settings.autoBlocks} aria-labelledby="set-blocks" onclick={() => (settings.autoBlocks = !settings.autoBlocks)}></button>
    </div>
  </section>

  <section id="settings-rdp" aria-labelledby="set-rdp">
    <h3 class="label" id="set-rdp">Remote desktop</h3>
    <div class="opt">
      <div class="txt"><span id="set-rdpclip">Share the clipboard with remote desktops</span><span class="hint">While a remote desktop is focused, what you copy on this Mac is sent to it, and what it copies replaces your clipboard. Turn this off for servers you do not fully trust. Each remote desktop tab also has its own switch.</span></div>
      <button type="button" class="switch" role="switch" aria-checked={settings.rdpClipboard} aria-labelledby="set-rdpclip" onclick={() => (settings.rdpClipboard = !settings.rdpClipboard)}></button>
    </div>
  </section>

  <section id="settings-vault" aria-labelledby="set-vault" bind:this={vaultEl}>
    <div class="head">
      <h3 class="label" id="set-vault">Vault</h3>
      <span class="chip {status.tone}">{status.text}</span>
      {#if vault.exists && saved}<span class="hint">{saved} password{saved === 1 ? "" : "s"} saved</span>{/if}
    </div>
    <p class="hint lead">Saved connection passwords are encrypted with your master password and filled in when ssh asks.</p>

    <label class="field">
      <span>Lock the vault after</span>
      <select bind:value={settings.vaultTimeout}>
        <option value={5}>5 minutes of inactivity</option>
        <option value={15}>15 minutes of inactivity</option>
        <option value={60}>1 hour of inactivity</option>
        <option value={0}>When the app quits</option>
      </select>
    </label>

    {#if done}<div class="notice good" role="status">{done}</div>{/if}
    {#if error && !form}<div class="notice bad" role="alert">{error}</div>{/if}

    <div class="actions">
      {#if !vault.exists}
        <button type="button" class="btn primary" onclick={() => (prompt = "create")}>Create vault</button>
      {:else}
        {#if vault.unlocked}
          <button type="button" class="btn" onclick={lockNow} disabled={busy}>Lock now</button>
          <button type="button" class="btn" aria-expanded={form === "master"} onclick={() => open("master")}>Change master password</button>
        {:else}
          <button type="button" class="btn primary" onclick={() => (prompt = "unlock")}>Unlock vault</button>
        {/if}
        <button type="button" class="btn ghost danger" aria-expanded={form === "reset"} onclick={() => open("reset")}>Reset vault…</button>
      {/if}
    </div>

    {#if form === "master"}
      <form class="inline" onsubmit={changeMaster}>
        {#if error}<div class="notice bad" role="alert">{error}</div>{/if}
        <label class="field"><span>Current master password</span><input type="password" bind:value={oldInput} autocomplete="current-password" /></label>
        <div class="pair">
          <label class="field"><span>New master password</span><input type="password" bind:value={newInput} autocomplete="new-password" /></label>
          <label class="field"><span>Confirm new password</span><input type="password" bind:value={confirmInput} autocomplete="new-password" /></label>
        </div>
        <div class="actions end">
          <button type="button" class="btn" onclick={() => open("master")}>Cancel</button>
          <button type="submit" class="btn primary" disabled={busy}>{busy ? "Saving…" : "Change password"}</button>
        </div>
      </form>
    {:else if form === "reset"}
      <form class="inline" onsubmit={resetVault}>
        <div class="notice warn" role="note">If you forget the master password the only way back in is to reset the vault, which deletes the saved passwords.</div>
        {#if error}<div class="notice bad" role="alert">{error}</div>{/if}
        <label class="field"><span>Type RESET to confirm</span><input type="text" bind:value={resetInput} placeholder="RESET" autocomplete="off" /></label>
        <div class="actions end">
          <button type="button" class="btn" onclick={() => open("reset")}>Cancel</button>
          <button type="submit" class="btn danger" disabled={resetInput !== "RESET" || busy}>{busy ? "Resetting…" : "Reset vault"}</button>
        </div>
      </form>
    {/if}
  </section>
</Modal>

{#if prompt}<VaultPanel mode={prompt} onclose={() => (prompt = null)} />{/if}

<style>
  section { display: grid; gap: var(--s4); }
  section + section { margin-top: var(--s5); padding-top: var(--s5); border-top: 1px solid var(--line); }
  h3 { margin: 0; }
  .themes { display: grid; grid-template-columns: repeat(auto-fill, minmax(112px, 1fr)); gap: var(--s2); }
  .theme { display: grid; gap: var(--s2); padding: var(--s2); text-align: left; background: var(--field); border: 1px solid var(--fieldline); border-radius: var(--r-sm); color: var(--text2); font-size: var(--t-sm); transition: border-color var(--dur-1) var(--ease); }
  .theme:hover { border-color: var(--floatline); }
  .theme[aria-checked="true"] { border-color: var(--accent); }
  .swatch { display: block; position: relative; height: 40px; border-radius: var(--r-xs); overflow: hidden; }
  .swatch i { position: absolute; left: var(--s2); top: var(--s2); width: 24px; height: 3px; }
  .pair { display: grid; grid-template-columns: 1fr 1fr; gap: var(--s3); }
  .seg { justify-self: start; }
  .opt { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--s4); }
  .opt .txt { display: grid; gap: 2px; }
  .opt .switch { margin-top: 1px; }
  .head { display: flex; align-items: center; gap: var(--s2); }
  .head .hint { margin-left: var(--s1); }
  .lead { margin: 0; }
  .actions { display: flex; flex-wrap: wrap; align-items: center; gap: var(--s2); }
  .actions.end { justify-content: flex-end; }
  .inline { display: grid; gap: var(--s3); padding: var(--s4); background: var(--band); border: 1px solid var(--line); border-radius: var(--r-md); }
</style>
