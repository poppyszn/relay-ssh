<script lang="ts">
  import ForwardDiagram from "./ForwardDiagram.svelte";
  import { ForwardsService, ProfilesService, type ForwardInfo } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

  let { target, onclose, onopen, onchanged }: { target: string; onclose: () => void; onopen: () => void; onchanged: () => void } = $props();

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
    const p = (await ProfilesService.List())?.find((x) => x.name === target);
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

<div class="scrim" role="presentation" onclick={onclose} onkeydown={(e) => e.key === "Escape" && onclose()}>
  <div class="panel" role="dialog" aria-label="Port forwards" tabindex="-1" onclick={(e) => e.stopPropagation()}>
    <header>
      <h2>Port forwards <span class="host mono">{target}</span>
        {#if loaded}<span class="state"><span class="dot" class:ok={connected}></span>{connected ? "Connected" : "Not connected"}</span>{/if}</h2>
      <button class="x" aria-label="Close" onclick={onclose}>&times;</button>
    </header>

    {#if loaded && !connected}
      <div class="empty">
        <p>{message || "This connection is not open."}</p>
        <p class="dim">Forwards ride on the connection of its terminal tab, so open it first.</p>
        <button class="btn primary" onclick={onopen}>Open {target} in a tab</button>
      </div>
    {:else}
      <div class="body">
        {#if forwards.length}
          <div class="row head"><span>Type</span><span>Listen</span><span>Target</span><span>From</span><span></span></div>
          {#each forwards as f (f.type + f.listen + f.target)}
            <div class="row pick" class:viewing={viewing === f} role="row" tabindex="0" onclick={() => (viewing = f)} onkeydown={(e) => e.key === "Enter" && (viewing = f)}>
              <span><span class="chip">{label[f.type as keyof typeof label]}</span></span>
              <span class="mono"><span class="dot" class:ok={f.listening}></span> {f.listen}</span>
              <span class="mono">{f.type === "dynamic" ? "SOCKS proxy" : f.target}</span>
              <span class="dim">{f.source === "profile" ? "Saved" : "Added now"}</span>
              <span class="acts">
                {#if f.source === "runtime"}<button class="btn sm" onclick={() => save(f)} disabled={busy} title="Keep this forward for every login">Save</button>{/if}
                {#if f.listening}<button class="btn sm" onclick={() => stop(f)} disabled={busy}>Stop</button>{:else}<button class="btn sm" onclick={() => start(f)} disabled={busy}>Start</button>{/if}
              </span>
            </div>
          {/each}
        {:else if loaded}
          <p class="none">No forwards on this connection yet.</p>
        {/if}
      </div>

      <form class="add" onsubmit={(e) => { e.preventDefault(); add(); }}>
        <ForwardDiagram kind={shown.type as "local" | "remote" | "dynamic"} listen={shown.listen} dest={shown.target} server={target} />
        <div class="label">Add a forward</div>
        <div class="fields">
          <select bind:value={kind} aria-label="Type" onchange={() => (viewing = null)}><option value="local">Local</option><option value="remote">Remote</option><option value="dynamic">SOCKS</option></select>
          <input class="mono" bind:value={listen} oninput={() => (viewing = null)} placeholder={kind === "remote" ? "remote port" : "local port"} aria-label="Listen port" required />
          {#if kind !== "dynamic"}<input class="mono" bind:value={dest} oninput={() => (viewing = null)} placeholder="host:port" aria-label="Target" required />{:else}<span class="na">no target needed</span>{/if}
          <button class="btn primary" disabled={busy}>Add</button>
        </div>
      </form>
    {/if}
    {#if error}<p class="msg bad" role="alert">{error}</p>{/if}
    {#if note && !error}<p class="msg good">{note}</p>{/if}
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; background: var(--scrim); z-index: 25; display: flex; align-items: center; justify-content: center; padding: 16px; }
  .panel { width: min(780px, 100%); max-height: 100%; display: flex; flex-direction: column; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 30px 80px var(--shadow); overflow: hidden; }
  header { height: 52px; flex: none; display: flex; align-items: center; justify-content: space-between; padding: 0 14px 0 22px; border-bottom: 1px solid var(--line2); }
  h2 { margin: 0; font-size: 15px; font-weight: 600; color: var(--strong); display: flex; align-items: center; gap: 10px; }
  .host { font-size: 12px; font-weight: 400; color: var(--muted); background: var(--line); padding: 1px 8px; border-radius: 3px; }
  .state { display: flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 400; color: var(--muted); }
  .x { width: 30px; height: 30px; border: 0; background: none; border-radius: 4px; color: var(--muted); font-size: 18px; }
  .x:hover { background: var(--line); color: var(--strong); }
  .empty { padding: 36px 28px 40px; text-align: center; display: grid; gap: 6px; justify-items: center; }
  .empty p { margin: 0; color: var(--text2); }
  .dim { color: var(--muted); font-size: 12px; }
  .body { min-height: 120px; max-height: 340px; overflow: auto; }
  .none { margin: 0; padding: 34px 22px; color: var(--muted); text-align: center; }
  .row { display: grid; grid-template-columns: 90px 1fr 1.3fr 90px 150px; align-items: center; gap: 10px; padding: 0 22px; height: 44px; border-bottom: 1px solid var(--line); }
  .row.head { height: 34px; background: var(--bar); font-size: 10.5px; font-weight: 600; letter-spacing: 0.1em; text-transform: uppercase; color: var(--muted); position: sticky; top: 0; }
  .row .dot { width: 7px; height: 7px; margin-right: 4px; display: inline-block; }
  .pick { cursor: default; }
  .pick:hover { background: var(--hover); }
  .pick.viewing { background: var(--active); box-shadow: inset 3px 0 var(--accent); }
  .acts { display: flex; gap: 6px; justify-content: flex-end; }
  .btn.sm { padding: 4px 12px; font-size: 12px; }
  .add { padding: 16px 22px 18px; border-top: 1px solid var(--line2); background: var(--foot); display: grid; gap: 10px; }
  .fields { display: grid; grid-template-columns: 110px 1fr 1.4fr auto; gap: 8px; align-items: center; }
  .fields select { font-family: var(--sans); }
  .na { font-size: 12px; color: var(--dim); }
  .help { margin: 0; font-size: 12px; color: var(--muted); }
  .msg { margin: 0; padding: 9px 22px; font-size: 12.5px; border-top: 1px solid var(--line2); }
  .msg.bad { color: var(--err-text); background: var(--err-bg); }
  .msg.good { color: var(--ok); }
</style>
