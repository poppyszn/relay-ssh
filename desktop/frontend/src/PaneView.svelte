<script lang="ts">
  import type { Node } from "./panes";
  import { isLeaf } from "./panes";
  import { ws } from "./workspace.svelte";
  import { sb, dragSource } from "./foldertree.svelte";
  import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
  import PaneView from "./PaneView.svelte";
  import SearchBar from "./SearchBar.svelte";
  import ConnectingOverlay from "./ConnectingOverlay.svelte";
  import BlocksHint from "./BlocksHint.svelte";

  let { node, tabId, multi, profiles }: { node: Node; tabId: string; multi: boolean; profiles: Profile[] } = $props();

  // Moves the pane's persistent terminal element into this leaf.
  function attach(host: HTMLElement, id: string) {
    host.replaceChildren(ws.element(id));
    return {
      // A leaf component can be reused for another pane when panes are rearranged.
      update(next: string) { host.replaceChildren(ws.element(next)); },
      destroy() { /* element stays alive in the workspace; the next leaf re-adopts it */ },
    };
  }

  function drag(e: PointerEvent, sid: string, dir: "row" | "col") {
    const handle = e.currentTarget as HTMLElement;
    const box = handle.parentElement!.getBoundingClientRect();
    handle.setPointerCapture(e.pointerId);
    const move = (ev: PointerEvent) =>
      ws.setRatio(sid, dir === "row" ? (ev.clientX - box.left) / box.width : (ev.clientY - box.top) / box.height);
    const up = () => { handle.removeEventListener("pointermove", move); handle.removeEventListener("pointerup", up); };
    handle.addEventListener("pointermove", move);
    handle.addEventListener("pointerup", up);
  }

  const tab = $derived(ws.tabs.find((t) => t.id === tabId));
</script>

{#if isLeaf(node)}
  {@const info = ws.panes[node.id]}
  {@const prof = profiles.find((p) => p.name === info?.target)}
  <section class="leaf" data-pane={node.id} class:focused={multi && tab?.focus === node.id} onpointerdowncapture={() => ws.focus(node.id)}>
    {#if multi}
      <header use:dragSource={{ kind: "pane", id: node.id, label: info?.target ?? "Local" }} title="Drag to move this pane">
        <span class="dot" class:ok={info && !info.ended && !info.connecting} class:warn={info?.connecting} class:err={info?.ended}></span>
        <strong>{info?.target ?? "Local"}</strong>
        {#if prof}<span class="sub">{prof.user}@{prof.host}</span>{/if}
        {#if info?.connecting}<span class="wait">connecting…</span>{/if}
        {#if info?.ended}<span class="gone">disconnected</span>{/if}
        <button class="close" data-nodrag aria-label="Close pane" onclick={() => ws.closePane(node.id)}>&times;</button>
      </header>
    {/if}
    <div class="term" use:attach={node.id}></div>
    {#if info?.connecting && info.target}
      <div class="cover" class:below-header={multi}><ConnectingOverlay target={info.target} {prof} since={info.since} oncancel={() => ws.cancelConnect(node.id)} /></div>
    {/if}
    {#if info?.target && !info.connecting && !info.ended && !info.blocks && !ws.autoBlocksFor(info.target)}
      {#key node.id}<BlocksHint id={node.id} />{/key}
    {/if}
    {#if sb.paneDrop?.id === node.id}
      <div class="dropzone {sb.paneDrop.edge}" class:below-header={multi} aria-hidden="true"><span>{sb.paneDrop.verb ? `${sb.paneDrop.verb} ${sb.paneDrop.edge}` : sb.drag?.kind === "pane" ? (sb.paneDrop.edge === "center" ? "Swap places" : `Move ${sb.paneDrop.edge}`) : sb.paneDrop.edge === "center" ? "Open in a new tab" : `Split ${sb.paneDrop.edge}`}</span></div>
    {/if}
    {#if ws.searchPane === node.id && ws.session(node.id)}<SearchBar session={ws.session(node.id)!} onclose={() => ws.closeSearch()} />{/if}
    {#if info?.ended}
      <div class="ended" role="alert">
        <div><strong>{info.error ? "Could not connect" : "Session ended"}</strong><span>{info.error || `Press Enter to reconnect to ${info.target ?? "local shell"}`}</span></div>
        <button class="btn primary" onclick={() => ws.reconnect(node.id)}>Reconnect</button>
      </div>
    {/if}
  </section>
{:else}
  <div class="split" class:col={node.dir === "col"}>
    <div class="side" style="flex: {node.ratio} 1 0"><PaneView node={node.a} {tabId} {multi} {profiles} /></div>
    <div class="handle" role="separator" aria-orientation={node.dir === "row" ? "vertical" : "horizontal"} onpointerdown={(e) => drag(e, node.sid, node.dir)}></div>
    <div class="side" style="flex: {1 - node.ratio} 1 0"><PaneView node={node.b} {tabId} {multi} {profiles} /></div>
  </div>
{/if}

<style>
  .split { display: flex; width: 100%; height: 100%; min-width: 0; min-height: 0; }
  .split.col { flex-direction: column; }
  .side { min-width: 0; min-height: 0; display: flex; }
  .handle { flex: none; width: 1px; background: var(--line); position: relative; cursor: col-resize; }
  .col > .handle { width: auto; height: 1px; cursor: row-resize; }
  .handle::after { content: ""; position: absolute; inset: 0 -3px; }
  .col > .handle::after { inset: -3px 0; }
  .handle:hover { background: var(--accent); }
  .leaf { flex: 1; min-width: 0; min-height: 0; display: flex; flex-direction: column; position: relative; background: var(--term); }
  .leaf.focused { box-shadow: inset 0 0 0 1px var(--accent-edge); }
  header { cursor: grab; height: 28px; flex: none; display: flex; align-items: center; gap: 8px; padding: 0 0 0 14px; background: var(--bar); border-bottom: 1px solid var(--line); font-size: 11.5px; color: var(--muted); }
  header strong { color: var(--text); font-weight: 500; }
  .dot { width: 7px; height: 7px; }
  .gone { color: var(--err-text); }
  .wait { color: var(--warn); }
  /* Over the terminal (not the pane header). */
  .dropzone { position: absolute; z-index: 4; pointer-events: none; display: grid; place-items: center; background: var(--accent-soft); border: 2px solid var(--accent); inset: 0; }
  .dropzone.below-header { top: 28px; }
  .dropzone.left { right: 50%; }
  .dropzone.right { left: 50%; }
  .dropzone.top { bottom: 50%; }
  .dropzone.top.below-header { bottom: calc(50% - 14px); }
  .dropzone.bottom { top: 50%; }
  .dropzone.bottom.below-header { top: calc(50% + 14px); }
  .dropzone.center { inset: 18%; border-style: dashed; }
  .dropzone span { padding: 4px 10px; background: var(--float); color: var(--accent-text); border: 1px solid var(--floatline); font-size: 12px; font-weight: 600; }
  .cover { position: absolute; inset: 0; z-index: 2; }
  .cover.below-header { top: 28px; }
  .close { all: unset; margin-left: auto; width: 28px; height: 100%; display: grid; place-items: center; font-size: 15px; cursor: pointer; opacity: 0.6; }
  .close:hover { opacity: 1; }
  .term { flex: 1; min-height: 0; min-width: 0; position: relative; overflow: hidden; }
  /* Padding goes on the xterm element itself: FitAddon subtracts that, but not the
     padding of a parent. Block dividers bleed past it (theme.css). */
  .term :global(.pane-term) { position: absolute; inset: 0; background: var(--term); }
  .term :global(.pane-term .xterm) { padding: 14px 24px 18px; }
  .ended { position: absolute; left: 50%; bottom: 28px; transform: translateX(-50%); display: flex; align-items: center; gap: 14px; padding: 10px 14px 10px 16px; background: var(--active); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 10px 30px var(--shadow); white-space: nowrap; z-index: 3; }
  .ended div { display: grid; gap: 1px; }
  .ended span { font-size: 11.5px; color: var(--muted); }
</style>
