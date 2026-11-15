<script lang="ts">
  import { PaneGroup, Pane, PaneResizer } from "paneforge";
  import type { Node } from "../lib/panes";
  import { isLeaf } from "../lib/panes";
  import { ws } from "../state/workspace.svelte";
  import { sb, dragSource } from "../state/foldertree.svelte";
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import PaneView from "./PaneView.svelte";
  import SearchBar from "./SearchBar.svelte";
  import ConnectingOverlay from "./ConnectingOverlay.svelte";
  import Icon from "./Icon.svelte";
  import RdpView from "./RdpView.svelte";

  let {
    node,
    tabId,
    multi,
    profiles,
    onfiles,
    onforwards,
    onsnippets,
    oninstallblocks,
  }: {
    node: Node;
    tabId: string;
    multi: boolean;
    profiles: Profile[];
    onfiles?: (target: string) => void;
    onforwards?: (target: string) => void;
    onsnippets?: () => void;
    oninstallblocks?: (paneId: string) => void;
  } = $props();

  function attach(host: HTMLElement, id: string) {
    host.replaceChildren(ws.element(id));
    return {
      update(next: string) { host.replaceChildren(ws.element(next)); },
      destroy() {},
    };
  }

  const tab = $derived(ws.tabs.find((t) => t.id === tabId));

  let dims = $state<{ cols: number; rows: number } | null>(null);
  $effect(() => {
    if (!isLeaf(node)) {
      dims = null;
      return;
    }
    const sess = ws.session(node.id);
    if (!sess) {
      dims = null;
      return;
    }
    dims = { cols: sess.term.cols, rows: sess.term.rows };
    const sub = sess.term.onResize(({ cols, rows }) => {
      dims = { cols, rows };
    });
    return () => sub.dispose();
  });
</script>

{#if isLeaf(node)}
  {@const info = ws.panes[node.id]}
  {@const prof = profiles.find((p) => p.id === info?.target)}
  {@const label = prof?.name ?? (info?.target ? ws.labelOf(info.target) : "Local Shell")}
  {@const isFocused = tab?.focus === node.id}
  {@const isRdp = prof?.protocol === "rdp"}
  <section
    class="leaf"
    data-pane={node.id}
    class:focused={multi && isFocused}
    onpointerdowncapture={() => ws.focus(node.id)}
  >
    {#if multi}
      <header use:dragSource={{ kind: "pane", id: node.id, label }} title="Drag to move this pane">
        <span class="dot" class:ok={info && !info.ended && !info.connecting} class:warn={info?.connecting} class:err={info?.ended}></span>
        <strong>{label}</strong>
        {#if prof}<span class="sub mono">{prof.user}@{prof.host}</span>{/if}
        {#if info?.filled}<span class="filled-note">password filled</span>{/if}
        {#if info?.connecting}<span class="wait">connecting…</span>{/if}
        {#if info?.ended}<span class="gone">disconnected</span>{/if}
        {#if isFocused && info?.blocks}<span class="focused-tag">Focused · Blocks On</span>{/if}
        <button type="button" class="icon-btn dense close" data-nodrag aria-label="Close pane" onclick={() => ws.closePane(node.id)}>
          <Icon name="x" size={14} />
        </button>
      </header>
    {/if}

    {#if isRdp}
      <RdpView target={info?.target ?? ""} {prof} />
    {:else}
      <div class="term" use:attach={node.id}></div>

      {#if info?.connecting && info.target}
        <div class="cover" class:below-header={multi}>
          <ConnectingOverlay target={info.target} {prof} since={info.since} oncancel={() => ws.cancelConnect(node.id)} />
        </div>
      {/if}

      {#if info?.fillHint}
        <div class="fill-hint" role="status">
          <span>{info.fillHint}</span>
          <button type="button" class="icon-btn dense" aria-label="Dismiss hint" onclick={() => { if (ws.panes[node.id]) ws.panes[node.id].fillHint = ""; }}>
            <Icon name="x" size={14} />
          </button>
        </div>
      {/if}

      {#if sb.paneDrop?.id === node.id}
        <div class="dropzone {sb.paneDrop.edge}" class:below-header={multi} aria-hidden="true">
          <span>{sb.paneDrop.verb ? `${sb.paneDrop.verb} ${sb.paneDrop.edge}` : sb.drag?.kind === "pane" ? (sb.paneDrop.edge === "center" ? "Swap places" : `Move ${sb.paneDrop.edge}`) : sb.paneDrop.edge === "center" ? "Open in a new tab" : `Split ${sb.paneDrop.edge}`}</span>
        </div>
      {/if}

      {#if ws.searchPane === node.id && ws.session(node.id)}
        <SearchBar session={ws.session(node.id)!} onclose={() => ws.closeSearch()} />
      {/if}

      {#if info?.ended && !info.bannerHidden}
        <div class="ended" class:below-header={multi} role="alert">
          <div>
            <strong>{info.error ? "Could not connect" : "Session ended"}</strong>
            <span>{info.error || `Press Enter to reconnect to ${info.target ? label : "local shell"}`}</span>
          </div>
          <button type="button" class="btn sm primary" onclick={() => ws.reconnect(node.id)}>Reconnect</button>
          <button type="button" class="icon-btn dense" aria-label="Dismiss" title="Dismiss (press Enter in the terminal to reconnect)" onclick={() => { if (ws.panes[node.id]) ws.panes[node.id].bannerHidden = true; }}>
            <Icon name="x" size={14} />
          </button>
        </div>
      {/if}

      {#if isFocused && !info?.ended && !info?.connecting && (dims || info?.target)}
        <div class="pane-hud" role="toolbar" aria-label="Pane quick actions">
          {#if dims}
            <span class="mono">{dims.cols}×{dims.rows}</span>
          {/if}
          {#if info?.blocks}
            <span class="hud-sep">·</span>
            <span class="hud-ok" title="Command blocks active"><span class="hud-dot"></span>Blocks</span>
          {:else if info?.target && oninstallblocks}
            <span class="hud-sep">·</span>
            <button type="button" class="hud-btn" onclick={() => oninstallblocks(node.id)} title="Install shell hook so commands show exit codes & copy controls">Enable blocks</button>
          {/if}
          {#if info?.target && onfiles}
            <span class="hud-sep">·</span>
            <button type="button" class="hud-btn" onclick={() => onfiles(info.target!)}>SFTP</button>
          {/if}
          {#if info?.target && onforwards}
            <span class="hud-sep">·</span>
            <button type="button" class="hud-btn" onclick={() => onforwards(info.target!)}>Forwards</button>
          {/if}
          {#if onsnippets}
            <span class="hud-sep">·</span>
            <button type="button" class="hud-btn" onclick={onsnippets}>Snippets</button>
          {/if}
        </div>
      {/if}
    {/if}
  </section>
{:else}
  {#key node.sid}
    <PaneGroup
      class="split-group"
      direction={node.dir === "row" ? "horizontal" : "vertical"}
      onLayoutChange={(sizes) => {
        if (sizes.length === 2 && Number.isFinite(sizes[0])) ws.setRatio(node.sid, sizes[0] / 100);
      }}
    >
      <Pane order={1} defaultSize={Math.max(15, Math.min(85, node.ratio * 100))} minSize={15}>
        {#snippet child({ props })}
          {@const { "data-pane": _drop, ...safeProps } = props as Record<string, unknown>}
          <div {...safeProps} class="side">
            <PaneView node={node.a} {tabId} {multi} {profiles} {onfiles} {onforwards} {onsnippets} {oninstallblocks} />
          </div>
        {/snippet}
      </Pane>
      <PaneResizer class="handle" />
      <Pane order={2} defaultSize={Math.max(15, Math.min(85, (1 - node.ratio) * 100))} minSize={15}>
        {#snippet child({ props })}
          {@const { "data-pane": _drop, ...safeProps } = props as Record<string, unknown>}
          <div {...safeProps} class="side">
            <PaneView node={node.b} {tabId} {multi} {profiles} {onfiles} {onforwards} {onsnippets} {oninstallblocks} />
          </div>
        {/snippet}
      </Pane>
    </PaneGroup>
  {/key}
{/if}

<style>
  :global(.split-group) { width: 100%; height: 100%; min-width: 0; min-height: 0; display: flex; }
  .side { min-width: 0; min-height: 0; display: flex; }
  :global(.handle) { flex: none; width: 1px; background: var(--hairline); position: relative; cursor: col-resize; transition: background var(--dur-1) var(--ease); }
  :global(.handle[data-direction="vertical"]) { width: auto; height: 1px; cursor: row-resize; }
  :global(.handle::after) { content: ""; position: absolute; inset: 0 -3px; }
  :global(.handle[data-direction="vertical"]::after) { inset: -3px 0; }
  :global(.handle:hover), :global(.handle[data-active]) { background: var(--accent); }
  .leaf { flex: 1; min-width: 0; min-height: 0; display: flex; flex-direction: column; position: relative; background: var(--term); }
  .leaf.focused { box-shadow: inset 0 0 0 1px var(--accent-edge); }
  header { cursor: grab; height: 32px; flex: none; display: flex; align-items: center; gap: var(--s2); padding: 0 var(--s2) 0 var(--s3); background: var(--rail); border-bottom: 1px solid var(--hairline); font-size: var(--t-xs); color: var(--muted); }
  header strong { color: var(--strong); font-weight: 500; }
  .sub { color: var(--dim); }
  .dot { width: 7px; height: 7px; }
  .gone { color: var(--err-text); }
  .wait { color: var(--warn); }
  .focused-tag { margin-left: auto; color: var(--accent-text); font-weight: 500; }
  header .focused-tag + .close { margin-left: 6px; }
  .dropzone { position: absolute; z-index: 4; pointer-events: none; display: grid; place-items: center; background: var(--accent-soft); border: 2px solid var(--accent); inset: 0; }
  .dropzone.below-header { top: 32px; }
  .dropzone.left { right: 50%; }
  .dropzone.right { left: 50%; }
  .dropzone.top { bottom: 50%; }
  .dropzone.top.below-header { bottom: calc(50% - 16px); }
  .dropzone.bottom { top: 50%; }
  .dropzone.bottom.below-header { top: calc(50% + 16px); }
  .dropzone.center { inset: 18%; border-style: dashed; }
  .dropzone span { padding: var(--s1) var(--s3); background: var(--float); color: var(--accent-text); border: 1px solid var(--float-edge); border-radius: var(--r-sm); font-size: var(--t-sm); font-weight: 600; box-shadow: var(--e-pop); }
  .cover { position: absolute; inset: 0; z-index: 2; }
  .cover.below-header { top: 32px; }
  .close { margin-left: auto; }
  .term { flex: 1; min-height: 0; min-width: 0; position: relative; overflow: hidden; }
  .term :global(.pane-term) { position: absolute; inset: 0; background: var(--term); }
  .term :global(.pane-term .xterm) { padding: 14px 24px 22px; }
  .ended { position: absolute; left: 50%; top: 14px; transform: translateX(-50%); display: flex; align-items: center; gap: var(--s3); padding: var(--s3) var(--s2) var(--s3) var(--s4); background: var(--float); border: 1px solid var(--float-edge); border-radius: var(--r-md); box-shadow: var(--e-pop); white-space: nowrap; z-index: 3; }
  .ended.below-header { top: 44px; }
  .ended div { display: grid; gap: 1px; }
  .ended span { font-size: var(--t-sm); color: var(--muted); }
  .filled-note { font-size: var(--t-xs); color: var(--ok); margin-left: 4px; }
  .fill-hint { position: absolute; top: 36px; left: 50%; transform: translateX(-50%); z-index: 5; display: flex; align-items: center; gap: var(--s2); background: var(--float); border: 1px solid var(--float-edge); color: var(--text); padding: var(--s2) var(--s2) var(--s2) var(--s3); border-radius: var(--r-md); font-size: var(--t-sm); box-shadow: var(--e-pop); }
  .pane-hud {
    position: absolute;
    right: 16px;
    bottom: 10px;
    z-index: 2;
    display: inline-flex;
    align-items: center;
    gap: 7px;
    padding: 4px 12px;
    border-radius: 999px;
    background: var(--float);
    border: 1px solid var(--hairline);
    color: var(--muted);
    font-size: 11.5px;
    opacity: 0.48;
    transition: opacity var(--dur-2) var(--ease), border-color var(--dur-2) var(--ease);
    backdrop-filter: blur(8px);
  }
  .pane-hud:hover, .pane-hud:focus-within {
    opacity: 1;
    border-color: var(--floatline);
  }
  .hud-sep { color: var(--dim); }
  .hud-ok { display: inline-flex; align-items: center; gap: 5px; color: var(--ok); }
  .hud-dot { width: 5px; height: 5px; border-radius: 999px; background: currentColor; }
  .hud-btn { color: var(--muted); font-size: 11.5px; padding: 0 2px; border-radius: 4px; }
  .hud-btn:hover { color: var(--strong); }
</style>
