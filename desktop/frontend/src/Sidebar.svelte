<script lang="ts">
  import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { buildTree } from "./lib";
  import { sb } from "./foldertree.svelte";
  import Branch from "./Branch.svelte";
  import ConnRow from "./ConnRow.svelte";
  import AddMenu from "./AddMenu.svelte";
  import Card from "./Card.svelte";
  import { connSearch } from "./connsearch.svelte";

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

  const add = (fn: () => void) => () => { addOpen = false; fn(); };
</script>

<aside>
  <div class="head">
    <div class="title"><span>Connections</span><span class="n">{profiles.length}</span></div>
    <button class="add" data-add-button aria-haspopup="menu" aria-expanded={addOpen} onclick={() => (addOpen = !addOpen)}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" aria-hidden="true"><path d="M12 5v14M5 12h14" /></svg><span>Add</span>
      <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
    </button>
    {#if addOpen}
      <AddMenu folder={sb.selectedFolder} onclose={() => (addOpen = false)}
        onconnection={add(() => onnew(sb.selectedFolder))} onfolder={add(() => sb.startCreate(""))} onsubfolder={add(() => sb.startCreate(sb.selectedFolder))}
        onimport={add(() => ontools("import"))} onkey={add(() => ontools("keys"))} />
    {/if}
  </div>

  {#if sb.error}<p class="err" role="alert">{sb.error}<button aria-label="Dismiss" onclick={() => (sb.error = "")}>&times;</button></p>{/if}

  <nav aria-label="Connections" data-scroll>
    {#if tree.unfiled.length}
      <div data-drop="">
        {#each tree.unfiled as p (p.name)}
          <ConnRow profile={p} depth={0} live={open.has(p.name)} current={current === p.name} {onconnect} {onedit} />
        {/each}
      </div>
    {/if}

    {#if sb.creatingIn === ""}
      <div class="new"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" /></svg>
        <input class="edit" placeholder="Folder name" bind:value={sb.newName} use:focus onblur={() => sb.commitCreate()}
               onkeydown={(e) => { if (e.key === "Enter") sb.commitCreate(); if (e.key === "Escape") { e.stopPropagation(); sb.creatingIn = null; } }} /></div>
    {/if}
    {#each tree.roots as n (n.path)}
      <Branch node={n} depth={0} {searching} {open} {current} {onconnect} {onedit} />
    {/each}
    {#if !tree.unfiled.length && !tree.roots.length && sb.creatingIn !== ""}
      <p class="empty">{profiles.length === 0 ? "No connections yet. Use Add to create one." : "No matches."}</p>
    {/if}

    {#if profiles.length && !sb.learnedOpen && !searching}
      <p class="tip"><b>Double-click</b> a connection to open it in a new tab. Drag it onto a terminal to split.</p>
    {/if}

    {#if sb.drag}
      <div class="rootzone" data-drop="" class:ok={sb.dropTarget === "" && sb.dropOk} class:bad={sb.dropTarget === "" && !sb.dropOk}>Drop here to move to the top level</div>
    {/if}
    {#if sb.pending && sb.pending.to === ""}
      <Card text={sb.pending.text} sub={sb.pending.sub} confirm="Move" indent={14} onconfirm={() => sb.confirmPending()} oncancel={() => (sb.pending = null)} />
    {/if}
  </nav>

  <footer>
    <span>{profiles.length} connection{profiles.length === 1 ? "" : "s"} · {folders.length} folder{folders.length === 1 ? "" : "s"}</span>
    <span class="tools">
      <button aria-label="Import from ssh config" title="Import from ~/.ssh/config" onclick={() => ontools("import")}>
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3v12M7 10l5 5 5-5M4 20h16" /></svg>
      </button>
      <button aria-label="Keys" title="Keys" onclick={() => ontools("keys")}>
        <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><circle cx="8" cy="15" r="4" /><path d="M11 12l9-9M16 7l3 3" /></svg>
      </button>
    </span>
  </footer>
</aside>

<style>
  aside { position: relative; width: 268px; flex: none; display: flex; flex-direction: column; background: var(--rail); border-right: 1px solid var(--line); min-height: 0; }
  .head { position: relative; display: flex; align-items: center; justify-content: space-between; padding: 14px 14px 12px 16px; border-bottom: 1px solid var(--line); }
  .title { display: flex; align-items: baseline; gap: 8px; font-size: 13px; font-weight: 600; color: var(--strong); }
  .n { font-size: 11px; font-weight: 400; color: var(--dim); }
  .add { display: flex; align-items: center; gap: 4px; height: 28px; padding: 0 8px 0 9px; border: 0; border-radius: 4px; background: var(--accent); color: var(--on-accent); font-weight: 600; font-size: 12px; }
  .add:hover { background: var(--accent-hover); }
  .err { margin: 0 12px 8px; padding: 7px 10px; display: flex; align-items: center; gap: 8px; font-size: 12px; color: var(--err-text); background: var(--err-bg); border: 1px solid var(--err); }
  .err button { all: unset; margin-left: auto; cursor: pointer; font-size: 15px; }
  nav { flex: 1; overflow: auto; position: relative; }
  .new { display: flex; align-items: center; gap: 8px; height: 36px; padding: 0 12px 0 22px; background: var(--hover); color: var(--accent-text); }
  .edit { flex: 1; min-width: 0; height: 24px; padding: 0 8px; font-size: 13px; font-weight: 500; }
  .tip { margin: 12px 16px; font-size: 11.5px; line-height: 1.5; color: var(--dim); }
  .tip b { color: var(--muted); font-weight: 600; }
  .empty { padding: 16px; margin: 0; color: var(--muted); }
  .rootzone { margin: 10px 12px; padding: 14px; text-align: center; font-size: 12px; color: var(--dim); border: 1px dashed var(--floatline); }
  .rootzone.ok { color: var(--accent-text); border-color: var(--accent); background: var(--accent-soft); }
  .rootzone.bad { color: var(--err-text); border-color: var(--err); }
  footer { display: flex; align-items: center; justify-content: space-between; height: 40px; padding: 0 8px 0 16px; border-top: 1px solid var(--line); background: var(--bar); font-size: 11.5px; color: var(--dim); }
  .tools { display: flex; gap: 2px; }
  .tools button { width: 28px; height: 28px; display: grid; place-items: center; border: 0; background: none; border-radius: 4px; color: var(--muted); }
  .tools button:hover { background: var(--line); color: var(--strong); }
</style>
