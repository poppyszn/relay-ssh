<script lang="ts">
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import type { TreeNode } from "../lib/lib";
  import { dragSource, sb } from "../state/foldertree.svelte";
  import ConnRow from "./ConnRow.svelte";
  import Card from "./Card.svelte";
  import Branch from "./Branch.svelte";
  import Icon from "./Icon.svelte";
  import Menu from "./Menu.svelte";

  let { node, depth, searching, open, current, onconnect, onedit, onduplicate }: {
    node: TreeNode; depth: number; searching: boolean; open: Set<string>; current: string;
    onconnect: (p: Profile) => void; onedit: (p: Profile) => void; onduplicate?: (p: Profile) => void;
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
    {#each Array(Math.max(0, depth)) as _, i}<span class="guide" style="left: {18 + 14 * i}px"></span>{/each}
    <button type="button" class="icon-btn dense caret" data-nodrag aria-label={expanded ? "Collapse" : "Expand"} onclick={() => sb.toggle(node.path)} style="transform: rotate({expanded ? 90 : 0}deg)"><Icon name="chevron-right" size={14} /></button>
    <button type="button" class="name" onclick={() => !sb.justDragged && (sb.select(node.path), searching || sb.toggle(node.path))} ondblclick={() => sb.startRename(node.path)}>
      <span class="ico" class:empty={node.count === 0}><Icon name={expanded ? "folder-open" : "folder"} /></span>
      {#if sb.renaming === node.path}
        <input class="edit" aria-label="Folder name" bind:value={sb.renameTo} use:focus onclick={(e) => e.stopPropagation()} onblur={() => sb.commitRename()}
               onkeydown={(e) => { if (e.key === "Enter") sb.commitRename(); if (e.key === "Escape") { e.stopPropagation(); sb.renaming = ""; } }} />
      {:else}<span class="label-text">{node.name}</span>{/if}
    </button>
    <span class="count">{node.count}</span>
    <button type="button" class="icon-btn dense more" data-nodrag data-folder-more aria-label="Folder options" aria-haspopup="menu" onclick={() => (sb.menuFolder = sb.menuFolder === node.path ? "" : node.path)}><Icon name="more" size={14} /></button>
    {#if sb.menuFolder === node.path}
      <Menu label="Folder options" ignore="[data-folder-more]" style="right: 6px; top: 30px" width={168}
        items={[{ id: "sub", label: "New subfolder" }, { id: "rename", label: "Rename" }, { id: "delete", label: "Delete", danger: true }]}
        onclose={() => (sb.menuFolder = "")}
        onselect={(id) => {
          if (id === "sub") sb.startCreate(node.path);
          else if (id === "rename") sb.startRename(node.path);
          else { sb.confirmDelete = node.path; }
          sb.menuFolder = "";
        }} />
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
      <div class="new" style="padding-left: {pad + 14 + 8}px"><Icon name="folder" />
        <input class="edit" placeholder="Folder name" aria-label="Folder name" bind:value={sb.newName} use:focus onblur={() => sb.commitCreate()}
               onkeydown={(e) => { if (e.key === "Enter") sb.commitCreate(); if (e.key === "Escape") { e.stopPropagation(); sb.creatingIn = null; } }} /></div>
    {/if}
    {#each node.children as c (c.path)}
      <Branch node={c} depth={depth + 1} {searching} {open} {current} {onconnect} {onedit} {onduplicate} />
    {/each}
    {#each node.profiles as p (p.name)}
      <ConnRow profile={p} depth={depth + 1} live={open.has(p.name)} current={current === p.name} {onconnect} {onedit} {onduplicate} />
    {/each}
  {/if}
</div>

<style>
  /* Same inset, rounded highlight as connection rows (see ConnRow), and the same rule: no stacking
     context on .head, or the folder menu opens underneath the rows that follow. */
  .head { position: relative; display: flex; align-items: center; gap: var(--s1); height: var(--h-md); padding-right: var(--s4); color: var(--text2); user-select: none; }
  .head::before { content: ""; position: absolute; inset: 1px var(--s2); pointer-events: none; border-radius: var(--r-sm); transition: background var(--dur-1) var(--ease); }
  .head:hover::before { background: var(--hover); }
  .head.sel::before { background: var(--accent-faint); }
  .head.ok::before { background: var(--accent-hl); outline: 1px dashed var(--accent); outline-offset: -1px; }
  .head.bad::before { outline: 1px dashed var(--err); outline-offset: -1px; }
  .head.bad { cursor: not-allowed; }
  .guide { position: absolute; top: 0; bottom: 0; width: 1px; background: var(--line); pointer-events: none; }
  .caret, .name, .count, .more, .edit { position: relative; }
  .caret { width: 20px; border-radius: var(--r-xs); color: var(--dim); transition: transform var(--dur-2) var(--ease), background var(--dur-1) var(--ease); }
  .name { flex: 1; min-width: 0; display: flex; align-items: center; gap: var(--s2); height: 100%; padding: 0; background: none; border: 0; color: inherit; text-align: left; cursor: default; font-weight: 600; }
  .ico { display: grid; color: var(--accent-text); flex: none; }
  .ico.empty { color: var(--dim); }
  .label-text { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .count { font-size: var(--t-xs); color: var(--dim); }
  .more { display: none; }
  .head:hover .more, .more:focus-visible { display: inline-grid; }
  .head:hover .count { display: none; }
  .edit { flex: 1; min-width: 0; height: var(--h-sm); padding: 0 var(--s2); font-size: var(--t-md); font-weight: 500; }
  .new { display: flex; align-items: center; gap: var(--s2); height: 36px; margin: 0 var(--s2); padding-right: var(--s3); border-radius: var(--r-sm); background: var(--hover); color: var(--accent-text); }
</style>
