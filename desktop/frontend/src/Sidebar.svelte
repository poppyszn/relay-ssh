<script lang="ts">
  import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
  import { groupByFolder, matches } from "./lib";

  let { profiles, active, onconnect, onnew, onedit, ondelete, onpalette }: {
    profiles: Profile[];
    active: string;
    onconnect: (p: Profile) => void;
    onnew: () => void;
    onedit: (p: Profile) => void;
    ondelete: (p: Profile) => void;
    onpalette: () => void;
  } = $props();

  let query = $state("");
  let collapsed = $state<Record<string, boolean>>({});
  const groups = $derived(groupByFolder(profiles.filter((p) => matches(p, query))));
  const searching = $derived(query.trim() !== "");
</script>

<aside>
  <div class="top">
    <input bind:value={query} placeholder="Search  (#tag to filter by tag)" aria-label="Search connections" />
    <button class="btn primary" onclick={onnew} aria-label="New connection">+</button>
  </div>
  <button class="hint" onclick={onpalette}>Quick open <kbd>⌘K</kbd></button>

  <nav aria-label="Connections">
    {#each groups as g (g.path)}
      {#if g.path !== ""}
        <button class="folder" onclick={() => (collapsed[g.path] = !collapsed[g.path])} aria-expanded={searching || !collapsed[g.path]}>
          <span class="caret">{searching || !collapsed[g.path] ? "▾" : "▸"}</span>{g.path}
          <span class="count">{g.profiles.length}</span>
        </button>
      {/if}
      {#if searching || !collapsed[g.path]}
        {#each g.profiles as p (p.name)}
          <div class="row" class:active={p.name === active} class:nested={g.path !== ""}>
            <button class="main" ondblclick={() => onconnect(p)} onclick={() => onconnect(p)}>
              <span class="name">{p.name}</span>
              <span class="sub">{p.user}@{p.host}{p.port !== 22 ? `:${p.port}` : ""}</span>
              {#if p.tags?.length}<span class="tags">{#each p.tags as t}<span class="chip">{t}</span>{/each}</span>{/if}
            </button>
            <span class="actions">
              <button title="Edit" aria-label="Edit {p.name}" onclick={() => onedit(p)}>✎</button>
              <button title="Delete" aria-label="Delete {p.name}" onclick={() => ondelete(p)}>🗑</button>
            </span>
          </div>
        {/each}
      {/if}
    {:else}
      <p class="empty">{profiles.length === 0 ? "No connections yet. Press + to add one." : "No matches."}</p>
    {/each}
  </nav>
</aside>

<style>
  aside { width: 290px; flex: none; display: flex; flex-direction: column; background: var(--panel); border-right: 1px solid var(--border); min-height: 0; }
  .top { display: flex; gap: 6px; padding: 10px 10px 6px; }
  .top input { flex: 1; }
  .hint { margin: 0 10px 6px; text-align: left; background: none; border: 0; color: var(--muted); padding: 2px; }
  kbd { font: 11px var(--mono); border: 1px solid var(--border); border-radius: 4px; padding: 0 4px; }
  nav { overflow: auto; padding: 0 6px 10px; flex: 1; }
  .folder { display: flex; width: 100%; align-items: center; gap: 6px; background: none; border: 0; color: var(--muted); padding: 7px 6px 3px; text-align: left; font-size: 11px; text-transform: uppercase; letter-spacing: 0.06em; }
  .caret { width: 10px; }
  .count { margin-left: auto; font-size: 11px; }
  .row { display: flex; align-items: center; border-radius: var(--radius); position: relative; }
  .row.nested { margin-left: 12px; }
  .row:hover, .row.active { background: var(--raised); }
  .row.active { box-shadow: inset 2px 0 var(--accent); }
  .main { flex: 1; min-width: 0; background: none; border: 0; text-align: left; padding: 6px 8px; display: grid; gap: 1px; }
  .name { font-weight: 600; }
  .sub { color: var(--muted); font: 11px var(--mono); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .tags { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 2px; }
  .actions { display: none; gap: 2px; padding-right: 6px; }
  .row:hover .actions, .row:focus-within .actions { display: flex; }
  .actions button { background: none; border: 0; padding: 2px 5px; border-radius: 4px; color: var(--muted); }
  .actions button:hover { color: var(--text); background: var(--border); }
  .empty { color: var(--muted); padding: 16px 10px; }
</style>
