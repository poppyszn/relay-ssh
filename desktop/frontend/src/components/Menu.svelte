<script lang="ts" module>
  import type { IconName } from "./Icon.svelte";
  export interface MenuItem {
    id: string;
    label: string;
    /** Second line; makes the row taller. */
    sub?: string;
    icon?: IconName;
    kbd?: string;
    danger?: boolean;
    disabled?: boolean;
    /** A hairline above this row. */
    gapBefore?: boolean;
  }
</script>

<script lang="ts">
  import { onMount } from "svelte";
  import Icon from "./Icon.svelte";
  import { pop } from "../lib/motion";

  /**
   * The one menu: arrows, Enter, Esc, click-away, focus on open. The caller positions it
   * (pass `style`, e.g. "top: 46px; left: 12px"), and `ignore` names the trigger so clicking it
   * to close does not immediately reopen the menu.
   */
  let { items, label, onselect, onclose, ignore = "", style = "", width = 232 }: {
    items: MenuItem[];
    label: string;
    onselect: (id: string) => void;
    onclose: () => void;
    ignore?: string;
    style?: string;
    width?: number;
  } = $props();

  let sel = $state(0);
  let box: HTMLElement;
  const rich = $derived(items.some((i) => i.sub));

  function step(dir: 1 | -1) {
    for (let n = 1; n <= items.length; n++) {
      const i = (sel + dir * n + items.length * n) % items.length;
      if (!items[i].disabled) { sel = i; return; }
    }
  }
  function run(i: number) {
    if (!items[i] || items[i].disabled) return;
    onselect(items[i].id);
  }

  onMount(() => {
    sel = Math.max(0, items.findIndex((i) => !i.disabled));
    const away = (e: PointerEvent) => {
      const t = e.target as HTMLElement;
      if (!box.contains(t) && !(ignore && t.closest(ignore))) onclose();
    };
    window.addEventListener("pointerdown", away, true);
    box.focus({ preventScroll: true });
    return () => window.removeEventListener("pointerdown", away, true);
  });

  function key(e: KeyboardEvent) {
    if (e.key === "Escape") { e.preventDefault(); e.stopPropagation(); onclose(); }
    else if (e.key === "ArrowDown") { e.preventDefault(); step(1); }
    else if (e.key === "ArrowUp") { e.preventDefault(); step(-1); }
    else if (e.key === "Home") { e.preventDefault(); sel = items.findIndex((i) => !i.disabled); }
    else if (e.key === "Enter" || e.key === " ") { e.preventDefault(); run(sel); }
    else if (e.key === "Tab") onclose();
  }
</script>

<div class="menu" role="menu" aria-label={label} tabindex="-1" bind:this={box} onkeydown={key} style="width: {width}px; {style}" in:pop|global={{ duration: 120, y: 4 }}>
  {#each items as it, i (it.id)}
    {#if it.gapBefore}<div class="sep" role="separator"></div>{/if}
    <button type="button" class="item" class:rich class:row-hl={i === sel} class:danger={it.danger} role="menuitem" disabled={it.disabled} tabindex="-1" onmouseenter={() => !it.disabled && (sel = i)} onclick={() => run(i)}>
      {#if it.icon}<span class="ico"><Icon name={it.icon} /></span>{/if}
      <span class="txt"><span class="t">{it.label}</span>{#if it.sub}<span class="s">{it.sub}</span>{/if}</span>
      {#if it.kbd}<span class="kbd">{it.kbd}</span>{/if}
    </button>
  {/each}
</div>

<style>
  .menu { position: absolute; z-index: var(--z-pop); padding: var(--s1); background: var(--float); border: 1px solid var(--floatline); border-radius: var(--r-md); box-shadow: var(--e-pop); outline: none; }
  .sep { height: 1px; margin: var(--s1) var(--s2); background: var(--line); }
  .item { display: flex; align-items: center; gap: var(--s3); width: 100%; min-height: var(--h-md); padding: 0 var(--s3); background: none; border: 0; border-radius: var(--r-sm); color: var(--text); text-align: left; font-size: var(--t-md); }
  .item.rich { min-height: 48px; }
  .item:disabled { opacity: 0.45; cursor: default; }
  .item.danger { color: var(--err-text); }
  .ico { flex: none; display: grid; place-items: center; width: 28px; height: 28px; border-radius: var(--r-sm); background: var(--line); color: var(--text2); }
  .item:not(.rich) .ico { width: 16px; height: 16px; background: none; }
  .row-hl .ico { color: var(--accent-text); background: var(--accent-ring); }
  .item:not(.rich).row-hl .ico { background: none; }
  .txt { display: grid; gap: 1px; flex: 1; min-width: 0; }
  .t { font-weight: 500; }
  .s { font-size: var(--t-sm); color: var(--muted); }
  .row-hl .s { color: var(--text2); }
</style>
