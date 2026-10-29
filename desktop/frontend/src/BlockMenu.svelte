<script lang="ts">
  import { onMount } from "svelte";
  import type { BlockRef } from "./blocks";

  let { block, rect, oncopy, onclose }: {
    block: BlockRef;
    /** The menu button's position: the menu opens just below it. */
    rect: DOMRect;
    oncopy: (label: string, text: string) => void;
    onclose: () => void;
  } = $props();

  // Read once, when the menu opens, so what you copy is what was on screen then (and the
  // position is fixed at the button's place when it opened).
  // svelte-ignore state_referenced_locally
  const texts = { command: block.command(), output: block.output(), all: block.all() };
  const items = [
    { label: "Copy command", done: "command", text: texts.command, icon: "M9 9h10v10H9zM5 15V5h10" },
    { label: "Copy output", done: "output", text: texts.output, icon: "M4 6h16M4 12h16M4 18h10" },
    { label: "Copy command and output", done: "command and output", text: texts.all, icon: "M5 5h14v14H5zM8 9h8M8 13h8M8 17h5" },
  ];

  const W = 252;
  // svelte-ignore state_referenced_locally
  const left = Math.max(8, Math.min(window.innerWidth - W - 8, rect.right - W));
  // svelte-ignore state_referenced_locally
  const top = Math.min(rect.bottom + 6, window.innerHeight - 160);
  let sel = $state(0);
  let box: HTMLElement;

  const pick = (i: number) => {
    const it = items[i];
    if (!it.text) return;
    oncopy(it.done, it.text);
    onclose();
  };
  onMount(() => {
    const away = (e: PointerEvent) => !box.contains(e.target as Node) && onclose();
    window.addEventListener("pointerdown", away, true);
    box.focus();
    return () => window.removeEventListener("pointerdown", away, true);
  });
  function key(e: KeyboardEvent) {
    if (e.key === "Escape") onclose();
    else if (e.key === "ArrowDown") { sel = (sel + 1) % items.length; e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = (sel + items.length - 1) % items.length; e.preventDefault(); }
    else if (e.key === "Enter") { pick(sel); e.preventDefault(); }
  }
</script>

<div class="menu" role="menu" aria-label="Block actions" tabindex="-1" bind:this={box} onkeydown={key} style="left: {left}px; top: {top}px; width: {W}px">
  {#each items as it, i}
    <button class="item" class:hl={i === sel} role="menuitem" disabled={!it.text} onmouseenter={() => (sel = i)} onclick={() => pick(i)}>
      <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d={it.icon} /></svg>
      <span>{it.label}</span>
      {#if !it.text}<span class="none">empty</span>{/if}
    </button>
  {/each}
</div>

<style>
  .menu { position: fixed; z-index: 40; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 14px 40px var(--shadow); padding: 4px 0; outline: none; }
  .item { all: unset; box-sizing: border-box; width: 100%; display: flex; align-items: center; gap: 10px; padding: 9px 14px; cursor: pointer; color: var(--text); }
  .item svg { color: var(--muted); flex: none; }
  .item.hl:not(:disabled) { background: var(--accent-hl); box-shadow: inset 3px 0 var(--accent); }
  .item.hl:not(:disabled) svg { color: var(--accent-text); }
  .item:disabled { opacity: 0.45; cursor: default; }
  .none { margin-left: auto; font-size: 11px; color: var(--dim); }
</style>
