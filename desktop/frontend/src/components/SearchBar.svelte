<script lang="ts">
  import { tick } from "svelte";
  import type { Session } from "../lib/terminal";
  import { settings } from "../state/settings.svelte";

  let { session, onclose }: { session: Session; onclose: () => void } = $props();

  let query = $state("");
  let caseSensitive = $state(false);
  let count = $state({ index: -1, total: 0 });
  let input: HTMLInputElement;

  // Match highlights come from the theme: yellow, brighter for the current match.
  const decorations = () => {
    const c = settings.themeDef.search;
    return { matchBackground: c.match, matchBorder: c.matchBorder, matchOverviewRuler: c.active, activeMatchBackground: c.active, activeMatchBorder: c.activeBorder, activeMatchColorOverviewRuler: c.activeBorder };
  };
  const opts = () => ({ caseSensitive, decorations: decorations() });

  $effect(() => {
    const sub = session.search.onDidChangeResults((r) => (count = { index: r.resultIndex, total: r.resultCount }));
    tick().then(() => { input?.focus(); input?.select(); });
    return () => { sub.dispose(); session.search.clearDecorations(); };
  });
  $effect(() => {
    // incremental: re-run whenever the query or case option changes
    caseSensitive;
    if (query) session.search.findNext(query, { ...opts(), incremental: true });
    else { session.search.clearDecorations(); count = { index: -1, total: 0 }; }
  });

  const next = () => query && session.search.findNext(query, opts());
  const prev = () => query && session.search.findPrevious(query, opts());
  function key(e: KeyboardEvent) {
    if (e.key === "Escape") { e.stopPropagation(); onclose(); }
    else if (e.key === "Enter") { e.preventDefault(); e.shiftKey ? prev() : next(); }
  }
  const label = $derived(!query ? "" : count.total === 0 ? "No results" : count.index < 0 ? `${count.total}+` : `${count.index + 1} of ${count.total}`);
</script>

<div class="bar" role="search" onkeydown={key}>
  <input bind:this={input} bind:value={query} class="mono" placeholder="Find" aria-label="Find in scrollback" />
  <span class="count" class:none={query && count.total === 0}>{label}</span>
  <button aria-label="Previous match" title="Previous (⇧↵)" onclick={prev}>
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 15l6-6 6 6" /></svg>
  </button>
  <button aria-label="Next match" title="Next (↵)" onclick={next}>
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M6 9l6 6 6-6" /></svg>
  </button>
  <button class="aa" class:on={caseSensitive} aria-label="Match case" aria-pressed={caseSensitive} title="Match case" onclick={() => (caseSensitive = !caseSensitive)}>Aa</button>
  <button aria-label="Close search" title="Close (Esc)" onclick={onclose}>&times;</button>
</div>

<style>
  .bar { position: absolute; top: 12px; right: 18px; z-index: 5; display: flex; align-items: center; gap: 6px; padding: 6px 8px; background: var(--active); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 8px 24px var(--shadow); }
  input { height: 28px; min-width: 170px; font-size: 12.5px; border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .count { min-width: 56px; text-align: center; font-size: 11.5px; color: var(--muted); }
  .count.none { color: var(--err-text); }
  button { width: 26px; height: 26px; display: grid; place-items: center; border: 0; background: none; border-radius: 4px; color: var(--text); font-size: 16px; }
  button:hover { background: var(--line); }
  .aa { width: auto; padding: 0 6px; font-size: 12px; color: var(--muted); }
  .aa.on { color: var(--strong); background: var(--accent-soft); box-shadow: inset 0 0 0 1px var(--accent); }
</style>
