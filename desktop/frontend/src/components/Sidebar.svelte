<script lang="ts">
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { buildTree } from "../lib/lib";
  import { sb } from "../state/foldertree.svelte";
  import { ui } from "../state/ui.svelte";
  import Branch from "./Branch.svelte";
  import ConnRow from "./ConnRow.svelte";
  import Menu, { type MenuItem } from "./Menu.svelte";
  import Icon from "./Icon.svelte";
  import EmptyState from "./EmptyState.svelte";
  import Card from "./Card.svelte";
  import { connSearch } from "../state/connsearch.svelte";

  let { profiles, folders, open, current, embedded = false, onconnect, onnew, onedit, onduplicate, ontools, onchanged, onclose }: {
    profiles: Profile[];
    folders: string[];
    /** Connections that have an open tab. */
    open: Set<string>;
    /** The connection of the focused pane, if any. */
    current: string;
    /** Renders inside Command Deck's 348px slide-over drawer instead of a resizable left sidebar. */
    embedded?: boolean;
    onconnect: (p: Profile) => void;
    /** Starts a new connection, in the given folder. */
    onnew: (folder: string) => void;
    onedit: (p: Profile) => void;
    /** Opens the new-connection form prefilled with a copy of this connection. */
    onduplicate: (p: Profile) => void;
    ontools: (tab: "import" | "keys") => void;
    onchanged: () => void;
    onclose?: () => void;
  } = $props();

  let addOpen = $state(false);
  let resizing = $state(false);
  const searching = $derived(connSearch.query.trim() !== "");
  const tree = $derived(buildTree(profiles, folders, connSearch.query));
  const focus = (el: HTMLElement) => el.focus();

  $effect(() => {
    sb.data.profiles = profiles;
    sb.data.folders = folders;
    sb.onchanged = onchanged;
  });

  const mac = navigator.userAgent.includes("Mac");
  const addItems = $derived<MenuItem[]>([
    { id: "connection", label: "Connection", sub: "A server you SSH into", icon: "terminal", kbd: mac ? "⌘N" : "" },
    { id: "folder", label: "Folder", sub: "Group connections", icon: "folder-plus", kbd: mac ? "⇧⌘N" : "" },
    ...(sb.selectedFolder ? [{ id: "subfolder", label: `Subfolder in “${sb.selectedFolder.split("/").pop()}”`, sub: "Nest inside the selected folder", icon: "folder-plus" } as MenuItem] : []),
    { id: "import", label: "Import from ~/.ssh/config", sub: "Hosts you already have", icon: "import", gapBefore: true },
    { id: "key", label: "Key", sub: "Adopt or generate one", icon: "key" },
  ]);
  function pick(id: string) {
    addOpen = false;
    if (id === "connection") onnew(sb.selectedFolder);
    else if (id === "folder") sb.startCreate("");
    else if (id === "subfolder") sb.startCreate(sb.selectedFolder);
    else if (id === "import") ontools("import");
    else if (id === "key") ontools("keys");
  }

  function startResize(e: PointerEvent) {
    if (e.button !== 0) return;
    e.preventDefault();
    const startX = e.clientX;
    const startW = ui.sidebarWidth;
    resizing = true;
    const move = (ev: PointerEvent) => ui.setSidebarWidth(startW + (ev.clientX - startX));
    const stop = () => {
      resizing = false;
      window.removeEventListener("pointermove", move);
      window.removeEventListener("pointerup", stop);
    };
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", stop);
  }
</script>

<aside style={embedded ? undefined : `width: ${ui.sidebarWidth}px`} class:embedded class:resizing>
  <div class="head">
    <div class="title"><span>{embedded ? "Saved Connections" : "Connections"}</span><span class="n">{profiles.length}</span></div>
    <div class="head-acts">
      <button type="button" class="btn sm ghost add" data-add-button aria-haspopup="menu" aria-expanded={addOpen} onclick={() => (addOpen = !addOpen)}>
        <Icon name="plus" size={14} /><span>{embedded ? "Host" : "New"}</span>
      </button>
      {#if embedded && onclose}
        <button type="button" class="icon-btn dense" aria-label="Close drawer" onclick={onclose}><Icon name="x" size={14} /></button>
      {/if}
    </div>
    {#if addOpen}
      <Menu items={addItems} label="Add" ignore="[data-add-button]" style="top: 44px; right: 12px" width={248} onselect={pick} onclose={() => (addOpen = false)} />
    {/if}
  </div>

  {#if profiles.length > 0 || folders.length > 0 || searching}
    <div class="filter" role="search">
      <Icon name="search" size={14} />
      <input bind:value={connSearch.query} placeholder="Filter connections" aria-label="Search connections"
             onkeydown={(e) => { if (e.key === "Escape" && connSearch.query) { connSearch.query = ""; e.stopPropagation(); } }} />
      {#if connSearch.query}
        <button type="button" class="icon-btn dense" aria-label="Clear filter" onclick={() => (connSearch.query = "")}><Icon name="x" size={14} /></button>
      {/if}
    </div>
  {/if}

  {#if sb.error}
    <div class="notice bad err" role="alert"><span>{sb.error}</span><button type="button" class="icon-btn dense" aria-label="Dismiss" onclick={() => (sb.error = "")}><Icon name="x" size={14} /></button></div>
  {/if}

  <nav aria-label="Connections" data-scroll>
    {#if tree.unfiled.length}
      <div data-drop="">
        {#each tree.unfiled as p (p.id)}
          <ConnRow profile={p} depth={0} live={open.has(p.id)} current={current === p.id} {onconnect} {onedit} {onduplicate} />
        {/each}
      </div>
    {/if}

    {#if sb.creatingIn === ""}
      <div class="new"><Icon name="folder" />
        <input class="edit" placeholder="Folder name" aria-label="Folder name" bind:value={sb.newName} use:focus onblur={() => sb.commitCreate()}
               onkeydown={(e) => { if (e.key === "Enter") sb.commitCreate(); if (e.key === "Escape") { e.stopPropagation(); sb.creatingIn = null; } }} /></div>
    {/if}
    {#each tree.roots as n (n.path)}
      <Branch node={n} depth={0} {searching} {open} {current} {onconnect} {onedit} {onduplicate} />
    {/each}
    {#if !tree.unfiled.length && !tree.roots.length && sb.creatingIn !== ""}
      {#if profiles.length === 0 && folders.length === 0}
        <EmptyState icon="terminal" title="No connections yet" text="Add a server you SSH into, or import hosts from ~/.ssh/config.">
          <button type="button" class="btn sm primary" onclick={() => onnew("")}>New connection</button>
          <button type="button" class="btn sm" onclick={() => ontools("import")}>Import</button>
        </EmptyState>
      {:else}
        <EmptyState title="No matches" text="Try another name, host, or tag." />
      {/if}
    {/if}

    {#if sb.drag}
      <div class="rootzone" data-drop="" class:ok={sb.dropTarget === "" && sb.dropOk} class:bad={sb.dropTarget === "" && !sb.dropOk}>Drop here to move to top level</div>
    {/if}
    {#if sb.pending && sb.pending.to === ""}
      <Card text={sb.pending.text} sub={sb.pending.sub} confirm="Move" indent={14} onconfirm={() => sb.confirmPending()} oncancel={() => (sb.pending = null)} />
    {/if}
  </nav>

  {#if embedded}
    <div class="drawer-foot">
      <span>Drag host onto pane edge to split</span>
      <button type="button" class="import-link" onclick={() => ontools("import")}>Import</button>
    </div>
  {:else if profiles.length && !sb.learnedOpen && !searching}
    <p class="tip">Double-click to open · Drag onto pane to split</p>
  {/if}

  {#if !embedded}
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div class="resize" role="separator" aria-orientation="vertical" aria-label="Resize connections sidebar" onpointerdown={startResize}></div>
  {/if}
</aside>

<style>
  aside { position: relative; flex: none; display: flex; flex-direction: column; background: var(--rail); border-right: 1px solid var(--hairline); min-height: 0; }
  aside.embedded { flex: 1; width: 100%; background: transparent; border-right: 0; }
  aside.resizing { user-select: none; }
  .head { position: relative; display: flex; align-items: center; justify-content: space-between; height: 42px; padding: 0 var(--s3) 0 var(--s4); }
  .head-acts { display: flex; align-items: center; gap: 4px; }
  .drawer-foot { display: flex; align-items: center; justify-content: space-between; gap: var(--s2); padding: 10px var(--s4); border-top: 1px solid var(--hairline); font-size: var(--t-xs); color: var(--muted); }
  .import-link { font-size: var(--t-xs); color: var(--accent-text); font-weight: 500; }
  .import-link:hover { text-decoration: underline; }
  .title { display: flex; align-items: baseline; gap: var(--s2); font-size: var(--t-sm); font-weight: 600; color: var(--strong); }
  .n { font-size: var(--t-xs); font-weight: 500; font-family: var(--mono); color: var(--dim); }
  .add { gap: var(--s1); padding: 0 var(--s2); color: var(--muted); }
  .add[aria-expanded="true"] { background: var(--active); color: var(--strong); }
  .filter { display: flex; align-items: center; gap: var(--s2); height: 34px; margin: 0 var(--s2) var(--s1); padding: 0 var(--s1) 0 var(--s3); border-radius: var(--r-sm); color: var(--muted); transition: background var(--dur-1) var(--ease); }
  .filter:hover, .filter:focus-within { background: var(--field); }
  .filter input { flex: 1; min-width: 0; height: 100%; border: 0; background: none; padding: 0; box-shadow: none; font-size: var(--t-sm); color: var(--text); }
  .filter input:focus { box-shadow: none; }
  .err { margin: var(--s2) var(--s3) 0; justify-content: space-between; align-items: center; }
  nav { flex: 1; overflow: auto; position: relative; padding: var(--s1) 0 var(--s2); }
  .new { display: flex; align-items: center; gap: var(--s2); height: 36px; margin: 0 var(--s2); padding: 0 var(--s3) 0 14px; border-radius: var(--r-sm); background: var(--hover); color: var(--accent-text); }
  .edit { flex: 1; min-width: 0; height: var(--h-sm); padding: 0 var(--s2); font-size: var(--t-md); font-weight: 500; }
  .tip { margin: 0; padding: var(--s2) var(--s4); font-size: var(--t-xs); color: var(--dim); border-top: 1px solid var(--hairline); }
  .rootzone { margin: var(--s3); padding: var(--s4); text-align: center; font-size: var(--t-sm); color: var(--dim); border: 1px dashed var(--floatline); border-radius: var(--r-sm); }
  .rootzone.ok { color: var(--accent-text); border-color: var(--accent); background: var(--accent-soft); }
  .rootzone.bad { color: var(--err-text); border-color: var(--err); }
  .resize { position: absolute; top: 0; right: -3px; bottom: 0; width: 6px; cursor: col-resize; z-index: 3; }
  .resize:hover::after, aside.resizing .resize::after { content: ""; position: absolute; top: 0; bottom: 0; left: 2px; width: 2px; background: var(--accent-edge); }
</style>
