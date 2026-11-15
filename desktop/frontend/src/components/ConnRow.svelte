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

<div class="row" class:live class:current class:selected={sb.selectedConn === profile.id} class:dragging={sb.drag?.kind === "profile" && sb.drag.id === profile.id} style="padding-left: {14 + 14 * depth + 8}px" use:dragSource={{ kind: "profile", id: profile.id, label: profile.name }}>
  {#each Array(Math.max(0, depth)) as _, i}<span class="guide" style="left: {18 + 14 * i}px"></span>{/each}
  <!-- One click selects the row and shows its open session (if any); a double-click (or Enter on the focused row, which arrives with detail 0) opens a new session. -->
  <button type="button" class="main" onclick={(e) => { if (sb.justDragged) return; if (e.detail === 0) onconnect(profile); else { sb.selectedConn = profile.id; sb.onFocusConnection(profile.id); } }}
          ondblclick={() => { if (sb.justDragged) return; sb.learnOpen(); onconnect(profile); }} title={live ? `Click to show ${profile.name}. Double-click to open another tab.` : `Double-click to open ${profile.name} in a new tab`}>
    <span class="lead">
      {#if rdp}<span class="rdp-icon" class:ok={live} class:cur={current} title="Remote desktop (RDP)"><Icon name="rdp" size={14} /></span>{:else}<span class="dot" class:ok={live} class:live class:idle={!live && !current}></span>{/if}
    </span>
    <span class="name">{profile.name}</span>
    <span class="trail">
      <span class="tags">{#if tags[0]}<span class="chip">{tags[0]}</span>{/if}{#if moreTags}<span class="more" title={(profile.tags ?? []).slice(1).join(", ")}>+{moreTags}</span>{/if}</span>
      {#if current}<span class="state-badge is-active">Active</span>{:else if live}<span class="state-badge is-open">Open</span>{/if}
    </span>
    <span class="l2"><span class="host mono">{address}</span>{#if profile.jump}<span class="via" title="Connects through {profile.jump}">via {profile.jump}</span>{/if}</span>
  </button>
  {#if onduplicate}<button type="button" class="icon-btn dense dup" data-nodrag aria-label="Duplicate {profile.name}" title="Duplicate" onclick={() => onduplicate(profile)}><Icon name="copy" size={14} /></button>{/if}
  <button type="button" class="icon-btn dense edit" data-nodrag aria-label="Edit {profile.name}" title="Edit" onclick={() => onedit(profile)}><Icon name="pencil" size={14} /></button>
</div>

<style>
  /* The highlight is an inset, frosted pill card behind the row, so indent guides stay continuous lines.
     No stacking context here (isolation or z-index on .row) or menus opened in a row slide under the next row. */
  .row { position: relative; display: flex; align-items: center; user-select: none; }
  .row::before { content: ""; position: absolute; inset: 2px var(--s2); pointer-events: none; border-radius: 10px; transition: background var(--dur-1) var(--ease), box-shadow var(--dur-1) var(--ease); }
  .row:hover::before { background: var(--hover); }
  .row.selected:not(.current):not(.live)::before { background: var(--active); box-shadow: inset 0 0 0 1px var(--hairline); }
  .row.live:not(.current)::before { background: color-mix(in srgb, var(--ok) 6%, var(--hover)); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--ok) 22%, transparent); }
  .row.live:not(.current):hover::before { background: color-mix(in srgb, var(--ok) 10%, var(--active)); box-shadow: inset 0 0 0 1px var(--ok-edge); }
  .row.current::before { background: linear-gradient(90deg, var(--accent-hl) 0%, var(--accent-faint) 100%); box-shadow: inset 0 0 0 1px var(--accent-edge), 0 4px 14px -4px color-mix(in srgb, var(--accent) 30%, transparent); }
  .row.current:hover::before { background: linear-gradient(90deg, color-mix(in srgb, var(--accent) 22%, transparent) 0%, var(--accent-soft) 100%); }
  .row.dragging { opacity: 0.38; }
  .guide { position: absolute; top: 0; bottom: 0; width: 1px; background: var(--hairline); pointer-events: none; }
  /* Clean borderless two-line layout with 14px name and 12px host metadata floor. */
  .main { position: relative; flex: 1; min-width: 0; display: grid; grid-template-columns: 14px minmax(0, 1fr) auto; column-gap: var(--s2); row-gap: 2px; align-items: center; padding: 8px var(--s3) 8px 0; background: none; border: 0; color: inherit; text-align: left; cursor: default; }
  .lead { display: grid; place-items: center; }
  .dot.idle { background: transparent; box-shadow: inset 0 0 0 1.5px var(--dot-off); }
  .row.current .dot.ok { box-shadow: 0 0 0 3px var(--ok-bg), 0 0 8px var(--ok-glow); }
  .rdp-icon { display: inline-flex; color: var(--muted); }
  .rdp-icon.ok { color: var(--ok); }
  .rdp-icon.cur:not(.ok) { color: var(--accent-text); }
  .name { min-width: 0; font-size: var(--t-md); font-weight: 500; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .row.live .name, .row.selected .name { color: var(--strong); }
  .row.current .name { color: var(--strong); font-weight: 600; }
  .trail { display: flex; align-items: center; gap: 6px; min-width: 0; transition: opacity var(--dur-1) var(--ease); }
  .tags { display: flex; align-items: center; gap: var(--s1); min-width: 0; max-width: 112px; overflow: hidden; }
  .row:hover .trail, .row:focus-within .trail { opacity: 0; }
  .state-badge { display: inline-flex; align-items: center; font-size: 10px; font-weight: 600; letter-spacing: 0.04em; text-transform: uppercase; line-height: 1; padding: 3px 7px; border-radius: 999px; white-space: nowrap; }
  .state-badge.is-active { background: var(--accent); color: var(--on-accent); }
  .state-badge.is-open { background: var(--ok-bg); color: var(--ok); box-shadow: inset 0 0 0 1px var(--ok-edge); }
  .more { font-size: var(--t-xs); color: var(--dim); font-family: var(--mono); }
  .l2 { grid-column: 2 / -1; display: flex; align-items: baseline; gap: var(--s2); min-width: 0; font-size: var(--t-xs); white-space: nowrap; padding-right: 4px; }
  .host { flex: 0 0.01 auto; min-width: 0; color: var(--dim); overflow: hidden; text-overflow: ellipsis; }
  .row:hover .host, .row.live .host, .row.selected .host { color: var(--muted); }
  .row.current .host { color: var(--text2); }
  .via { flex: 0 1 auto; min-width: 0; color: var(--dim); overflow: hidden; text-overflow: ellipsis; }
  .via::before { content: "·"; margin-right: var(--s2); }
  .edit, .dup { position: absolute; right: var(--s3); top: 8px; opacity: 0; pointer-events: none; transition: opacity var(--dur-1) var(--ease); }
  .dup { right: calc(var(--s3) + 26px); }
  .row:hover .edit, .row:hover .dup, .row:focus-within .edit, .row:focus-within .dup { opacity: 1; pointer-events: auto; }
</style>
