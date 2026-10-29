<script lang="ts">
  import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { dragSource, sb } from "./foldertree.svelte";

  let { profile, depth, live, current, onconnect, onedit }: {
    profile: Profile;
    depth: number;
    live: boolean;
    current: boolean;
    onconnect: (p: Profile) => void;
    onedit: (p: Profile) => void;
  } = $props();

  const tags = $derived((profile.tags ?? []).slice(0, 2));
</script>

<div class="row" class:current class:dragging={sb.drag?.kind === "profile" && sb.drag.id === profile.name} style="padding-left: {14 + 14 * depth + 8}px" use:dragSource={{ kind: "profile", id: profile.name, label: profile.name }}>
  {#each Array(Math.max(0, depth)) as _, i}<span class="guide" style="left: {18 + 14 * i}px"></span>{/each}
  <button class="main" onclick={() => !sb.justDragged && onconnect(profile)} title="Open {profile.name}">
    <span class="l1">
      <span class="dot" class:live></span>
      <span class="name">{profile.name}</span>
      {#each tags as t}<span class="tag">{t}</span>{/each}
      {#if profile.jump}<span class="jump" title="Connects through {profile.jump}"><svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12h12M13 8l4 4-4 4" /></svg>{profile.jump}</span>{/if}
    </span>
    <span class="host mono">{profile.user}@{profile.host}{profile.port !== 22 ? `:${profile.port}` : ""}</span>
  </button>
  <button class="edit" data-nodrag aria-label="Edit {profile.name}" title="Edit" onclick={() => onedit(profile)}>
    <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z" /></svg>
  </button>
</div>

<style>
  .row { position: relative; display: flex; align-items: stretch; padding-right: 4px; user-select: none; }
  .row:hover { background: var(--hover); }
  .row.current { background: var(--active); box-shadow: inset 3px 0 var(--accent); }
  .row.dragging { opacity: 0.38; }
  .guide { position: absolute; top: 0; bottom: 0; width: 1px; background: var(--line); pointer-events: none; }
  .main { all: unset; flex: 1; min-width: 0; padding: 7px 0; display: grid; gap: 2px; cursor: default; }
  .l1 { display: flex; align-items: center; gap: 8px; min-width: 0; }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: var(--dot-off); flex: none; }
  .dot.live { background: var(--ok); box-shadow: 0 0 0 3px var(--ok-glow); }
  .name { font-weight: 600; color: var(--strong); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .tag { font-size: 10px; color: var(--muted); background: var(--line); padding: 0 6px; border-radius: 3px; flex: none; }
  .jump { display: flex; align-items: center; gap: 3px; font-size: 10.5px; color: var(--dim); flex: none; }
  .host { padding-left: 16px; font-size: 11px; color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .edit { all: unset; width: 28px; display: none; place-items: center; color: var(--muted); cursor: pointer; }
  .row:hover .edit, .edit:focus-visible { display: grid; }
  .edit:hover { color: var(--strong); }
</style>
