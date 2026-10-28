<script lang="ts">
  import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
  import { parseTags } from "./lib";

  let { initial, isNew, others, folders, onsave, oncancel }: {
    initial: Profile;
    isNew: boolean;
    others: string[];
    folders: string[];
    onsave: (p: Profile) => Promise<void>;
    oncancel: () => void;
  } = $props();

  let p = $state<Profile>(structuredClone($state.snapshot(initial)));
  let tags = $state((initial.tags ?? []).join(", "));
  let error = $state("");
  let busy = $state(false);

  async function submit(e: Event) {
    e.preventDefault();
    busy = true;
    error = "";
    try {
      await onsave({ ...p, port: Number(p.port), tags: parseTags(tags), key: p.auth === "key" ? p.key : "" } as Profile);
    } catch (err) {
      error = (err as Error).message ?? String(err);
    } finally {
      busy = false;
    }
  }
  const addTunnel = () => (p.tunnels ??= []).push({ type: "local", listen: "", target: "" });
</script>

<div class="scrim" role="presentation" onkeydown={(e) => e.key === "Escape" && oncancel()}>
  <form role="dialog" aria-label={isNew ? "New connection" : "Edit connection"} onsubmit={submit}>
    <h2>{isNew ? "New connection" : `Edit ${initial.name}`}</h2>
    <div class="grid">
      <label>Name<input bind:value={p.name} required autofocus placeholder="web-1" /></label>
      <label>Folder<input bind:value={p.folder} list="folders" placeholder="prod/eu" /></label>
      <label>Host<input bind:value={p.host} required placeholder="10.0.0.5 or example.com" /></label>
      <label>Port<input type="number" bind:value={p.port} min="1" max="65535" required /></label>
      <label>User<input bind:value={p.user} required /></label>
      <label>Auth
        <select bind:value={p.auth}>
          <option value="agent">SSH agent</option>
          <option value="key">Key file</option>
          <option value="password">Password (asked by ssh)</option>
        </select>
      </label>
      {#if p.auth === "key"}
        <label class="wide">Key<input bind:value={p.key} required placeholder="id_ed25519 or ~/.ssh/id_ed25519" /></label>
      {/if}
      <label>Tags<input bind:value={tags} placeholder="db, critical" /></label>
      <label>Jump host
        <select bind:value={p.jump}>
          <option value="">None</option>
          {#each others as o}<option value={o}>{o}</option>{/each}
        </select>
      </label>
      <label class="check wide"><input type="checkbox" bind:checked={p.forwardAgent} /> Forward SSH agent</label>
    </div>

    <fieldset>
      <legend>Port forwarding</legend>
      {#each p.tunnels ?? [] as t, i}
        <div class="tunnel">
          <select bind:value={t.type} aria-label="Tunnel type">
            <option value="local">Local</option><option value="remote">Remote</option><option value="dynamic">SOCKS</option>
          </select>
          <input bind:value={t.listen} placeholder={t.type === "remote" ? "remote port" : "local port"} aria-label="Listen" required />
          {#if t.type !== "dynamic"}<input bind:value={t.target} placeholder="host:port" aria-label="Target" required />{/if}
          <button type="button" class="btn" aria-label="Remove tunnel" onclick={() => p.tunnels?.splice(i, 1)}>✕</button>
        </div>
      {/each}
      <button type="button" class="btn" onclick={addTunnel}>+ Add tunnel</button>
    </fieldset>

    <datalist id="folders">{#each folders as f}<option value={f}></option>{/each}</datalist>
    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <div class="buttons">
      <button type="button" class="btn" onclick={oncancel}>Cancel</button>
      <button class="btn primary" disabled={busy}>{isNew ? "Add" : "Save"}</button>
    </div>
  </form>
</div>

<style>
  .scrim { position: fixed; inset: 0; background: rgba(5, 9, 15, 0.6); display: flex; align-items: center; justify-content: center; z-index: 10; overflow: auto; padding: 20px; }
  form { width: min(560px, 100%); background: var(--panel); border: 1px solid var(--border); border-radius: 12px; padding: 18px 20px; box-shadow: 0 20px 60px rgba(0,0,0,.5); }
  h2 { margin: 0 0 14px; font-size: 16px; }
  .grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px 12px; }
  label { display: grid; gap: 4px; color: var(--muted); font-size: 12px; }
  label input, label select { color: var(--text); font-size: 13px; }
  .wide { grid-column: 1 / -1; }
  .check { display: flex; align-items: center; gap: 8px; color: var(--text); font-size: 13px; }
  fieldset { border: 1px solid var(--border); border-radius: var(--radius); margin: 14px 0 0; display: grid; gap: 8px; }
  legend { color: var(--muted); font-size: 12px; padding: 0 6px; }
  .tunnel { display: flex; gap: 6px; }
  .tunnel input { flex: 1; }
  .error { color: var(--err); background: var(--err-bg); border: 1px solid var(--err); border-radius: 6px; padding: 8px 10px; margin: 12px 0 0; }
  .buttons { display: flex; justify-content: flex-end; gap: 8px; margin-top: 16px; }
</style>
