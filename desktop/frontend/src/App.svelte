<script lang="ts">
  import { onMount } from "svelte";
  import { openSession, type Session } from "./terminal";
  import { runBench } from "./bench";
  import { blankProfile } from "./lib";
  import Sidebar from "./Sidebar.svelte";
  import Palette from "./Palette.svelte";
  import ProfileForm from "./ProfileForm.svelte";
  import { ProfilesService, TerminalService, type Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";

  let host: HTMLElement;
  let profiles = $state<Profile[]>([]);
  let active = $state<Profile | null>(null);
  let session: Session | undefined;
  let started = $state(false);
  let ended = $state(false);
  let banner = $state("");
  let palette = $state(false);
  let form = $state<{ initial: Profile; oldName: string } | null>(null);
  let confirmDelete = $state<Profile | null>(null);

  const folders = $derived([...new Set(profiles.map((p) => p.folder).filter(Boolean))].sort());

  async function refresh() {
    try {
      profiles = (await ProfilesService.List()) ?? [];
    } catch (e) {
      banner = String(e);
    }
  }

  async function connect(p: Profile | null) {
    palette = false;
    session?.dispose();
    host.replaceChildren();
    active = p;
    ended = false;
    banner = "";
    try {
      session = await openSession(host, p ? p.name : "local");
      session.ws.addEventListener("close", () => (ended = true));
      session.term.focus();
      started = true;
    } catch (e) {
      banner = String(e);
    }
  }

  async function save(prof: Profile) {
    await ProfilesService.Save(form!.oldName, prof);
    if (active && active.name === form!.oldName) active = { ...prof };
    form = null;
    await refresh();
  }

  async function remove(p: Profile) {
    confirmDelete = null;
    try {
      await ProfilesService.Delete(p.name);
      if (active?.name === p.name) active = null;
      await refresh();
    } catch (e) {
      banner = String(e);
    }
  }

  function keydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      palette = !palette;
    } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "n") {
      e.preventDefault();
      form = { initial: blankProfile(), oldName: "" };
    }
  }

  onMount(async () => {
    const mode = await TerminalService.BenchMode();
    if (mode) return runBench(host, mode);
    await refresh();
  });
</script>

<svelte:window onkeydown={keydown} />

<div class="shell">
  <Sidebar
    {profiles}
    active={active?.name ?? ""}
    onconnect={connect}
    onnew={() => (form = { initial: blankProfile(), oldName: "" })}
    onedit={(p) => (form = { initial: p, oldName: p.name })}
    ondelete={(p) => (confirmDelete = p)}
    onpalette={() => (palette = true)}
  />
  <main>
    <header>
      {#if active}
        <strong>{active.name}</strong><span class="sub">{active.user}@{active.host}</span>
        {#each active.tags as t}<span class="chip">{t}</span>{/each}
      {:else}
        <strong>Local shell</strong>
      {/if}
      <span class="spacer"></span>
      {#if ended}<span class="ended">Session ended</span><button class="btn" onclick={() => connect(active)}>Reconnect</button>{/if}
      <button class="btn" onclick={() => connect(null)}>Local shell</button>
    </header>
    {#if banner}<p class="banner" role="alert">{banner}</p>{/if}
    <div class="term" bind:this={host}></div>
    {#if !started && !banner}
      <div class="welcome">
        <h1>Relay</h1>
        <p>Pick a connection on the left, or press <kbd>⌘K</kbd> to jump to one.</p>
      </div>
    {/if}
  </main>
</div>

{#if palette}<Palette {profiles} onpick={connect} onclose={() => (palette = false)} />{/if}
{#if form}
  <ProfileForm initial={form.initial} isNew={form.oldName === ""} others={profiles.map((p) => p.name).filter((n) => n !== form!.oldName)} {folders} onsave={save} oncancel={() => (form = null)} />
{/if}
{#if confirmDelete}
  <div class="scrim" role="presentation" onkeydown={(e) => e.key === "Escape" && (confirmDelete = null)}>
    <div class="confirm" role="alertdialog" aria-label="Delete connection">
      <p>Delete <strong>{confirmDelete.name}</strong>? This removes it from connections.toml and your SSH config.</p>
      <div class="buttons">
        <button class="btn" onclick={() => (confirmDelete = null)}>Cancel</button>
        <button class="btn danger" onclick={() => remove(confirmDelete!)}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<style>
  .shell { display: flex; height: 100%; }
  main { flex: 1; min-width: 0; display: flex; flex-direction: column; position: relative; }
  header { display: flex; align-items: center; gap: 8px; padding: 8px 14px; border-bottom: 1px solid var(--border); background: var(--panel); min-height: 44px; }
  .sub { color: var(--muted); font: 12px var(--mono); }
  .spacer { flex: 1; }
  .ended { color: var(--err); }
  .term { flex: 1; min-height: 0; padding: 6px 8px 0; background: var(--bg); }
  .welcome { position: absolute; inset: 44px 0 0 0; display: grid; place-content: center; text-align: center; color: var(--muted); pointer-events: none; }
  .welcome h1 { color: var(--text); margin: 0 0 6px; font-weight: 600; }
  kbd { font: 11px var(--mono); border: 1px solid var(--border); border-radius: 4px; padding: 0 4px; }
  .banner { margin: 0; padding: 8px 14px; color: var(--err); background: var(--err-bg); border-bottom: 1px solid var(--err); }
  .scrim { position: fixed; inset: 0; background: rgba(5, 9, 15, 0.6); display: flex; align-items: center; justify-content: center; z-index: 10; }
  .confirm { width: min(420px, 92vw); background: var(--panel); border: 1px solid var(--border); border-radius: 12px; padding: 18px 20px; }
  .confirm p { margin: 0 0 16px; }
  .buttons { display: flex; justify-content: flex-end; gap: 8px; }
</style>
