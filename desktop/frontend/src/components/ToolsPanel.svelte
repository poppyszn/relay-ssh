<script lang="ts">
  import { ToolsService, type HealthCheck, type ImportReport, type KeyInfo } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import EmptyState from "./EmptyState.svelte";
  import Modal from "./Modal.svelte";

  type Tab = "health" | "backup" | "import" | "keys";
  let { initial = "health", onclose, onchanged, ongenerate }: { initial?: Tab; onclose: () => void; onchanged: () => void; ongenerate: () => void } = $props();

  // svelte-ignore state_referenced_locally
  let tab = $state<Tab>(initial);
  const tabs: { id: Tab; label: string }[] = [
    { id: "health", label: "Health" },
    { id: "backup", label: "Backup & restore" },
    { id: "import", label: "Import" },
    { id: "keys", label: "Keys" },
  ];

  const msg = (e: unknown) => (e as Error)?.message ?? String(e);

  // Health
  let checks = $state<HealthCheck[]>([]);
  let fixErrors = $state<string[]>([]);
  let busy = $state(false);
  const problems = $derived(checks.filter((c) => c.status !== "ok").length);
  async function recheck() { busy = true; try { checks = (await ToolsService.Health()) ?? []; } catch (e) { fixErrors = [msg(e)]; } finally { busy = false; } }
  async function fix() {
    busy = true;
    try {
      const r = await ToolsService.FixHealth();
      checks = r.checks ?? [];
      fixErrors = r.errors ?? [];
      onchanged();
    } catch (e) { fixErrors = [msg(e)]; } finally { busy = false; }
  }

  // Backup and restore
  let pass = $state("");
  let pass2 = $state("");
  let backupNote = $state<{ ok: boolean; text: string } | null>(null);
  async function backup() {
    backupNote = null;
    if (pass.length < 1) return (backupNote = { ok: false, text: "Enter a passphrase." });
    if (pass !== pass2) return (backupNote = { ok: false, text: "The passphrases do not match." });
    try {
      const path = await ToolsService.PickBackupPath();
      if (!path) return;
      await ToolsService.Backup(path, pass);
      backupNote = { ok: true, text: `Backup saved to ${path}` };
      pass = pass2 = "";
    } catch (e) { backupNote = { ok: false, text: msg(e) }; }
  }
  let restorePath = $state("");
  let restorePass = $state("");
  let needsPass = $state(false);
  let confirmRestore = $state(false);
  let restoreNote = $state<{ ok: boolean; text: string } | null>(null);
  async function chooseRestore() {
    restoreNote = null; confirmRestore = false;
    try {
      const path = await ToolsService.PickRestorePath();
      if (!path) return;
      restorePath = path;
      needsPass = await ToolsService.NeedsPassphrase(path);
    } catch (e) { restoreNote = { ok: false, text: msg(e) }; }
  }
  async function restore() {
    confirmRestore = false;
    try {
      await ToolsService.Restore(restorePath, restorePass);
      restoreNote = { ok: true, text: "Backup restored. Your connections and keys are back." };
      restorePass = "";
      onchanged();
    } catch (e) { restoreNote = { ok: false, text: msg(e) }; }
  }

  // Tabs: arrow keys move between them (the list is vertical).
  let navEl: HTMLElement;
  function onNavKey(e: KeyboardEvent) {
    const i = tabs.findIndex((t) => t.id === tab);
    const next = e.key === "ArrowDown" ? (i + 1) % tabs.length : e.key === "ArrowUp" ? (i - 1 + tabs.length) % tabs.length : e.key === "Home" ? 0 : e.key === "End" ? tabs.length - 1 : -1;
    if (next < 0) return;
    e.preventDefault();
    tab = tabs[next].id;
    navEl.querySelectorAll<HTMLElement>('[role="tab"]')[next]?.focus();
  }

  // Import
  let report = $state<ImportReport | null>(null);
  const added = $derived(report?.added ?? []);
  const skipped = $derived(report?.skipped ?? []);
  let importError = $state("");
  async function doImport() {
    importError = ""; report = null; busy = true;
    try { report = await ToolsService.ImportSSHConfig(); if ((report.added ?? []).length) onchanged(); } catch (e) { importError = msg(e); } finally { busy = false; }
  }

  // Keys
  let keyList = $state<KeyInfo[]>([]);
  let keyError = $state("");
  async function loadKeys() { try { keyList = (await ToolsService.Keys()) ?? []; } catch (e) { keyError = msg(e); } }
  const mac = navigator.userAgent.includes("Mac");
  /** The key whose passphrase is being asked for, and whether it is being adopted or just remembered. */
  let ask = $state<{ path: string; name: string; mode: "adopt" | "remember" } | null>(null);
  let keyPass = $state("");
  async function adopt(path: string) {
    keyError = "";
    try {
      if (await ToolsService.KeyNeedsPassphrase(path)) {
        ask = { path, name: path.split("/").pop() ?? path, mode: "adopt" };
        keyPass = "";
        return;
      }
      await ToolsService.AdoptKey(path, "");
      onchanged();
      await loadKeys();
    } catch (e) { keyError = msg(e); }
  }
  async function submitPass() {
    if (!ask) return;
    keyError = "";
    busy = true;
    try {
      if (ask.mode === "adopt") await ToolsService.AdoptKey(ask.path, keyPass);
      else await ToolsService.RememberPassphrase(ask.name, keyPass);
      keyPass = "";
      ask = null;
      onchanged();
      await loadKeys();
    } catch (e) { keyError = msg(e); } finally { busy = false; }
  }
  function cancelPass() { ask = null; keyPass = ""; keyError = ""; }
  async function adoptOther() {
    try { const p = await ToolsService.PickKeyToAdopt(); if (p) await adopt(p); } catch (e) { keyError = msg(e); }
  }

  $effect(() => { if (tab === "health") recheck(); else if (tab === "keys") loadKeys(); });
</script>

<Modal title="Tools" size="lg" tall flush {onclose}>
  <div class="layout">
    <div class="nav" role="tablist" aria-label="Tools" aria-orientation="vertical" tabindex="-1" bind:this={navEl} onkeydown={onNavKey}>
      {#each tabs as t}
        <button type="button" role="tab" id="tools-tab-{t.id}" aria-selected={tab === t.id} aria-controls="tools-panel" tabindex={tab === t.id ? 0 : -1} data-autofocus={tab === t.id ? "" : undefined} class="tab" class:row-hl={tab === t.id} class:row-hover={tab !== t.id} onclick={() => (tab = t.id)}>{t.label}</button>
      {/each}
    </div>
    <div class="content" role="tabpanel" id="tools-panel" aria-labelledby="tools-tab-{tab}" tabindex="-1">
      {#if tab === "health"}
        <div class="head">
          <div><h3>Health</h3><p class="hint">Checks that your connections, the generated SSH config and your keys agree.</p></div>
          <div class="actions">
            <button type="button" class="btn" onclick={recheck} disabled={busy}>Re-check</button>
            {#if problems}<button type="button" class="btn primary" onclick={fix} disabled={busy}>Fix {problems} issue{problems === 1 ? "" : "s"}</button>{/if}
          </div>
        </div>
        {#if checks.length}
          <ul class="rows">
            {#each checks as c}
              <li><span class="dot" class:ok={c.status === "ok"} class:warn={c.status === "warn"} class:err={c.status === "fail"}></span><b>{c.name}</b><span class="detail">{c.detail}</span></li>
            {/each}
          </ul>
        {/if}
        {#each fixErrors.filter(Boolean) as e}<div class="notice bad" role="alert">{e}</div>{/each}
        {#if checks.length && !problems}<div class="notice good">Everything looks healthy.</div>{/if}
      {:else if tab === "backup"}
        <section>
          <h3>Back up</h3>
          <p class="hint">Saves your connections and adopted keys into one file, encrypted with a passphrase. Relay never stores the passphrase: without it the file cannot be opened.</p>
          <div class="pair">
            <label class="field"><span>Passphrase</span><input type="password" bind:value={pass} autocomplete="off" /></label>
            <label class="field"><span>Confirm passphrase</span><input type="password" bind:value={pass2} autocomplete="off" /></label>
          </div>
          <div class="actions"><button type="button" class="btn primary" onclick={backup}>Choose location and back up</button></div>
          {#if backupNote?.text}<div class="notice" class:good={backupNote.ok} class:bad={!backupNote.ok} role={backupNote.ok ? "status" : "alert"}>{backupNote.text}</div>{/if}
        </section>

        <section>
          <h3>Restore</h3>
          <p class="hint">Replaces your current connections and adopted keys with the contents of a backup.</p>
          <div class="actions"><button type="button" class="btn" onclick={chooseRestore}>Choose backup…</button><span class="path mono">{restorePath || "No file chosen"}</span></div>
          {#if restorePath}
            {#if !needsPass}<div class="notice warn" role="alert">This backup is not encrypted, so nothing proves it is yours. Only restore a file you made or trust: it replaces your connections and keys. Connections that could not be written safely into your ssh config are refused.</div>{/if}
            {#if needsPass}<label class="field one"><span>Passphrase</span><input type="password" bind:value={restorePass} autocomplete="off" /></label>{/if}
            {#if confirmRestore}
              <div class="notice warn" role="alert">This replaces your current connections and adopted keys.</div>
              <div class="actions">
                <button type="button" class="btn" onclick={() => (confirmRestore = false)}>Cancel</button>
                <button type="button" class="btn danger" onclick={restore}>Replace</button>
              </div>
            {:else}
              <div class="actions"><button type="button" class="btn" onclick={() => (confirmRestore = true)}>Restore</button></div>
            {/if}
          {/if}
          {#if restoreNote?.text}<div class="notice" class:good={restoreNote.ok} class:bad={!restoreNote.ok} role={restoreNote.ok ? "status" : "alert"}>{restoreNote.text}</div>{/if}
        </section>
      {:else if tab === "import"}
        <h3>Import from ~/.ssh/config</h3>
        <p class="hint">Adds a connection for every concrete Host entry in your SSH config that Relay doesn't know yet. Entries without a HostName or User get ssh's own defaults (the alias and your login). Wildcard hosts are ignored, and nothing in your SSH config is changed.</p>
        <div class="actions"><button type="button" class="btn primary" onclick={doImport} disabled={busy}>Import</button></div>
        {#if importError}<div class="notice bad" role="alert">{importError}</div>{/if}
        {#if report}
          <div class="notice" class:good={added.length > 0} role="status">{added.length ? `Imported ${added.length}: ${added.join(", ")}` : "No new connections found."}</div>
          {#each skipped as s}<div class="notice bad">Skipped {s}</div>{/each}
        {/if}
      {:else if tab === "keys"}
        <div class="head">
          <div><h3>Keys</h3><p class="hint">Adopted keys are copied into Relay's vault and included in backups. Other keys stay where they are and are not backed up. For a key with a passphrase, adopting also asks for it once so ssh does not ask every time.</p></div>
          <div class="actions">
            <button type="button" class="btn" onclick={adoptOther}>Adopt a key…</button>
            <button type="button" class="btn" onclick={ongenerate}>Generate a key…</button>
          </div>
        </div>
        {#if keyError}<div class="notice bad" role="alert">{keyError}</div>{/if}
        {#if keyList.length}
          <ul class="rows">
            {#each keyList as k}
              <li class="key">
                <div class="info">
                  <div class="kname"><b class="mono">{k.name}</b>{#if k.adopted}<span class="chip ok">In backups</span>{:else}<span class="chip">Not backed up</span>{/if}{#if k.encrypted}<span class="chip">Passphrase</span>{:else}<span class="chip warn" title="Anyone who can read this file can use the key. Add a passphrase with: ssh-keygen -p -f {k.path}">No passphrase</span>{/if}</div>
                  <div class="kmeta mono">{k.path}</div>
                  <div class="kuse">{k.usedBy?.length ? "Used by " + k.usedBy.join(", ") : "Not used by any connection"}</div>
                </div>
                {#if ask && ask.path === k.path}
                  <form class="pass" onsubmit={(e) => { e.preventDefault(); submitPass(); }}>
                    <label class="field"><span>Passphrase for {k.name}</span><input type="password" bind:value={keyPass} autocomplete="off" data-autofocus /></label>
                    <p class="hint">{mac ? "ssh-add stores it in your macOS Keychain and loads the key into the ssh-agent, so ssh stops asking. Relay does not keep it." : "The key is loaded into your ssh-agent for this login session, so ssh stops asking. Relay does not keep the passphrase."}</p>
                    <div class="actions">
                      <button type="button" class="btn sm" onclick={cancelPass}>Cancel</button>
                      <button type="submit" class="btn sm primary" disabled={busy || !keyPass}>{ask.mode === "adopt" ? "Adopt and remember" : "Remember"}</button>
                    </div>
                  </form>
                {:else if !k.adopted}<button type="button" class="btn sm" onclick={() => adopt(k.path)}>Adopt</button>
                {:else if k.encrypted}<button type="button" class="btn sm" title="Load this key into the ssh-agent{mac ? ' and the Keychain' : ''} so ssh stops asking for its passphrase" onclick={() => { ask = { path: k.path, name: k.name, mode: "remember" }; keyPass = ""; keyError = ""; }}>Remember passphrase</button>{/if}
              </li>
            {/each}
          </ul>
        {:else}
          <EmptyState icon="key" title="No private keys found" text="Relay looks for keys in ~/.ssh. Generate one or adopt a key from elsewhere." />
        {/if}
      {/if}
    </div>
  </div>
</Modal>

<style>
  .layout { flex: 1; min-height: 0; display: flex; }
  .nav { width: 196px; flex: none; display: grid; align-content: start; gap: 2px; padding: var(--s3) var(--s2); background: var(--bar); border-right: 1px solid var(--line); }
  .tab { height: var(--h-md); padding: 0 var(--s3); text-align: left; background: transparent; border: 0; border-radius: var(--r-sm); color: var(--text2); font-size: var(--t-md); }
  .tab[aria-selected="true"] { font-weight: 500; }
  .content { flex: 1; min-width: 0; overflow: auto; padding: var(--s5); display: grid; align-content: start; gap: var(--s4); outline: none; }
  h3 { margin: 0 0 var(--s1); font-size: var(--t-lg); font-weight: 600; color: var(--strong); }
  p { margin: 0; }
  section { display: grid; gap: var(--s4); }
  section + section { padding-top: var(--s5); border-top: 1px solid var(--line); }
  .head { display: flex; justify-content: space-between; align-items: flex-start; gap: var(--s4); }
  .actions { display: flex; align-items: center; gap: var(--s2); flex-wrap: wrap; }
  .head .actions { flex: none; }
  .pair { display: grid; grid-template-columns: 1fr 1fr; gap: var(--s3); }
  .field.one { max-width: 50%; }
  .path { font-size: var(--t-sm); color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; min-width: 0; }
  .rows { list-style: none; margin: 0; padding: 0; border: 1px solid var(--line); border-radius: var(--r-sm); }
  .rows li { display: flex; align-items: center; gap: var(--s3); min-height: 36px; padding: var(--s2) var(--s3); border-top: 1px solid var(--line); }
  .rows li:first-child { border-top: 0; }
  .rows b { font-weight: 500; min-width: 140px; }
  .detail { color: var(--muted); }
  .rows li.key { display: grid; grid-template-columns: 1fr auto; align-items: center; }
  .info { display: grid; gap: 2px; min-width: 0; }
  .kname { display: flex; align-items: center; flex-wrap: wrap; gap: var(--s2); }
  .kname b { min-width: 0; }
  .kmeta { font-size: var(--t-sm); color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .kuse { font-size: var(--t-sm); color: var(--text2); }
  .pass { grid-column: 1 / -1; display: grid; gap: var(--s2); padding-top: var(--s3); border-top: 1px solid var(--line); }
  .pass .field { max-width: 320px; }
</style>
