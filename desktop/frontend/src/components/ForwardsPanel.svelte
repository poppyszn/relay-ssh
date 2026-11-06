<script lang="ts">
  import EmptyState from "./EmptyState.svelte";
  import ForwardDiagram from "./ForwardDiagram.svelte";
  import Modal from "./Modal.svelte";
  import { ForwardsService, ProfilesService, type ForwardInfo } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

  let { target, name = "", onclose, onopen, onchanged }: { target: string; name?: string; onclose: () => void; onopen: () => void; onchanged: () => void } = $props();

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

<Modal title="Port forwards" subtitle={name || target} size="lg" flush {onclose}>
  {#snippet extra()}
    {#if loaded}<span class="state"><span class="dot" class:ok={connected} class:live={connected}></span>{connected ? "Connected" : "Not connected"}</span>{/if}
  {/snippet}

  {#if loaded && !connected}
    <EmptyState icon="forwards" title="This connection is not open" text={`${message ? message + " " : ""}Forwards ride on the connection of its terminal tab, so open it first.`}>
      <button type="button" class="btn primary" data-autofocus onclick={onopen}>Open {name || target} in a tab</button>
    </EmptyState>
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
</Modal>

<style>
  .state { display: flex; align-items: center; gap: var(--s2); font-size: var(--t-sm); color: var(--muted); }
  .grid { max-height: 280px; min-height: 0; overflow: auto; }
  .row { display: grid; grid-template-columns: 90px 1fr 1.3fr 90px 150px; align-items: center; gap: var(--s3); padding: 0 var(--s5); height: 36px; border-bottom: 1px solid var(--line); }
  .row.head { position: sticky; top: 0; z-index: 1; height: 32px; background: var(--bar); }
  .cell { display: flex; align-items: center; gap: var(--s2); }
  .pick { cursor: default; outline-offset: -2px; }
  .dim { color: var(--muted); font-size: var(--t-sm); }
  .acts { display: flex; gap: var(--s2); justify-content: flex-end; }
  .strip { padding: var(--s3) var(--s5) 0; }
  .add { display: grid; gap: var(--s3); padding: var(--s5); }
  h3 { margin: 0; }
  .fields { display: grid; grid-template-columns: 120px 1fr 1.4fr auto; gap: var(--s2); align-items: end; }
</style>
