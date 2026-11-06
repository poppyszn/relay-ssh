<script lang="ts">
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { dragSource, sb } from "../state/foldertree.svelte";
  import Icon from "./Icon.svelte";

  let { profile, depth, live, current, onconnect, onedit, onduplicate }: {
    profile: Profile;
    depth: number;
    live: boolean;
    current: boolean;
    onconnect: (p: Profile) => void;
    onedit: (p: Profile) => void;
    onduplicate?: (p: Profile) => void;
  } = $props();

  const tags = $derived(profile.tags ?? []);
  const moreTags = $derived(Math.max(0, tags.length - 1));
  const rdp = $derived(profile.protocol === "rdp");
  // RDP always shows its port (it is not ssh's 22); ssh shows it only when it is not the default.
  const address = $derived(`${rdp && profile.domain ? `${profile.domain}\\` : ""}${profile.user}@${profile.host}${rdp ? `:${profile.port || 3389}` : profile.port !== 22 ? `:${profile.port}` : ""}`);
</script>

<div class="row" class:current class:selected={sb.selectedConn === profile.id} class:dragging={sb.drag?.kind === "profile" && sb.drag.id === profile.id} style="padding-left: {14 + 14 * depth + 8}px" use:dragSource={{ kind: "profile", id: profile.id, label: profile.name }}>
  {#each Array(Math.max(0, depth)) as _, i}<span class="guide" style="left: {18 + 14 * i}px"></span>{/each}
  <!-- One click selects the row and shows its open session (if any); a double-click (or Enter on the focused row, which arrives with detail 0) opens a new session. -->
  <button type="button" class="main" onclick={(e) => { if (sb.justDragged) return; if (e.detail === 0) onconnect(profile); else { sb.selectedConn = profile.id; sb.onFocusConnection(profile.id); } }}
          ondblclick={() => { if (sb.justDragged) return; sb.learnOpen(); onconnect(profile); }} title={live ? `Click to show ${profile.name}. Double-click to open another tab.` : `Double-click to open ${profile.name} in a new tab`}>
    <span class="lead">
      {#if rdp}<span class="rdp-icon" class:ok={live} title="Remote desktop (RDP)"><Icon name="rdp" size={14} /></span>{:else}<span class="dot" class:ok={live} class:live class:idle={!live}></span>{/if}
    </span>
    <span class="name">{profile.name}</span>
    <span class="tags">{#if tags[0]}<span class="chip">{tags[0]}</span>{/if}{#if moreTags}<span class="more" title={(profile.tags ?? []).slice(1).join(", ")}>+{moreTags}</span>{/if}</span>
    <span class="l2"><span class="host mono">{address}</span>{#if profile.jump}<span class="via" title="Connects through {profile.jump}">via {profile.jump}</span>{/if}</span>
  </button>
  {#if onduplicate}<button type="button" class="icon-btn dense dup" data-nodrag aria-label="Duplicate {profile.name}" title="Duplicate" onclick={() => onduplicate(profile)}><Icon name="copy" size={14} /></button>{/if}
  <button type="button" class="icon-btn dense edit" data-nodrag aria-label="Edit {profile.name}" title="Edit" onclick={() => onedit(profile)}><Icon name="pencil" size={14} /></button>
</div>

<style>
  /* The highlight is an inset, rounded layer behind the row, so indent guides stay continuous lines.
     No stacking context here (isolation or z-index on .row) or menus opened in a row slide under the next row. */
  .row { position: relative; display: flex; align-items: center; user-select: none; }
  .row::before { content: ""; position: absolute; inset: 1px var(--s2); pointer-events: none; border-radius: var(--r-sm); transition: background var(--dur-1) var(--ease); }
  .row:hover::before { background: var(--hover); }
  .row.selected::before { background: var(--active); }
  .row.current::before { background: var(--accent-hl); }
  .row.dragging { opacity: 0.38; }
  .guide { position: absolute; top: 0; bottom: 0; width: 1px; background: var(--line); pointer-events: none; }
  /* Two lines: name (and one tag) over the address. The lead column holds the state dot or the RDP glyph,
     and the tags leave the edit button its room on the first line, so hovering never re-truncates the name. */
  .main { position: relative; flex: 1; min-width: 0; display: grid; grid-template-columns: 14px minmax(0, auto) 1fr; column-gap: var(--s2); row-gap: 1px; align-items: center; padding: 7px var(--s4) 7px 0; background: none; border: 0; color: inherit; text-align: left; cursor: default; }
  .lead { display: grid; place-items: center; }
  .dot.idle { background: transparent; box-shadow: inset 0 0 0 1.5px var(--dot-off); }
  .rdp-icon { display: inline-flex; color: var(--muted); }
  .rdp-icon.ok { color: var(--ok); }
  .name { min-width: 0; font-weight: 600; color: var(--strong); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .tags { display: flex; align-items: center; gap: var(--s1); min-width: 0; overflow: hidden; margin-right: 46px; }
  .more { font-size: var(--t-xs); color: var(--dim); }
  .l2 { grid-column: 2 / -1; display: flex; align-items: baseline; gap: var(--s2); min-width: 0; font-size: var(--t-xs); white-space: nowrap; }
  /* When both do not fit, the jump host gives way before the address does. */
  .host { flex: 0 0.01 auto; min-width: 0; color: var(--muted); overflow: hidden; text-overflow: ellipsis; }
  .via { flex: 0 1 auto; min-width: 0; color: var(--dim); overflow: hidden; text-overflow: ellipsis; }
  .via::before { content: "·"; margin-right: var(--s2); }
  .edit, .dup { position: absolute; right: var(--s3); top: 5px; opacity: 0; pointer-events: none; }
  .dup { right: calc(var(--s3) + 26px); }
  .row:hover .edit, .row:hover .dup, .edit:focus-visible, .dup:focus-visible { opacity: 1; pointer-events: auto; }
</style>
