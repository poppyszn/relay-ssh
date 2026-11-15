<script lang="ts">
  import EmptyState from "./EmptyState.svelte";
  import ForwardDiagram from "./ForwardDiagram.svelte";
  import Icon from "./Icon.svelte";
  import Modal from "./Modal.svelte";
  import { ForwardsService, ProfilesService, type ForwardInfo } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

  let { target, name = "", embedded = false, onclose, onopen, onchanged }: { target: string; name?: string; embedded?: boolean; onclose: () => void; onopen: () => void; onchanged: () => void } = $props();

  let connected = $state(false);
  let message = $state("");
  let forwards = $state<ForwardInfo[]>([]);
  let loaded = $state(false);
  let error = $state("");
  let note = $state("");
  let busy = $state(false);

  let kind = $state<"local" | "remote" | "dynamic">("local");
  let listen = $state("");
  let dest = $state("");
  let listenInput = $state<HTMLInputElement | null>(null);
  // Clicking a row shows that forward in the diagram; typing in the form takes over again.
  let viewing = $state<ForwardInfo | null>(null);
  const shown = $derived(viewing ?? { type: kind, listen, target: dest });

  const label = { local: "Local", remote: "Remote", dynamic: "SOCKS" } as const;
  const msg = (e: unknown) => (e as Error)?.message ?? String(e);

  async function refresh() {
    try {
      const s = await ForwardsService.Status(target);
      connected = s.connected; message = s.message; forwards = s.forwards ?? [];
    } catch (e) { error = msg(e); } finally { loaded = true; }
  }
  async function run(fn: () => Promise<void>, ok: string) {
    busy = true; error = ""; note = "";
    try { await fn(); note = ok; await refresh(); } catch (e) { error = msg(e); } finally { busy = false; }
  }
  const add = () => run(async () => { await ForwardsService.Add(target, kind, listen.trim(), dest.trim()); listen = ""; dest = ""; }, "Forward added.");
  const stop = (f: ForwardInfo) => run(() => ForwardsService.Cancel(target, f.type, f.listen, f.target), "Forward stopped.");
  const start = (f: ForwardInfo) => run(() => ForwardsService.Add(target, f.type, f.listen, f.target), "Forward started.");
  // Makes a runtime forward part of the saved connection, so it starts with every login.
  const save = (f: ForwardInfo) => run(async () => {
    const p = (await ProfilesService.List())?.find((x) => x.id === target);
    if (!p) throw new Error("Connection not found.");
    p.tunnels = [...(p.tunnels ?? []), { type: f.type, listen: f.listen, target: f.target }];
    await ProfilesService.Save(target, p);
    onchanged();
  }, "Saved to the connection. It will start automatically next time.");

  $effect(() => {
    refresh();
    const t = setInterval(refresh, 3000);
    return () => clearInterval(t);
  });
</script>

{#snippet body()}
  {#if loaded && !connected}
    <EmptyState icon="forwards" title="This connection is not open" text={`${message ? message + " " : ""}Forwards ride on the connection of its terminal tab, so open it first.`}>
      <button type="button" class="btn primary" data-autofocus onclick={onopen}>Open {name || target} in a tab</button>
    </EmptyState>
  {:else if embedded}
    <div class="drawer-scroll">
      <ForwardDiagram compact kind={shown.type as "local" | "remote" | "dynamic"} listen={shown.listen} dest={shown.target} server={name || target} />

      {#if error}<div class="notice bad" role="alert">{error}</div>{/if}
      {#if note && !error}<div class="notice good">{note}</div>{/if}

      {#if forwards.length}
        <div class="card-list" role="grid" aria-label="Forwards">
          {#each forwards as f (f.type + f.listen + f.target)}
            <div class="fwd-card row-hover" class:row-hl={viewing === f} role="row" aria-selected={viewing === f} tabindex="0" onclick={() => (viewing = f)} onkeydown={(e) => e.target === e.currentTarget && e.key === "Enter" && (viewing = f)}>
              <div class="fwd-top">
                <div class="fwd-left">
                  <span class="chip">{label[f.type as keyof typeof label]}</span>
                  <span class="dot" class:ok={f.listening} role="img" aria-label={f.listening ? "Listening" : "Not listening"}></span>
                  <span class="mono strong truncate" title={f.listen}>{f.listen}</span>
                </div>
                <div class="acts">
                  {#if f.source === "runtime"}<button type="button" class="btn sm ghost" onclick={(e) => { e.stopPropagation(); save(f); }} disabled={busy}>Save</button>{/if}
                  {#if f.listening}
                    <button type="button" class="btn sm ghost danger" onclick={(e) => { e.stopPropagation(); stop(f); }} disabled={busy}>Stop</button>
                  {:else}
                    <button type="button" class="btn sm" onclick={(e) => { e.stopPropagation(); start(f); }} disabled={busy}>Start</button>
                  {/if}
                </div>
              </div>
              <div class="fwd-sub">
                <span class="mono truncate" title={f.type === "dynamic" ? "Dynamic SOCKS5 proxy" : f.target}>{f.type === "dynamic" ? "Dynamic SOCKS5 proxy" : `→ ${f.target}`}</span>
                <span class="fwd-src">{f.source === "profile" ? "Saved" : "Runtime"}</span>
              </div>
            </div>
          {/each}
        </div>
      {/if}

      <form class="add-card" onsubmit={(e) => { e.preventDefault(); add(); }}>
        <div class="add-head">
          <div class="add-title">Add Forward</div>
          <div class="seg seg-3" role="group" aria-label="Tunnel type">
            <button type="button" aria-pressed={kind === "local"} onclick={() => { kind = "local"; viewing = null; }}>Local</button>
            <button type="button" aria-pressed={kind === "remote"} onclick={() => { kind = "remote"; viewing = null; }}>Remote</button>
            <button type="button" aria-pressed={kind === "dynamic"} onclick={() => { kind = "dynamic"; viewing = null; }}>SOCKS</button>
          </div>
        </div>
        <div class="add-inputs" class:duo={kind !== "dynamic"}>
          <input class="mono compact-in" bind:this={listenInput} bind:value={listen} oninput={() => (viewing = null)} placeholder={kind === "remote" ? "Remote (:8080)" : "Listen (:5432)"} required />
          {#if kind !== "dynamic"}
            <input class="mono compact-in" bind:value={dest} oninput={() => (viewing = null)} placeholder="Dest (127.0.0.1:3000)" required />
          {/if}
        </div>
        <button type="submit" class="btn sm primary full" disabled={busy}>Start Forward</button>
      </form>
    </div>
  {:else}
    <div class="grid" role="grid" aria-label="Forwards">
      {#if forwards.length}
        <div class="row head" role="row"><span class="label" role="columnheader">Type</span><span class="label" role="columnheader">Listen</span><span class="label" role="columnheader">Target</span><span class="label" role="columnheader">From</span><span role="columnheader"></span></div>
        {#each forwards as f (f.type + f.listen + f.target)}
          <div class="row pick row-hover" class:row-hl={viewing === f} role="row" aria-selected={viewing === f} tabindex="0" onclick={() => (viewing = f)} onkeydown={(e) => e.target === e.currentTarget && e.key === "Enter" && (viewing = f)}>
            <span role="gridcell"><span class="chip">{label[f.type as keyof typeof label]}</span></span>
            <span class="mono cell" role="gridcell"><span class="dot" class:ok={f.listening} role="img" aria-label={f.listening ? "Listening" : "Not listening"}></span>{f.listen}</span>
            <span class="mono" role="gridcell">{f.type === "dynamic" ? "SOCKS proxy" : f.target}</span>
            <span class="dim" role="gridcell">{f.source === "profile" ? "Saved" : "Added now"}</span>
            <span class="acts" role="gridcell">
              {#if f.source === "runtime"}<button type="button" class="btn sm" onclick={() => save(f)} disabled={busy} title="Keep this forward for every login">Save</button>{/if}
              {#if f.listening}<button type="button" class="btn sm" onclick={() => stop(f)} disabled={busy}>Stop</button>{:else}<button type="button" class="btn sm" onclick={() => start(f)} disabled={busy}>Start</button>{/if}
            </span>
          </div>
        {/each}
      {:else if loaded}
        <EmptyState icon="forwards" title="No forwards yet" text="Forward a port through this connection, or open a SOCKS proxy.">
          <button type="button" class="btn sm" onclick={() => listenInput?.focus()}>Add a forward</button>
        </EmptyState>
      {/if}
    </div>

    {#if error}<div class="strip"><div class="notice bad" role="alert">{error}</div></div>{/if}
    {#if note && !error}<div class="strip"><div class="notice good">{note}</div></div>{/if}

    <form class="add" onsubmit={(e) => { e.preventDefault(); add(); }}>
      <ForwardDiagram kind={shown.type as "local" | "remote" | "dynamic"} listen={shown.listen} dest={shown.target} server={name || target} />
      <h3 class="label">Add a forward</h3>
      <div class="fields">
        <label class="field"><span>Type</span><select bind:value={kind} onchange={() => (viewing = null)}><option value="local">Local</option><option value="remote">Remote</option><option value="dynamic">SOCKS</option></select></label>
        <label class="field"><span>{kind === "remote" ? "Remote port" : "Listen port"}</span><input class="mono" bind:this={listenInput} bind:value={listen} oninput={() => (viewing = null)} placeholder="8080" required /></label>
        {#if kind !== "dynamic"}
          <label class="field"><span>Target</span><input class="mono" bind:value={dest} oninput={() => (viewing = null)} placeholder="host:port" required /></label>
        {:else}
          <label class="field"><span>Target</span><input class="mono" value="" placeholder="Not needed for SOCKS" disabled /></label>
        {/if}
        <button type="submit" class="btn primary" disabled={busy}>Add</button>
      </div>
    </form>
  {/if}
{/snippet}

{#if embedded}
  <div class="drawer-panel">
    <div class="drawer-head">
      <div class="drawer-title">
        <span class="title-strong">Port Forwards · {name || target}</span>
        {#if loaded}
          <span class="state"><span class="dot" class:ok={connected} class:live={connected}></span>{connected ? "SSH Multiplexer Connected" : "Not connected"}</span>
        {/if}
      </div>
      <button type="button" class="icon-btn dense" aria-label="Close drawer" onclick={onclose}><Icon name="x" size={14} /></button>
    </div>
    {@render body()}
  </div>
{:else}
  <Modal title="Port forwards" subtitle={name || target} size="lg" flush {onclose}>
    {#snippet extra()}
      {#if loaded}<span class="state"><span class="dot" class:ok={connected} class:live={connected}></span>{connected ? "Connected" : "Not connected"}</span>{/if}
    {/snippet}
    {@render body()}
  </Modal>
{/if}

<style>
  .drawer-panel { flex: 1; min-width: 0; min-height: 0; display: flex; flex-direction: column; }
  .drawer-head { display: flex; align-items: center; justify-content: space-between; gap: var(--s2); padding: 10px 14px; border-bottom: 1px solid var(--hairline); }
  .drawer-title { display: grid; gap: 2px; min-width: 0; }
  .title-strong { font-size: var(--t-sm); font-weight: 600; color: var(--strong); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .drawer-scroll { flex: 1; min-width: 0; min-height: 0; overflow-y: auto; overflow-x: hidden; padding: 10px 12px; display: grid; align-content: start; gap: 8px; }
  .card-list { display: grid; gap: 6px; min-width: 0; }
  .fwd-card { padding: 8px 10px; border-radius: 10px; background: var(--hover); border: 1px solid var(--hairline); display: grid; gap: 4px; cursor: default; min-width: 0; }
  .fwd-top { display: flex; align-items: center; justify-content: space-between; gap: 8px; min-width: 0; }
  .fwd-left { display: flex; align-items: center; gap: 6px; min-width: 0; flex: 1; }
  .strong { color: var(--strong); font-size: var(--t-xs); }
  .truncate { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .fwd-sub { display: flex; align-items: center; justify-content: space-between; gap: 8px; font-size: 11px; color: var(--muted); min-width: 0; }
  .fwd-src { flex: none; }
  .add-card { padding: 10px; border-radius: 12px; background: var(--field); border: 1px solid var(--hairline); display: grid; gap: 8px; min-width: 0; }
  .add-head { display: flex; align-items: center; justify-content: space-between; gap: 8px; min-width: 0; }
  .add-title { font-size: var(--t-xs); font-weight: 600; color: var(--text2); white-space: nowrap; }
  .seg-3 { display: inline-grid; grid-template-columns: repeat(3, 1fr); flex: none; }
  .seg-3 > button { height: 24px; padding: 0 8px; justify-content: center; font-size: 11px; }
  .add-inputs { display: grid; grid-template-columns: minmax(0, 1fr); gap: 6px; min-width: 0; }
  .add-inputs.duo { grid-template-columns: minmax(0, 0.85fr) minmax(0, 1.35fr); }
  .compact-in { width: 100%; min-width: 0; height: 30px; font-size: var(--t-xs); }
  .full { width: 100%; }
  .state { display: flex; align-items: center; gap: var(--s2); font-size: var(--t-xs); color: var(--muted); }
  .grid { max-height: 280px; min-height: 0; overflow: auto; }
  .row { display: grid; grid-template-columns: 90px 1fr 1.3fr 90px 150px; align-items: center; gap: var(--s3); padding: 0 var(--s5); height: 38px; border-bottom: 1px solid var(--hairline); font-size: var(--t-md); }
  .row.head { position: sticky; top: 0; z-index: 1; height: 32px; background: var(--term); font-size: var(--t-xs); }
  .cell { display: flex; align-items: center; gap: var(--s2); }
  .pick { cursor: default; outline-offset: -2px; }
  .dim { color: var(--muted); font-size: var(--t-sm); }
  .acts { display: flex; gap: var(--s2); justify-content: flex-end; }
  .strip { padding: var(--s3) var(--s5) 0; }
  .add { display: grid; gap: var(--s3); padding: var(--s5); }
  h3 { margin: 0; }
  .fields { display: grid; grid-template-columns: 120px 1fr 1.4fr auto; gap: var(--s2); align-items: end; }
</style>
