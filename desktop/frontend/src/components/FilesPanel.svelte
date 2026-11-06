<script lang="ts">
  import { tick } from "svelte";
  import { FilesService, type FileEntry } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import EmptyState from "./EmptyState.svelte";
  import Icon from "./Icon.svelte";
  import Modal from "./Modal.svelte";

  let { target, name = "", onclose }: { target: string; name?: string; onclose: () => void } = $props();

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

  let listEl: HTMLElement;
  /** The row that takes Tab: the selected one, else the first. */
  const stop = (name: string, i: number) => (selected ? selected === name : i === 0);
  async function focusRow(i: number) {
    await tick();
    listEl?.querySelector<HTMLElement>(`[data-i="${i}"]`)?.focus();
  }
  function listKey(k: KeyboardEvent) {
    if ((k.target as HTMLElement).tagName === "INPUT" || !shown.length) return;
    const at = shown.findIndex((e) => e.name === selected);
    let to = -1;
    if (k.key === "ArrowDown") to = Math.min(at + 1, shown.length - 1);
    else if (k.key === "ArrowUp") to = at < 0 ? shown.length - 1 : Math.max(at - 1, 0);
    else if (k.key === "Home") to = 0;
    else if (k.key === "End") to = shown.length - 1;
    else if (k.key === "Enter" && sel) { k.preventDefault(); open(sel); return; }
    else return;
    k.preventDefault();
    selected = shown[to].name;
    focusRow(to);
  }
  // Esc inside an inline name field cancels the edit, not the whole panel.
  function fieldKey(k: KeyboardEvent, commit: () => void, cancel: () => void) {
    if (k.key === "Enter") commit();
    else if (k.key === "Escape") { k.preventDefault(); k.stopPropagation(); cancel(); }
  }

  $effect(() => { load(""); });
</script>

<Modal title="Files" subtitle={name || target} size="lg" tall flush {onclose}>
  <div class="bar">
    <button type="button" class="icon-btn" aria-label="Up one folder" title="Up one folder" onclick={() => load(parent(path))} disabled={path === "/" || !!busy}><Icon name="arrow-up" /></button>
    <button type="button" class="icon-btn" aria-label="Refresh" title="Refresh" onclick={() => load()} disabled={!!busy}><Icon name="refresh" /></button>
    <input class="mono path" bind:value={pathInput} aria-label="Folder path" onkeydown={(e) => e.key === "Enter" && load(pathInput)} />
    <label class="check"><input type="checkbox" bind:checked={showHidden} />Show hidden</label>
  </div>

  <div class="tools">
    <button type="button" class="btn sm" onclick={pickUpload} disabled={!!busy}><Icon name="import" size={14} />Upload</button>
    <button type="button" class="btn sm" onclick={() => { creating = true; newName = ""; }} disabled={!!busy}><Icon name="folder-plus" size={14} />New folder</button>
    <span class="sp"></span>
    {#if sel}
      {#if !sel.dir}<button type="button" class="btn sm" onclick={() => download(sel)} disabled={!!busy}>Download</button>{/if}
      <button type="button" class="btn sm" onclick={() => { renaming = sel.name; renameTo = sel.name; }} disabled={!!busy}>Rename</button>
      <button type="button" class="btn sm ghost danger" onclick={() => (confirm = { kind: "delete", name: sel.name })} disabled={!!busy}>Delete</button>
    {/if}
  </div>

  {#if error}<div class="strip"><div class="notice bad" role="alert">{error}</div></div>{/if}
  {#if note && !error}<div class="strip"><div class="notice good">{note}</div></div>{/if}

  <div class="row head" aria-hidden="true"><span class="label">Name</span><span class="label num">Size</span><span class="label num">Modified</span><span class="label num">Permissions</span></div>
  <div class="list" role="listbox" aria-label="Files" tabindex="-1" bind:this={listEl} onkeydown={listKey}>
    {#if creating}
      <div class="row">
        <span class="n"><span class="ico dir"><Icon name="folder" /></span><input class="mono edit" bind:value={newName} data-autofocus aria-label="New folder name" placeholder="Folder name" onkeydown={(k) => fieldKey(k, mkdir, () => (creating = false))} /></span>
      </div>
    {/if}
    {#each shown as e, i (e.name)}
      <!-- svelte-ignore a11y_click_events_have_key_events (arrows and Enter are handled on the listbox) -->
      <div class="row item row-hover" class:row-hl={selected === e.name} role="option" aria-selected={selected === e.name} data-i={i} tabindex={stop(e.name, i) ? 0 : -1} onclick={() => (selected = e.name)} ondblclick={() => open(e)}>
        <span class="n">
          <span class="ico" class:dir={e.dir}><Icon name={e.dir ? "folder" : "files"} /></span>
          {#if renaming === e.name}
            <input class="mono edit" bind:value={renameTo} data-autofocus aria-label="New name" onclick={(ev) => ev.stopPropagation()} onkeydown={(k) => fieldKey(k, rename, () => (renaming = ""))} />
          {:else}<span class="name">{e.name}</span>{#if e.link}<span class="chip">link</span>{/if}{/if}
        </span>
        <span class="mono dim num">{size(e.size, e.dir)}</span><span class="mono dim num">{e.modified}</span><span class="mono dim num">{e.perm}</span>
      </div>
    {:else}
      {#if !busy && !error && !creating}
        <EmptyState icon="folder" title="This folder is empty" text="Upload a file or create a folder to get started.">
          <button type="button" class="btn sm" onclick={pickUpload}><Icon name="import" size={14} />Upload</button>
        </EmptyState>
      {/if}
    {/each}
  </div>

  {#snippet footer()}
    <span class="status">{#if busy}<span class="spinner"></span>{/if}{busy || `${shown.length} item${shown.length === 1 ? "" : "s"}`}</span>
    <span class="hint">Enter or double-click opens a folder or downloads a file</span>
  {/snippet}
</Modal>

{#if confirm}
  {@const c = confirm}
  <Modal title={c.kind === "delete" ? `Delete ${sel?.dir ? "folder" : "file"}?` : "Replace file?"} subtitle={c.name} size="sm" role="alertdialog" onclose={() => (confirm = null)}>
    {#if c.kind === "delete"}
      <p class="ask">This removes it from {name || target}. {sel?.dir ? "Only an empty folder can be deleted." : "It cannot be undone."}</p>
    {:else}
      <p class="ask">A file with this name already exists in {path}. Uploading replaces it.</p>
    {/if}
    {#snippet footer()}
      <button type="button" class="btn" data-autofocus onclick={() => (confirm = null)}>Cancel</button>
      {#if c.kind === "delete"}
        <button type="button" class="btn danger" onclick={remove}>Delete</button>
      {:else}
        <button type="button" class="btn danger" onclick={() => upload(c.local!, true)}>Replace</button>
      {/if}
    {/snippet}
  </Modal>
{/if}

<style>
  .bar { display: flex; align-items: center; gap: var(--s2); padding: var(--s3) var(--s5); border-bottom: 1px solid var(--line); }
  .path { flex: 1; }
  .check { display: flex; align-items: center; gap: var(--s2); font-size: var(--t-sm); color: var(--muted); white-space: nowrap; }
  .tools { display: flex; align-items: center; gap: var(--s2); padding: var(--s3) var(--s5); border-bottom: 1px solid var(--line); }
  .sp { flex: 1; }
  .strip { padding: var(--s3) var(--s5) 0; }
  .row { display: grid; grid-template-columns: minmax(0, 1fr) 80px 150px 100px; gap: var(--s3); align-items: center; padding: 0 var(--s5); height: 36px; border-bottom: 1px solid var(--line); }
  .row.head { flex: none; height: 32px; background: var(--bar); }
  .num { text-align: right; }
  .list { flex: 1; min-height: 0; overflow: auto; outline: none; }
  .item { cursor: default; outline-offset: -2px; }
  .n { display: flex; align-items: center; gap: var(--s2); min-width: 0; }
  .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .ico { display: grid; color: var(--dim); flex: none; }
  .ico.dir { color: var(--accent-text); }
  .dim { color: var(--muted); font-size: var(--t-sm); white-space: nowrap; }
  .edit { flex: 1; max-width: 360px; }
  .status { display: flex; align-items: center; gap: var(--s2); margin-right: auto; font-size: var(--t-sm); color: var(--muted); }
  .ask { margin: 0; color: var(--text2); }
</style>
