<script lang="ts">
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { dragSource, sb } from "../state/foldertree.svelte";
  import Icon from "./Icon.svelte";

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

<div class="row" class:current class:selected={sb.selectedConn === profile.name} class:dragging={sb.drag?.kind === "profile" && sb.drag.id === profile.name} style="padding-left: {14 + 14 * depth + 8}px" use:dragSource={{ kind: "profile", id: profile.name, label: profile.name }}>
  {#each Array(Math.max(0, depth)) as _, i}<span class="guide" style="left: {18 + 14 * i}px"></span>{/each}
  <!-- One click selects the row and shows its open session (if any); a double-click (or Enter on the focused row, which arrives with detail 0) opens a new session. -->
  <button type="button" class="main" onclick={(e) => { if (sb.justDragged) return; if (e.detail === 0) onconnect(profile); else { sb.selectedConn = profile.name; sb.onFocusConnection(profile.name); } }}
          ondblclick={() => { if (sb.justDragged) return; sb.learnOpen(); onconnect(profile); }} title={live ? `Click to show ${profile.name}. Double-click to open another tab.` : `Double-click to open ${profile.name} in a new tab`}>
    <span class="l1">
      <span class="dot" class:ok={live} class:live></span>
      <span class="name">{profile.name}</span>
      {#each tags as t}<span class="chip">{t}</span>{/each}
      {#if profile.jump}<span class="jump" title="Connects through {profile.jump}"><Icon name="arrow-right" size={14} />{profile.jump}</span>{/if}
    </span>
    <span class="host mono">{profile.user}@{profile.host}{profile.port !== 22 ? `:${profile.port}` : ""}</span>
  </button>
  <button type="button" class="icon-btn dense edit" data-nodrag aria-label="Edit {profile.name}" title="Edit" onclick={() => onedit(profile)}><Icon name="pencil" size={14} /></button>
</div>

<style>
  .row { position: relative; display: flex; align-items: center; padding-right: var(--s1); user-select: none; transition: background var(--dur-1) var(--ease); }
  .row:hover { background: var(--hover); }
  .row.selected { background: var(--active); }
  .row.current { background: var(--accent-hl); }
  .row.dragging { opacity: 0.38; }
  .guide { position: absolute; top: 0; bottom: 0; width: 1px; background: var(--line); pointer-events: none; }
  .main { flex: 1; min-width: 0; padding: var(--s2) 0; display: grid; gap: 2px; background: none; border: 0; color: inherit; text-align: left; cursor: default; }
  .l1 { display: flex; align-items: center; gap: var(--s2); min-width: 0; }
  .name { font-weight: 600; color: var(--strong); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .l1 .chip { flex: none; }
  .jump { display: flex; align-items: center; gap: 2px; font-size: var(--t-xs); color: var(--dim); flex: none; }
  .host { padding-left: var(--s4); font-size: var(--t-xs); color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .edit { display: none; }
  .row:hover .edit, .edit:focus-visible { display: inline-grid; }
</style>
