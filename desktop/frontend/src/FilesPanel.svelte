<script lang="ts">
  import { FilesService, type FileEntry } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

  let { target, onclose }: { target: string; onclose: () => void } = $props();

  let path = $state("");
  let pathInput = $state("");
  let entries = $state<FileEntry[]>([]);
  let selected = $state("");
  let showHidden = $state(false);
  let busy = $state("");
  let error = $state("");
  let note = $state("");
  let renaming = $state("");
  let renameTo = $state("");
  let creating = $state(false);
  let newName = $state("");
  let confirm = $state<null | { kind: "delete" | "overwrite"; name: string; local?: string }>(null);

  const msg = (e: unknown) => (e as Error)?.message ?? String(e);
  const join = (dir: string, name: string) => (dir === "/" ? "/" + name : dir + "/" + name);
  const parent = (p: string) => p.replace(/\/[^/]+\/?$/, "") || "/";
  const base = (p: string) => p.split(/[\\/]/).pop() ?? p;
  const sel = $derived(entries.find((e) => e.name === selected));
  const shown = $derived(
    entries
      .filter((e) => showHidden || !e.name.startsWith("."))
      .sort((a, b) => Number(b.dir) - Number(a.dir) || a.name.localeCompare(b.name, undefined, { sensitivity: "base" })),
  );
  function size(n: number, dir: boolean) {
    if (dir) return "";
    if (n < 1024) return `${n} B`;
    if (n < 1024 ** 2) return `${(n / 1024).toFixed(1)} KB`;
    if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MB`;
    return `${(n / 1024 ** 3).toFixed(2)} GB`;
  }

  async function load(dir = path) {
    busy = "Loading…"; error = ""; selected = ""; renaming = ""; creating = false; confirm = null;
    try {
      const l = await FilesService.List(target, dir);
      path = l.path; pathInput = l.path; entries = l.entries ?? [];
    } catch (e) { error = msg(e); } finally { busy = ""; }
  }
  const open = (e: FileEntry) => (e.dir || e.link ? load(join(path, e.name)).then(() => { if (error && e.link) { error = ""; download(e); } }) : download(e));

  async function download(e: FileEntry) {
    error = ""; note = "";
    try {
      const dest = await FilesService.PickDownloadPath(e.name);
      if (!dest) return;
      busy = `Downloading ${e.name}…`;
      await FilesService.Download(target, join(path, e.name), dest);
      note = `Downloaded ${e.name}`;
    } catch (er) { error = msg(er); } finally { busy = ""; }
  }
  async function upload(local: string, overwrite = false) {
    const name = base(local);
    if (!overwrite && entries.some((e) => e.name === name)) return (confirm = { kind: "overwrite", name, local });
    confirm = null; error = ""; note = "";
    try {
      busy = `Uploading ${name}…`;
      await FilesService.Upload(target, local, join(path, name));
      note = `Uploaded ${name}`;
      await load();
    } catch (er) { error = msg(er); busy = ""; }
  }
  async function pickUpload() {
    try { const l = await FilesService.PickUploadPath(); if (l) await upload(l); } catch (e) { error = msg(e); }
  }
  async function mkdir() {
    const name = newName.trim();
    creating = false; newName = "";
    if (!name) return;
    try { await FilesService.Mkdir(target, join(path, name)); await load(); } catch (e) { error = msg(e); }
  }
  async function rename() {
    const from = renaming, to = renameTo.trim();
    renaming = "";
    if (!to || to === from) return;
    try { await FilesService.Rename(target, join(path, from), join(path, to)); await load(); } catch (e) { error = msg(e); }
  }
  async function remove() {
    const e = sel; confirm = null;
    if (!e) return;
    try { await FilesService.Remove(target, join(path, e.name), e.dir); note = `Deleted ${e.name}`; await load(); } catch (er) { error = msg(er); }
  }

  $effect(() => { load(""); });
</script>

<div class="scrim" role="presentation" onclick={onclose} onkeydown={(e) => e.key === "Escape" && !renaming && !creating && onclose()}>
  <div class="panel" role="dialog" aria-label="Files" tabindex="-1" onclick={(e) => e.stopPropagation()}>
    <header>
      <h2>Files <span class="host mono">{target}</span></h2>
      <button class="x" aria-label="Close" onclick={onclose}>&times;</button>
    </header>

    <div class="bar">
      <button class="icon" aria-label="Up one folder" title="Up" onclick={() => load(parent(path))} disabled={path === "/" || !!busy}>
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 19V5M6 11l6-6 6 6" /></svg>
      </button>
      <button class="icon" aria-label="Refresh" title="Refresh" onclick={() => load()} disabled={!!busy}>
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7" /></svg>
      </button>
      <input class="mono" bind:value={pathInput} aria-label="Folder path" onkeydown={(e) => e.key === "Enter" && load(pathInput)} />
      <label class="hid"><input type="checkbox" bind:checked={showHidden} />Hidden</label>
    </div>

    <div class="tools">
      <button class="btn" onclick={pickUpload} disabled={!!busy}>Upload…</button>
      <button class="btn" onclick={() => { creating = true; newName = ""; }} disabled={!!busy}>New folder</button>
      <span class="sp"></span>
      {#if sel && !confirm}
        {#if !sel.dir}<button class="btn" onclick={() => download(sel)} disabled={!!busy}>Download</button>{/if}
        <button class="btn" onclick={() => { renaming = sel.name; renameTo = sel.name; }} disabled={!!busy}>Rename</button>
        <button class="btn danger" onclick={() => (confirm = { kind: "delete", name: sel.name })} disabled={!!busy}>Delete</button>
      {/if}
    </div>

    {#if confirm}
      <div class="confirm" role="alertdialog">
        {#if confirm.kind === "delete"}
          <span>Delete <b>{confirm.name}</b> from {target}? {sel?.dir ? "Only an empty folder can be deleted." : "This cannot be undone."}</span>
          <button class="btn" onclick={() => (confirm = null)}>Cancel</button><button class="btn primary danger-fill" onclick={remove}>Delete</button>
        {:else}
          <span><b>{confirm.name}</b> already exists here. Replace it?</span>
          <button class="btn" onclick={() => (confirm = null)}>Cancel</button><button class="btn primary" onclick={() => upload(confirm!.local!, true)}>Replace</button>
        {/if}
      </div>
    {/if}
    {#if error}<p class="msg bad" role="alert">{error}</p>{/if}
    {#if note && !error}<p class="msg good">{note}</p>{/if}

    <div class="list" role="grid" aria-label="Files">
      <div class="row head"><span>Name</span><span>Size</span><span>Modified</span><span>Permissions</span></div>
      {#if creating}
        <div class="row"><span class="n"><svg class="ico" width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" /></svg><!-- svelte-ignore a11y_autofocus --><input class="mono edit" bind:value={newName} autofocus placeholder="Folder name" onkeydown={(e) => { if (e.key === "Enter") mkdir(); if (e.key === "Escape") { e.stopPropagation(); creating = false; } }} /></span></div>
      {/if}
      {#each shown as e (e.name)}
        <div class="row item" class:sel={selected === e.name} role="row" tabindex="0" onclick={() => (selected = e.name)} ondblclick={() => open(e)} onkeydown={(k) => k.key === "Enter" && open(e)}>
          <span class="n">
            <svg class="ico" class:dir={e.dir} width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round" aria-hidden="true">
              {#if e.dir}<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />{:else}<path d="M6 3h8l4 4v14H6zM14 3v4h4" />{/if}
            </svg>
            {#if renaming === e.name}
              <!-- svelte-ignore a11y_autofocus -->
              <input class="mono edit" bind:value={renameTo} autofocus onclick={(ev) => ev.stopPropagation()} onkeydown={(k) => { if (k.key === "Enter") rename(); if (k.key === "Escape") { k.stopPropagation(); renaming = ""; } }} />
            {:else}<span class="name">{e.name}</span>{#if e.link}<span class="chip">link</span>{/if}{/if}
          </span>
          <span class="mono dim">{size(e.size, e.dir)}</span><span class="dim">{e.modified}</span><span class="mono dim">{e.perm}</span>
        </div>
      {:else}
        {#if !busy}<p class="empty">{error ? "" : "This folder is empty."}</p>{/if}
      {/each}
    </div>
    <footer>{busy || `${shown.length} item${shown.length === 1 ? "" : "s"}`}<span class="sp"></span><span>Double-click a folder to open it or a file to download it</span></footer>
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; background: var(--scrim); z-index: 25; display: flex; align-items: center; justify-content: center; padding: 16px; }
  .panel { width: min(920px, 100%); height: min(620px, 100%); display: flex; flex-direction: column; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 30px 80px var(--shadow); overflow: hidden; }
  header { height: 52px; flex: none; display: flex; align-items: center; justify-content: space-between; padding: 0 14px 0 22px; border-bottom: 1px solid var(--line2); }
  h2 { margin: 0; font-size: 15px; font-weight: 600; color: var(--strong); display: flex; align-items: center; gap: 10px; }
  .host { font-size: 12px; font-weight: 400; color: var(--muted); background: var(--line); padding: 1px 8px; border-radius: 3px; }
  .x { width: 30px; height: 30px; border: 0; background: none; border-radius: 4px; color: var(--muted); font-size: 18px; }
  .x:hover { background: var(--line); color: var(--strong); }
  .bar { display: flex; align-items: center; gap: 8px; padding: 10px 14px; border-bottom: 1px solid var(--line2); }
  .bar input[type="text"], .bar > input { flex: 1; height: 32px; font-size: 12.5px; }
  .icon { width: 32px; height: 32px; display: grid; place-items: center; background: var(--field); border: 1px solid var(--fieldline); border-radius: 4px; color: var(--text2); }
  .icon:hover:not(:disabled) { border-color: var(--accent); color: var(--strong); }
  .icon:disabled { opacity: 0.4; }
  .hid { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--muted); }
  .hid input { width: 14px; height: 14px; padding: 0; accent-color: var(--accent); }
  .tools { display: flex; align-items: center; gap: 8px; padding: 10px 14px; border-bottom: 1px solid var(--line2); min-height: 52px; }
  .tools .btn { padding: 6px 14px; }
  .sp { flex: 1; }
  .confirm { display: flex; align-items: center; gap: 10px; padding: 10px 14px; background: var(--err-bg); border-bottom: 1px solid var(--line2); }
  .confirm span { flex: 1; }
  .danger-fill { background: var(--err); border-color: var(--err); color: var(--on-err); }
  .msg { margin: 0; padding: 8px 14px; font-size: 12.5px; border-bottom: 1px solid var(--line2); }
  .msg.bad { color: var(--err-text); background: var(--err-bg); }
  .msg.good { color: var(--ok); }
  .list { flex: 1; min-height: 0; overflow: auto; }
  .row { display: grid; grid-template-columns: minmax(0, 1fr) 90px 150px 110px; align-items: center; padding: 0 14px; height: 34px; border-bottom: 1px solid var(--line); }
  .row.head { position: sticky; top: 0; background: var(--bar); font-size: 10.5px; font-weight: 600; letter-spacing: 0.1em; text-transform: uppercase; color: var(--muted); z-index: 1; }
  .item { cursor: default; }
  .item:hover { background: var(--hover); }
  .item.sel { background: var(--active); box-shadow: inset 3px 0 var(--accent); }
  .n { display: flex; align-items: center; gap: 10px; min-width: 0; }
  .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .ico { color: var(--dim); flex: none; }
  .ico.dir { color: var(--accent-text); }
  .dim { color: var(--muted); font-size: 12px; }
  .edit { height: 26px; font-size: 12.5px; flex: 1; }
  .empty { padding: 24px 14px; margin: 0; color: var(--muted); }
  footer { display: flex; align-items: center; padding: 0 14px; height: 30px; font-size: 11.5px; color: var(--muted); border-top: 1px solid var(--line2); background: var(--foot); }
</style>
