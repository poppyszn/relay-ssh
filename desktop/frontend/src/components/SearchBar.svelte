<script lang="ts">
  import { tick } from "svelte";
  import type { Session } from "../lib/terminal";
  import { settings } from "../state/settings.svelte";
  import Icon from "./Icon.svelte";

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

<!-- Esc and Enter come from whichever control inside the bar has focus. -->
<!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
<div class="bar" role="search" onkeydown={key}>
  <input bind:this={input} bind:value={query} class="mono" placeholder="Find" aria-label="Find in scrollback" />
  <span class="count" class:none={query && count.total === 0}>{label}</span>
  <button type="button" class="icon-btn dense" aria-label="Previous match" title="Previous (⇧↵)" onclick={prev}><Icon name="chevron-up" size={14} /></button>
  <button type="button" class="icon-btn dense" aria-label="Next match" title="Next (↵)" onclick={next}><Icon name="chevron-down" size={14} /></button>
  <button type="button" class="icon-btn dense aa" class:on={caseSensitive} aria-label="Match case" aria-pressed={caseSensitive} title="Match case" onclick={() => (caseSensitive = !caseSensitive)}>Aa</button>
  <button type="button" class="icon-btn dense" aria-label="Close search" title="Close (Esc)" onclick={onclose}><Icon name="x" size={14} /></button>
</div>

<style>
  .bar { position: absolute; top: var(--s3); right: 18px; z-index: 5; display: flex; align-items: center; gap: var(--s1); padding: var(--s1) var(--s2); background: var(--active); border: 1px solid var(--floatline); border-radius: var(--r-md); box-shadow: var(--e-pop); }
  input { height: var(--h-sm); min-width: 170px; font-size: var(--t-sm); }
  .count { min-width: 56px; text-align: center; font-size: var(--t-sm); color: var(--muted); }
  .count.none { color: var(--err-text); }
  .aa { width: auto; padding: 0 6px; font-size: var(--t-sm); font-weight: 600; }
</style>
