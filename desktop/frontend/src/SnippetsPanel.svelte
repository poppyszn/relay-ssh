<script lang="ts">
  import { SnippetsService, type Snippet } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { snippetPreview } from "./lib";

  let { into, onuse, onclose }: {
    /** The pane snippets go into, for the header: "" when there is none. */
    into: string;
    /** Types a snippet into the focused pane; resolves false when there is nowhere to type. */
    onuse: (command: string, run: boolean) => boolean;
    onclose: () => void;
  } = $props();

  let list = $state<Snippet[]>([]);
  let query = $state("");
  let loaded = $state(false);
  let error = $state("");
  /** The snippet being edited ("" for a new one), or null when the form is closed. */
  let editing = $state<string | null>(null);
  let name = $state("");
  let command = $state("");
  let confirmDelete = $state("");

  const shown = $derived(list.filter((s) => `${s.name}\n${s.command}`.toLowerCase().includes(query.trim().toLowerCase())));

  async function load() {
    try {
      list = (await SnippetsService.List()) ?? [];
      error = "";
    } catch (e) {
      error = String((e as Error)?.message ?? e);
    }
    loaded = true;
  }
  load();

  function edit(s?: Snippet) {
    editing = s?.name ?? "";
    name = s?.name ?? "";
    command = s?.command ?? "";
    error = "";
  }
  async function save() {
    try {
      await SnippetsService.Save(editing ?? "", { name, command });
      editing = null;
      await load();
    } catch (e) {
      error = String((e as Error)?.message ?? e);
    }
  }
  async function remove(n: string) {
    confirmDelete = "";
    try {
      await SnippetsService.Delete(n);
      await load();
    } catch (e) {
      error = String((e as Error)?.message ?? e);
    }
  }
  function use(s: Snippet, run: boolean) {
    if (onuse(s.command, run)) onclose();
    else error = "Open a terminal first: there is nowhere to type this.";
  }
  const focus = (el: HTMLElement) => el.focus();
</script>

<div class="scrim" role="presentation" onclick={onclose} onkeydown={(e) => e.key === "Escape" && (editing !== null ? (editing = null) : onclose())}>
  <div class="panel" role="dialog" aria-label="Snippets" tabindex="-1" onclick={(e) => e.stopPropagation()}>
    <header>
      <h2>Snippets {#if into}<span class="into mono">into {into}</span>{/if}</h2>
      <button class="x" aria-label="Close" onclick={onclose}>&times;</button>
    </header>

    {#if editing === null}
      <div class="bar">
        <input use:focus bind:value={query} placeholder="Search snippets" aria-label="Search snippets" />
        <button class="btn primary" onclick={() => edit()}>New snippet</button>
      </div>
      {#if error}<p class="note bad" role="alert">{error}</p>{/if}
      <div class="body">
        {#each shown as s (s.name)}
          <div class="item">
            <div class="txt">
              <b>{s.name}</b>
              <span class="mono cmd">{snippetPreview(s.command)}</span>
            </div>
            {#if confirmDelete === s.name}
              <div class="acts"><span class="ask">Delete?</span><button class="btn sm danger-fill" onclick={() => remove(s.name)}>Delete</button><button class="btn sm" onclick={() => (confirmDelete = "")}>Keep</button></div>
            {:else}
              <div class="acts">
                <button class="btn sm" title="Type it without pressing Enter, so you can edit it first" onclick={() => use(s, false)}>Insert</button>
                <button class="btn sm primary" title="Type it and press Enter" onclick={() => use(s, true)}>Run</button>
                <button class="ico" aria-label="Edit {s.name}" title="Edit" onclick={() => edit(s)}>
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z" /></svg>
                </button>
                <button class="ico" aria-label="Delete {s.name}" title="Delete" onclick={() => (confirmDelete = s.name)}>
                  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3" /></svg>
                </button>
              </div>
            {/if}
          </div>
        {:else}
          <p class="none">{!loaded ? "Loading…" : list.length ? "No snippets match." : "No snippets yet. Save a command you type often, then run it in any terminal."}</p>
        {/each}
      </div>
      <footer>Saved in <span class="mono">~/.config/sssh/snippets.toml</span>. Insert types it for you to edit; Run also presses Enter.</footer>
    {:else}
      <form class="form" onsubmit={(e) => { e.preventDefault(); save(); }}>
        <label>Name<input use:focus bind:value={name} placeholder="Tail the nginx log" /></label>
        <label>Command<textarea bind:value={command} rows="6" spellcheck="false" placeholder="tail -f /var/log/nginx/error.log"></textarea></label>
        {#if error}<p class="note bad" role="alert">{error}</p>{/if}
        <div class="end">
          <button type="button" class="btn" onclick={() => { editing = null; error = ""; }}>Cancel</button>
          <button type="submit" class="btn primary">{editing ? "Save changes" : "Add snippet"}</button>
        </div>
      </form>
    {/if}
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; background: var(--scrim); z-index: 25; display: flex; align-items: center; justify-content: center; padding: 16px; }
  .panel { width: min(680px, 100%); max-height: 100%; display: flex; flex-direction: column; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 30px 80px var(--shadow); overflow: hidden; }
  header { height: 52px; flex: none; display: flex; align-items: center; justify-content: space-between; padding: 0 14px 0 22px; border-bottom: 1px solid var(--line2); }
  h2 { margin: 0; font-size: 15px; font-weight: 600; color: var(--strong); display: flex; align-items: center; gap: 10px; }
  .into { font-size: 12px; font-weight: 400; color: var(--muted); background: var(--line); padding: 1px 8px; border-radius: 3px; }
  .x { width: 30px; height: 30px; border: 0; background: none; border-radius: 4px; color: var(--muted); font-size: 18px; }
  .x:hover { background: var(--line); color: var(--strong); }
  .bar { display: flex; gap: 10px; padding: 12px 22px; border-bottom: 1px solid var(--line); }
  .bar input { flex: 1; min-width: 0; }
  .body { min-height: 120px; max-height: 380px; overflow: auto; }
  .none { margin: 0; padding: 34px 22px; color: var(--muted); text-align: center; }
  .item { display: flex; align-items: center; gap: 12px; padding: 10px 22px; border-bottom: 1px solid var(--line); }
  .item:hover { background: var(--hover); }
  .txt { flex: 1; min-width: 0; display: grid; gap: 2px; }
  .txt b { color: var(--strong); font-weight: 500; }
  .cmd { font-size: 12px; color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .acts { display: flex; align-items: center; gap: 6px; flex: none; }
  .btn.sm { padding: 4px 12px; font-size: 12px; }
  .ico { width: 28px; height: 28px; display: grid; place-items: center; border: 0; background: none; border-radius: 4px; color: var(--muted); }
  .ico:hover { background: var(--line); color: var(--strong); }
  .ask { font-size: 12px; color: var(--err-text); }
  .note { margin: 0; padding: 8px 22px; font-size: 12px; }
  .note.bad { color: var(--err-text); }
  footer { padding: 10px 22px; border-top: 1px solid var(--line2); background: var(--foot); font-size: 11.5px; color: var(--muted); }
  .form { display: grid; gap: 14px; padding: 18px 22px 20px; }
  .form label { display: grid; gap: 6px; font-size: 12px; color: var(--muted); }
  .form textarea { font-family: var(--mono); font-size: 12.5px; resize: vertical; min-height: 90px; }
  .end { display: flex; justify-content: flex-end; gap: 10px; }
</style>
