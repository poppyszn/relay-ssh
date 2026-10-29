<script lang="ts">
  import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
  import { groupByFolder, matches } from "./lib";

  let { profiles, active, onconnect, onnew, onedit, onpalette }: {
    profiles: Profile[];
    active: Set<string>;
    onconnect: (p: Profile) => void;
    onnew: () => void;
    onedit: (p: Profile) => void;
    onpalette: () => void;
  } = $props();

  let query = $state("");
  let collapsed = $state<Record<string, boolean>>({});
  const groups = $derived(groupByFolder(profiles.filter((p) => matches(p, query))));
  const searching = $derived(query.trim() !== "");
</script>

<aside>
  <div class="top">
    <div class="search">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true"><circle cx="11" cy="11" r="7" /><path d="M20 20l-3.5-3.5" /></svg>
      <input bind:value={query} placeholder="Search" aria-label="Search connections" />
      <button class="kbd" onclick={onpalette} title="Quick open">⌘K</button>
    </div>
    <button class="add" onclick={onnew} aria-label="New connection" title="New connection">+</button>
  </div>

  <nav aria-label="Connections">
    {#each groups as g (g.path)}
      <button class="bar" onclick={() => (collapsed[g.path] = !collapsed[g.path])} aria-expanded={searching || !collapsed[g.path]}>
        <span class="label">{g.path === "" ? "Unfiled" : g.path}</span>
        <span class="label count">{g.profiles.length}</span>
      </button>
      {#if searching || !collapsed[g.path]}
        {#each g.profiles as p (p.name)}
          <div class="row" class:active={active.has(p.name)}>
            <button class="main" onclick={() => onconnect(p)}>
              <span class="line1"><span class="dot" class:ok={active.has(p.name)}></span><span class="name">{p.name}</span>{#each p.tags ?? [] as t}<span class="chip">{t}</span>{/each}</span>
              <span class="sub mono">{p.user}@{p.host}{p.port !== 22 ? `:${p.port}` : ""}</span>
            </button>
            <button class="edit" aria-label="Edit {p.name}" title="Edit" onclick={() => onedit(p)}>
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z" /></svg>
            </button>
          </div>
        {/each}
      {/if}
    {:else}
      <p class="empty">{profiles.length === 0 ? "No connections yet. Press + to add one." : "No matches."}</p>
    {/each}
  </nav>
</aside>

<style>
  aside { width: 248px; flex: none; display: flex; flex-direction: column; background: var(--rail); border-right: 1px solid var(--line); min-height: 0; }
  .top { display: flex; gap: 8px; padding: 12px; }
  .search { flex: 1; min-width: 0; display: flex; align-items: center; gap: 8px; height: 32px; padding: 0 8px 0 10px; background: var(--field); border: 1px solid var(--line); border-radius: 4px; color: var(--muted); }
  .search:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px rgba(59, 154, 225, 0.18); }
  .search input { flex: 1; height: 100%; border: 0; background: none; padding: 0; box-shadow: none; }
  .search input:focus { box-shadow: none; }
  .search .kbd { background: none; cursor: pointer; }
  .add { width: 32px; height: 32px; border: 0; border-radius: 4px; background: var(--accent); color: #06131f; font-size: 18px; font-weight: 600; }
  .add:hover { background: #4aa8ee; }
  nav { flex: 1; overflow: auto; }
  .bar { display: flex; justify-content: space-between; width: 100%; padding: 7px 16px; background: var(--bar); border: 0; border-top: 1px solid #1f2b3a; border-bottom: 1px solid #1f2b3a; margin-top: -1px; }
  .row { display: flex; align-items: stretch; position: relative; }
  .row:hover { background: var(--hover); }
  .row.active { background: var(--active); box-shadow: inset 3px 0 var(--accent); }
  .main { all: unset; flex: 1; min-width: 0; padding: 9px 16px; display: grid; gap: 1px; cursor: pointer; }
  .line1 { display: flex; align-items: center; gap: 8px; min-width: 0; }
  .name { font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .sub { padding-left: 16px; font-size: 11px; color: var(--muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .edit { all: unset; width: 32px; display: none; place-items: center; color: var(--muted); cursor: pointer; }
  .row:hover .edit, .edit:focus-visible { display: grid; }
  .edit:hover { color: var(--text); }
  .empty { color: var(--muted); padding: 16px; margin: 0; }
</style>
