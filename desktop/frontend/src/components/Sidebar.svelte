<script lang="ts">
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { buildTree } from "../lib/lib";
  import { sb } from "../state/foldertree.svelte";
  import Branch from "./Branch.svelte";
  import ConnRow from "./ConnRow.svelte";
  import Menu, { type MenuItem } from "./Menu.svelte";
  import Icon from "./Icon.svelte";
  import EmptyState from "./EmptyState.svelte";
  import Card from "./Card.svelte";
  import { connSearch } from "../state/connsearch.svelte";

  let { profiles, folders, open, current, onconnect, onnew, onedit, ontools, onchanged }: {
    profiles: Profile[];
    folders: string[];
    /** Connections that have an open tab. */
    open: Set<string>;
    /** The connection of the focused pane, if any. */
    current: string;
    onconnect: (p: Profile) => void;
    /** Starts a new connection, in the given folder. */
    onnew: (folder: string) => void;
    onedit: (p: Profile) => void;
    ontools: (tab: "import" | "keys") => void;
    onchanged: () => void;
  } = $props();

  let addOpen = $state(false);
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
</script>

<aside>
  <div class="head">
    <div class="title"><span>Connections</span><span class="n">{profiles.length}</span></div>
    <button type="button" class="btn sm primary add" data-add-button aria-haspopup="menu" aria-expanded={addOpen} onclick={() => (addOpen = !addOpen)}>
      <Icon name="plus" size={14} /><span>Add</span><Icon name="chevron-down" size={14} />
    </button>
    {#if addOpen}
      <Menu items={addItems} label="Add" ignore="[data-add-button]" style="top: 46px; left: 12px" width={244} onselect={pick} onclose={() => (addOpen = false)} />
    {/if}
  </div>

  {#if sb.error}
    <div class="notice bad err" role="alert"><span>{sb.error}</span><button type="button" class="icon-btn dense" aria-label="Dismiss" onclick={() => (sb.error = "")}><Icon name="x" size={14} /></button></div>
  {/if}

  <nav aria-label="Connections" data-scroll>
    {#if tree.unfiled.length}
      <div data-drop="">
        {#each tree.unfiled as p (p.name)}
          <ConnRow profile={p} depth={0} live={open.has(p.name)} current={current === p.name} {onconnect} {onedit} />
        {/each}
      </div>
    {/if}

    {#if sb.creatingIn === ""}
      <div class="new"><Icon name="folder" />
        <input class="edit" placeholder="Folder name" aria-label="Folder name" bind:value={sb.newName} use:focus onblur={() => sb.commitCreate()}
               onkeydown={(e) => { if (e.key === "Enter") sb.commitCreate(); if (e.key === "Escape") { e.stopPropagation(); sb.creatingIn = null; } }} /></div>
    {/if}
    {#each tree.roots as n (n.path)}
      <Branch node={n} depth={0} {searching} {open} {current} {onconnect} {onedit} />
    {/each}
    {#if !tree.unfiled.length && !tree.roots.length && sb.creatingIn !== ""}
      {#if profiles.length === 0 && folders.length === 0}
        <EmptyState icon="terminal" title="No connections yet" text="Add a server you SSH into, or bring in the hosts you already have in ~/.ssh/config.">
          <button type="button" class="btn sm primary" onclick={() => onnew("")}>New connection</button>
          <button type="button" class="btn sm" onclick={() => ontools("import")}>Import</button>
        </EmptyState>
      {:else}
        <EmptyState title="No matches" text="Try a name, host or tag." />
      {/if}
    {/if}

    {#if profiles.length && !sb.learnedOpen && !searching}
      <p class="tip"><b>Double-click</b> a connection to open it in a new tab. Click one that is open to jump to it. Drag it onto a terminal to split.</p>
    {/if}

    {#if sb.drag}
      <div class="rootzone" data-drop="" class:ok={sb.dropTarget === "" && sb.dropOk} class:bad={sb.dropTarget === "" && !sb.dropOk}>Drop here to move to the top level</div>
    {/if}
    {#if sb.pending && sb.pending.to === ""}
      <Card text={sb.pending.text} sub={sb.pending.sub} confirm="Move" indent={14} onconfirm={() => sb.confirmPending()} oncancel={() => (sb.pending = null)} />
    {/if}
  </nav>

  <footer>{profiles.length} connection{profiles.length === 1 ? "" : "s"} · {folders.length} folder{folders.length === 1 ? "" : "s"}</footer>
</aside>

<style>
  aside { position: relative; width: 268px; flex: none; display: flex; flex-direction: column; background: var(--rail); border-right: 1px solid var(--line); min-height: 0; }
  .head { position: relative; display: flex; align-items: center; justify-content: space-between; height: 52px; padding: 0 var(--s3) 0 var(--s4); border-bottom: 1px solid var(--line); }
  .title { display: flex; align-items: baseline; gap: var(--s2); font-size: var(--t-md); font-weight: 600; color: var(--strong); }
  .n { font-size: var(--t-xs); font-weight: 400; color: var(--dim); }
  .add { gap: var(--s1); padding: 0 var(--s2) 0 var(--s3); }
  .err { margin: var(--s2) var(--s3) 0; justify-content: space-between; align-items: center; }
  nav { flex: 1; overflow: auto; position: relative; }
  .new { display: flex; align-items: center; gap: var(--s2); height: 36px; padding: 0 var(--s3) 0 22px; background: var(--hover); color: var(--accent-text); }
  .edit { flex: 1; min-width: 0; height: var(--h-sm); padding: 0 var(--s2); font-size: var(--t-md); font-weight: 500; }
  .tip { margin: var(--s3) var(--s4); font-size: var(--t-sm); line-height: 1.5; color: var(--dim); }
  .tip b { color: var(--muted); font-weight: 600; }
  .rootzone { margin: var(--s3); padding: var(--s4); text-align: center; font-size: var(--t-sm); color: var(--dim); border: 1px dashed var(--floatline); border-radius: var(--r-sm); }
  .rootzone.ok { color: var(--accent-text); border-color: var(--accent); background: var(--accent-soft); }
  .rootzone.bad { color: var(--err-text); border-color: var(--err); }
  footer { display: flex; align-items: center; height: 28px; padding: 0 var(--s4); border-top: 1px solid var(--line); background: var(--bar); font-size: var(--t-sm); color: var(--dim); }
</style>
