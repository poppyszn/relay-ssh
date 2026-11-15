<script lang="ts">
  import { SnippetsService, type Snippet } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { snippetPreview } from "../lib/lib";
  import EmptyState from "./EmptyState.svelte";
  import Icon from "./Icon.svelte";
  import Modal from "./Modal.svelte";

  let { into, embedded = false, onuse, onclose }: {
    /** The pane snippets go into, for the header: "" when there is none. */
    into: string;
    embedded?: boolean;
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
    if (onuse(s.command, run)) {
      if (!embedded) onclose();
    } else {
      error = "Open a terminal first: there is nowhere to type this.";
    }
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

{#snippet body()}
  {#if editing === null}
    <div class="bar" class:embedded-pad={embedded}>
      <input class="grow" use:focus bind:value={query} data-autofocus placeholder="Search snippets" aria-label="Search snippets" />
      {#if !embedded && (!loaded || list.length)}<button type="button" class="btn primary" onclick={() => edit()}>New snippet</button>{/if}
    </div>
    {#if error}<div class="strip" class:embedded-pad={embedded}><div class="notice bad" role="alert">{error}</div></div>{/if}
    <div class="list" class:embedded-list={embedded}>
      {#each shown as s (s.name)}
        <div class="item" class:drawer-card={embedded} class:row-hover={!embedded}>
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
    <form id={formId} class="form" class:embedded-form={embedded} onsubmit={(e) => { e.preventDefault(); save(); }} onkeydown={formKey}>
      <label class="field"><span>Name</span><input use:focus data-autofocus bind:value={name} placeholder="Tail the nginx log" /></label>
      <label class="field"><span>Command</span><textarea class="mono" bind:value={command} rows="6" spellcheck="false" placeholder="tail -f /var/log/nginx/error.log"></textarea></label>
      {#if error}<div class="notice bad" role="alert">{error}</div>{/if}
    </form>
  {/if}
{/snippet}

{#snippet foot()}
  {#if editing === null}
    <span class="hint note">{embedded ? (into ? `Typing into ${into}` : "Open a shell pane to insert or run") : ""}{#if !embedded}Saved in <span class="mono">~/.config/relay-ssh/snippets.toml</span>. Insert types it for you to edit; Run also presses Enter.{/if}</span>
  {:else}
    <button type="button" class="btn ghost" onclick={cancelEdit}>Cancel</button>
    <button type="submit" form={formId} class="btn primary">{editing ? "Save changes" : "Add snippet"}</button>
  {/if}
{/snippet}

{#if embedded}
  <div class="drawer-panel">
    <div class="drawer-head">
      <div class="drawer-title">
        <span class="title-strong">{title}</span>
        {#if into}<span class="drawer-sub">Into {into}</span>{/if}
      </div>
      <div class="head-acts">
        {#if editing === null}
          <button type="button" class="btn sm ghost" onclick={() => edit()}><Icon name="plus" size={14} /><span>Snippet</span></button>
        {/if}
        <button type="button" class="icon-btn dense" aria-label="Close drawer" onclick={onclose}><Icon name="x" size={14} /></button>
      </div>
    </div>
    {@render body()}
    <div class="drawer-foot">
      {@render foot()}
    </div>
  </div>
{:else}
  <Modal {title} subtitle={into} size="md" tall flush={editing === null} {onclose}>
    {@render body()}
    {#snippet footer()}
      {@render foot()}
    {/snippet}
  </Modal>
{/if}

<style>
  .drawer-panel { flex: 1; min-height: 0; display: flex; flex-direction: column; }
  .drawer-head { display: flex; align-items: center; justify-content: space-between; gap: var(--s2); padding: 12px 14px 10px; border-bottom: 1px solid var(--hairline); }
  .drawer-title { display: grid; gap: 2px; min-width: 0; }
  .title-strong { font-size: var(--t-sm); font-weight: 600; color: var(--strong); }
  .drawer-sub { font-size: 11px; color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .head-acts { display: flex; align-items: center; gap: 4px; }
  .drawer-foot { display: flex; align-items: center; justify-content: flex-end; gap: var(--s2); padding: 10px 14px; border-top: 1px solid var(--hairline); }
  .bar { display: flex; gap: var(--s2); padding: var(--s3) var(--s5); border-bottom: 1px solid var(--hairline); }
  .bar.embedded-pad, .strip.embedded-pad { padding-left: 14px; padding-right: 14px; }
  .grow { flex: 1; min-width: 0; }
  .strip { padding: var(--s3) var(--s5) 0; }
  .list { flex: 1; min-height: 0; overflow: auto; }
  .list.embedded-list { padding: 10px 14px; display: grid; align-content: start; gap: 8px; }
  .item { display: flex; align-items: center; gap: var(--s3); min-height: 52px; padding: var(--s2) var(--s5); border-bottom: 1px solid var(--hairline); }
  .item.drawer-card { flex-direction: column; align-items: stretch; gap: 8px; padding: 10px 12px; border: 1px solid var(--hairline); border-radius: 12px; background: var(--hover); }
  .item.drawer-card:hover { background: var(--active); }
  .txt { flex: 1; min-width: 0; display: grid; gap: 2px; }
  .name { color: var(--strong); font-weight: 500; font-size: var(--t-md); }
  .cmd { font-size: var(--t-sm); color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .acts { display: flex; align-items: center; gap: var(--s2); flex: none; justify-content: flex-end; }
  .note { margin-right: auto; }
  .form { display: grid; gap: var(--s4); }
  .form.embedded-form { flex: 1; min-height: 0; overflow: auto; align-content: start; padding: 14px; }
  textarea { resize: vertical; min-height: 96px; }
</style>
