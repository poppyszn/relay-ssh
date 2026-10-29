<script lang="ts">
  import { ToolsService, type HealthCheck, type ImportReport, type KeyInfo } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";

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
    if (!confirmRestore) return (confirmRestore = true);
    confirmRestore = false;
    try {
      await ToolsService.Restore(restorePath, restorePass);
      restoreNote = { ok: true, text: "Backup restored. Your connections and keys are back." };
      restorePass = "";
      onchanged();
    } catch (e) { restoreNote = { ok: false, text: msg(e) }; }
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
  async function adopt(path: string) {
    keyError = "";
    try { await ToolsService.AdoptKey(path); onchanged(); await loadKeys(); } catch (e) { keyError = msg(e); }
  }
  async function adoptOther() {
    try { const p = await ToolsService.PickKeyToAdopt(); if (p) await adopt(p); } catch (e) { keyError = msg(e); }
  }

  $effect(() => { if (tab === "health") recheck(); else if (tab === "keys") loadKeys(); });
</script>

<div class="scrim" role="presentation" onclick={onclose} onkeydown={(e) => e.key === "Escape" && onclose()}>
  <div class="panel" role="dialog" aria-label="Tools" tabindex="-1" onclick={(e) => e.stopPropagation()}>
    <header><h2>Tools</h2><button class="x" aria-label="Close" onclick={onclose}>&times;</button></header>
    <div class="body">
      <nav aria-label="Tools">
        {#each tabs as t}<button class:on={tab === t.id} onclick={() => (tab = t.id)}>{t.label}</button>{/each}
      </nav>
      <div class="content">
        {#if tab === "health"}
          <div class="head"><div><h3>Health</h3><p>Checks that your connections, the generated SSH config and your keys agree.</p></div>
            <div class="actions"><button class="btn" onclick={recheck} disabled={busy}>Re-check</button>{#if problems}<button class="btn primary" onclick={fix} disabled={busy}>Fix {problems} issue{problems === 1 ? "" : "s"}</button>{/if}</div></div>
          <ul class="checks">
            {#each checks as c}
              <li><span class="dot" class:ok={c.status === "ok"} class:warn={c.status === "warn"} class:err={c.status === "fail"}></span><b>{c.name}</b><span class="detail">{c.detail}</span></li>
            {/each}
          </ul>
          {#each fixErrors as e}<p class="note bad">{e}</p>{/each}
          {#if checks.length && !problems}<p class="note good">Everything looks healthy.</p>{/if}
        {:else if tab === "backup"}
          <h3>Back up</h3>
          <p>Saves your connections and adopted keys into one file, encrypted with a passphrase. Relay never stores the passphrase: without it the file cannot be opened.</p>
          <div class="fields"><label>Passphrase<input type="password" bind:value={pass} autocomplete="off" /></label><label>Confirm<input type="password" bind:value={pass2} autocomplete="off" /></label></div>
          <div class="actions"><button class="btn primary" onclick={backup}>Choose location and back up</button></div>
          {#if backupNote}<p class="note" class:good={backupNote.ok} class:bad={!backupNote.ok}>{backupNote.text}</p>{/if}

          <h3 class="gap">Restore</h3>
          <p>Replaces your current connections and adopted keys with the contents of a backup.</p>
          <div class="actions"><button class="btn" onclick={chooseRestore}>Choose backup…</button><span class="path mono">{restorePath || "No file chosen"}</span></div>
          {#if restorePath}
            {#if needsPass}<div class="fields one"><label>Passphrase<input type="password" bind:value={restorePass} autocomplete="off" /></label></div>{/if}
            <div class="actions"><button class="btn" class:danger-fill={confirmRestore} onclick={restore}>{confirmRestore ? "Click again to replace current data" : "Restore"}</button></div>
          {/if}
          {#if restoreNote}<p class="note" class:good={restoreNote.ok} class:bad={!restoreNote.ok}>{restoreNote.text}</p>{/if}
        {:else if tab === "import"}
          <h3>Import from ~/.ssh/config</h3>
          <p>Adds a connection for every concrete Host entry in your SSH config that Relay doesn't know yet. Entries without a HostName or User get ssh's own defaults (the alias and your login). Wildcard hosts are ignored, and nothing in your SSH config is changed.</p>
          <div class="actions"><button class="btn primary" onclick={doImport} disabled={busy}>Import</button></div>
          {#if importError}<p class="note bad">{importError}</p>{/if}
          {#if report}
            <p class="note" class:good={added.length > 0}>{added.length ? `Imported ${added.length}: ${added.join(", ")}` : "No new connections found."}</p>
            {#each skipped as s}<p class="note bad">Skipped {s}</p>{/each}
          {/if}
        {:else}
          <div class="head"><div><h3>Keys</h3><p>Adopted keys are copied into Relay's vault and included in backups. Other keys stay where they are and are not backed up.</p></div>
            <div class="actions"><button class="btn" onclick={adoptOther}>Adopt a key…</button><button class="btn" onclick={ongenerate}>Generate…</button></div></div>
          {#if keyError}<p class="note bad">{keyError}</p>{/if}
          <ul class="keys">
            {#each keyList as k}
              <li><div class="kname"><b class="mono">{k.name}</b>{#if k.adopted}<span class="chip ok">in backups</span>{:else}<span class="chip">not backed up</span>{/if}</div>
                <div class="kmeta mono">{k.path}</div>
                <div class="kuse">{k.usedBy?.length ? "Used by " + k.usedBy.join(", ") : "Not used by any connection"}</div>
                {#if !k.adopted}<button class="btn adopt" onclick={() => adopt(k.path)}>Adopt</button>{/if}</li>
            {:else}<li class="empty">No private keys found in ~/.ssh.</li>{/each}
          </ul>
        {/if}
      </div>
    </div>
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; background: rgba(6, 10, 16, 0.62); z-index: 25; display: flex; align-items: center; justify-content: center; padding: 16px; }
  .panel { width: min(860px, 100%); height: min(600px, 100%); display: flex; flex-direction: column; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 30px 80px rgba(0, 0, 0, 0.6); overflow: hidden; }
  header { height: 52px; flex: none; display: flex; align-items: center; justify-content: space-between; padding: 0 14px 0 22px; border-bottom: 1px solid var(--line2); }
  h2 { margin: 0; font-size: 15px; font-weight: 600; color: #fff; }
  .x { width: 30px; height: 30px; border: 0; background: none; border-radius: 4px; color: var(--muted); font-size: 18px; }
  .x:hover { background: var(--line); color: #fff; }
  .body { flex: 1; min-height: 0; display: flex; }
  nav { width: 190px; flex: none; background: var(--bar); border-right: 1px solid var(--line2); padding: 8px 0; }
  nav button { display: block; width: 100%; text-align: left; padding: 10px 22px; border: 0; background: none; color: var(--text2); }
  nav button:hover { background: var(--hover); }
  nav button.on { background: var(--active); color: #fff; box-shadow: inset 3px 0 var(--accent); font-weight: 500; }
  .content { flex: 1; min-width: 0; overflow: auto; padding: 20px 26px 26px; }
  h3 { margin: 0 0 6px; font-size: 14px; font-weight: 600; color: #fff; }
  h3.gap { margin-top: 30px; padding-top: 22px; border-top: 1px solid var(--line2); }
  p { margin: 0 0 12px; color: var(--muted); line-height: 1.5; }
  .head { display: flex; justify-content: space-between; align-items: flex-start; gap: 16px; margin-bottom: 8px; }
  .actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; margin: 4px 0 10px; }
  .path { font-size: 11.5px; color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; max-width: 380px; }
  .fields { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin: 6px 0 4px; }
  .fields.one { grid-template-columns: 1fr; max-width: 50%; }
  label { display: grid; gap: 5px; font-size: 12px; color: var(--text2); }
  .checks, .keys { list-style: none; margin: 6px 0 0; padding: 0; border: 1px solid var(--line2); }
  .checks li { display: flex; align-items: center; gap: 10px; padding: 11px 14px; border-top: 1px solid var(--line2); }
  .checks li:first-child, .keys li:first-child { border-top: 0; }
  .checks b { font-weight: 500; min-width: 140px; }
  .detail { color: var(--muted); }
  .dot.warn { background: #e6c07b; }
  .keys li { position: relative; display: grid; gap: 2px; padding: 12px 14px; border-top: 1px solid var(--line2); }
  .kname { display: flex; align-items: center; gap: 8px; }
  .kmeta { font-size: 11.5px; color: var(--muted); }
  .kuse { font-size: 12px; color: var(--text2); }
  .chip.ok { color: var(--ok); }
  .adopt { position: absolute; right: 14px; top: 50%; transform: translateY(-50%); padding: 5px 12px; }
  .empty { color: var(--muted); padding: 16px; }
  .note { padding: 9px 12px; border: 1px solid var(--line2); margin: 10px 0 0; }
  .note.good { color: var(--ok); border-color: rgba(154, 208, 122, 0.4); background: rgba(154, 208, 122, 0.07); }
  .note.bad { color: var(--err-text); border-color: rgba(240, 123, 134, 0.5); background: var(--err-bg); }
  .danger-fill { background: var(--err); border-color: var(--err); color: #1a0508; font-weight: 600; }
</style>
