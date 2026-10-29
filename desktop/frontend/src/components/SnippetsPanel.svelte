<script lang="ts">
  import { SnippetsService, type Snippet } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { snippetPreview } from "../lib/lib";
  import EmptyState from "./EmptyState.svelte";
  import Icon from "./Icon.svelte";
  import Modal from "./Modal.svelte";

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
  const formId = "snippet-form";

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
  // Esc inside the form steps back to the list instead of closing the panel.
  function formKey(e: KeyboardEvent) {
    if (e.key !== "Escape") return;
    e.preventDefault();
    cancelEdit();
  }
  function cancelEdit() {
    editing = null;
    error = "";
  }
  const title = $derived(editing === null ? "Snippets" : editing ? "Edit snippet" : "New snippet");
</script>

<Modal {title} subtitle={into} size="md" tall flush={editing === null} {onclose}>
  {#if editing === null}
    <div class="bar">
      <input class="grow" use:focus bind:value={query} data-autofocus placeholder="Search snippets" aria-label="Search snippets" />
      {#if !loaded || list.length}<button type="button" class="btn primary" onclick={() => edit()}>New snippet</button>{/if}
    </div>
    {#if error}<div class="strip"><div class="notice bad" role="alert">{error}</div></div>{/if}
    <div class="list">
      {#each shown as s (s.name)}
        <div class="item row-hover">
          <div class="txt">
            <span class="name">{s.name}</span>
            <span class="mono cmd">{snippetPreview(s.command)}</span>
          </div>
          {#if confirmDelete === s.name}
            <div class="acts">
              <button type="button" class="btn sm" use:focus onclick={() => (confirmDelete = "")}>Cancel</button>
              <button type="button" class="btn sm danger" onclick={() => remove(s.name)}>Delete</button>
            </div>
          {:else}
            <div class="acts">
              <button type="button" class="btn sm" title="Type it without pressing Enter, so you can edit it first" onclick={() => use(s, false)}>Insert</button>
              <button type="button" class="btn sm primary" title="Type it and press Enter" onclick={() => use(s, true)}>Run</button>
              <button type="button" class="icon-btn" aria-label="Edit {s.name}" title="Edit" onclick={() => edit(s)}><Icon name="pencil" /></button>
              <button type="button" class="icon-btn" aria-label="Delete {s.name}" title="Delete" onclick={() => (confirmDelete = s.name)}><Icon name="trash" /></button>
            </div>
          {/if}
        </div>
      {:else}
        {#if loaded && list.length}
          <EmptyState icon="search" title="No snippets match" text="Try another word from the name or the command." />
        {:else if loaded}
          <EmptyState icon="snippets" title="No snippets yet" text="Save a command you type often, then run it in any terminal.">
            <button type="button" class="btn primary" onclick={() => edit()}>New snippet</button>
          </EmptyState>
        {/if}
      {/each}
    </div>
  {:else}
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <form id={formId} class="form" onsubmit={(e) => { e.preventDefault(); save(); }} onkeydown={formKey}>
      <label class="field"><span>Name</span><input use:focus data-autofocus bind:value={name} placeholder="Tail the nginx log" /></label>
      <label class="field"><span>Command</span><textarea class="mono" bind:value={command} rows="6" spellcheck="false" placeholder="tail -f /var/log/nginx/error.log"></textarea></label>
      {#if error}<div class="notice bad" role="alert">{error}</div>{/if}
    </form>
  {/if}

  {#snippet footer()}
    {#if editing === null}
      <span class="hint note">Saved in <span class="mono">~/.config/sssh/snippets.toml</span>. Insert types it for you to edit; Run also presses Enter.</span>
    {:else}
      <button type="button" class="btn ghost" onclick={cancelEdit}>Cancel</button>
      <button type="submit" form={formId} class="btn primary">{editing ? "Save changes" : "Add snippet"}</button>
    {/if}
  {/snippet}
</Modal>

<style>
  .bar { display: flex; gap: var(--s2); padding: var(--s3) var(--s5); border-bottom: 1px solid var(--line); }
  .grow { flex: 1; min-width: 0; }
  .strip { padding: var(--s3) var(--s5) 0; }
  .list { flex: 1; min-height: 0; overflow: auto; }
  .item { display: flex; align-items: center; gap: var(--s3); min-height: 52px; padding: var(--s2) var(--s5); border-bottom: 1px solid var(--line); }
  .txt { flex: 1; min-width: 0; display: grid; gap: 2px; }
  .name { color: var(--strong); font-weight: 500; }
  .cmd { font-size: var(--t-sm); color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .acts { display: flex; align-items: center; gap: var(--s2); flex: none; }
  .note { margin-right: auto; }
  .form { display: grid; gap: var(--s4); }
  textarea { resize: vertical; min-height: 96px; }
</style>
