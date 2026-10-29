<script lang="ts">
  import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
  import type { TreeNode } from "./lib";
  import { dragSource, sb } from "./foldertree.svelte";
  import ConnRow from "./ConnRow.svelte";
  import Card from "./Card.svelte";
  import Branch from "./Branch.svelte";

  let { node, depth, searching, open, current, onconnect, onedit }: {
    node: TreeNode; depth: number; searching: boolean; open: Set<string>; current: string;
    onconnect: (p: Profile) => void; onedit: (p: Profile) => void;
  } = $props();

  const expanded = $derived(sb.expanded(node.path, searching));
  const over = $derived(sb.dropTarget === node.path);
  const focus = (el: HTMLElement) => { el.focus(); (el as HTMLInputElement).select?.(); };
  const pad = $derived(14 + 14 * depth);
</script>

<div class="branch" data-drop={node.path}>
  <div class="head" class:sel={sb.selectedFolder === node.path} class:ok={over && sb.dropOk} class:bad={over && !sb.dropOk}
       style="padding-left: {pad}px" title={over && !sb.dropOk ? sb.dropReason : ""}
       use:dragSource={{ kind: "folder", id: node.path, label: node.name }}>
    {#each Array(depth) as _, i}<span class="guide" style="left: {18 + 14 * i}px"></span>{/each}
    <button class="caret" aria-label={expanded ? "Collapse" : "Expand"} onclick={() => sb.toggle(node.path)} style="transform: rotate({expanded ? 90 : 0}deg)">
      <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M9 6l6 6-6 6" /></svg>
    </button>
    <button class="name" onclick={() => !sb.justDragged && (sb.select(node.path), searching || sb.toggle(node.path))} ondblclick={() => sb.startRename(node.path)}>
      <svg class="ico" class:empty={node.count === 0} width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
        {#if expanded}<path d="M3 8a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v1H7l-3 8H5a2 2 0 0 1-2-2zM7 11h15l-3 8H4z" />{:else}<path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" />{/if}
      </svg>
      {#if sb.renaming === node.path}
        <input class="edit" bind:value={sb.renameTo} use:focus onclick={(e) => e.stopPropagation()} onblur={() => sb.commitRename()}
               onkeydown={(e) => { if (e.key === "Enter") sb.commitRename(); if (e.key === "Escape") { e.stopPropagation(); sb.renaming = ""; } }} />
      {:else}<span class="label-text">{node.name}</span>{/if}
    </button>
    <span class="count">{node.count}</span>
    <button class="more" aria-label="Folder options" onclick={() => (sb.menuFolder = sb.menuFolder === node.path ? "" : node.path)}>
      <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" aria-hidden="true"><circle cx="5" cy="12" r="1.5" /><circle cx="12" cy="12" r="1.5" /><circle cx="19" cy="12" r="1.5" /></svg>
    </button>
    {#if sb.menuFolder === node.path}
      <div class="pop" role="menu">
        <button role="menuitem" onclick={() => sb.startCreate(node.path)}>New subfolder</button>
        <button role="menuitem" onclick={() => sb.startRename(node.path)}>Rename</button>
        <button role="menuitem" class="danger" onclick={() => { sb.confirmDelete = node.path; sb.menuFolder = ""; }}>Delete</button>
      </div>
    {/if}
  </div>

  {#if sb.confirmDelete === node.path}
    <Card text={`Delete the folder ${node.name}?`} sub={node.count ? `It still holds ${node.count} connection${node.count === 1 ? "" : "s"}, so this will be refused. Move them out first.` : "It is empty. Nothing else is touched."} confirm="Delete" danger indent={pad + 8} onconfirm={() => sb.remove(node.path)} oncancel={() => (sb.confirmDelete = "")} />
  {/if}
  {#if sb.pending && sb.pending.to === node.path}
    <Card text={sb.pending.text} sub={sb.pending.sub} confirm="Move" indent={pad + 8} onconfirm={() => sb.confirmPending()} oncancel={() => (sb.pending = null)} />
  {/if}

  {#if expanded}
    {#if sb.creatingIn === node.path}
      <div class="new" style="padding-left: {pad + 14 + 8}px"><svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" aria-hidden="true"><path d="M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z" /></svg>
        <input class="edit" placeholder="Folder name" bind:value={sb.newName} use:focus onblur={() => sb.commitCreate()}
               onkeydown={(e) => { if (e.key === "Enter") sb.commitCreate(); if (e.key === "Escape") { e.stopPropagation(); sb.creatingIn = null; } }} /></div>
    {/if}
    {#each node.children as c (c.path)}
      <Branch node={c} depth={depth + 1} {searching} {open} {current} {onconnect} {onedit} />
    {/each}
    {#each node.profiles as p (p.name)}
      <ConnRow profile={p} depth={depth + 1} live={open.has(p.name)} current={current === p.name} {onconnect} {onedit} />
    {/each}
  {/if}
</div>

<style>
  .head { position: relative; display: flex; align-items: center; gap: 4px; height: 32px; padding-right: 6px; color: var(--text2); user-select: none; }
  .head:hover { background: var(--hover); }
  .head.sel { background: rgba(59, 154, 225, 0.08); box-shadow: inset 2px 0 var(--accent); }
  .head.ok { background: rgba(59, 154, 225, 0.16); outline: 1px dashed var(--accent); outline-offset: -1px; }
  .head.bad { outline: 1px dashed var(--err); outline-offset: -1px; cursor: not-allowed; }
  .guide { position: absolute; top: 0; bottom: 0; width: 1px; background: var(--line); pointer-events: none; }
  .caret { all: unset; width: 16px; height: 22px; display: grid; place-items: center; color: var(--dim); cursor: pointer; transition: transform 0.1s; }
  .name { all: unset; flex: 1; min-width: 0; display: flex; align-items: center; gap: 8px; height: 100%; cursor: default; font-weight: 600; }
  .ico { color: var(--accent-text); flex: none; }
  .ico.empty { color: #5c6b82; }
  .label-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .count { font-size: 11px; color: var(--dim); }
  .more { all: unset; width: 24px; height: 24px; display: none; place-items: center; border-radius: 4px; color: var(--muted); cursor: pointer; }
  .head:hover .more, .more:focus-visible { display: grid; }
  .head:hover .count { display: none; }
  .more:hover { background: var(--line); color: #fff; }
  .edit { flex: 1; min-width: 0; height: 24px; padding: 0 8px; font-size: 13px; font-weight: 500; }
  .new { display: flex; align-items: center; gap: 8px; height: 36px; padding-right: 12px; background: var(--hover); color: var(--accent-text); }
  .pop { position: absolute; right: 6px; top: 30px; z-index: 20; min-width: 150px; background: var(--float); border: 1px solid var(--floatline); border-radius: 6px; box-shadow: 0 12px 30px rgba(0, 0, 0, 0.5); padding: 4px 0; }
  .pop button { all: unset; box-sizing: border-box; display: block; width: 100%; padding: 7px 14px; cursor: pointer; }
  .pop button:hover { background: var(--hover); }
  .pop .danger { color: var(--err-text); }
</style>
