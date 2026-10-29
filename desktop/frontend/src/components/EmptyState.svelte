<script lang="ts">
  import type { Snippet } from "svelte";
  import Icon, { type IconName } from "./Icon.svelte";

  // The one empty state: say what is missing, and offer the way to fill it.
  let { icon, title, text = "", children }: { icon?: IconName; title: string; text?: string; children?: Snippet } = $props();
</script>

<div class="empty">
  {#if icon}<span class="glyph"><Icon name={icon} size={20} /></span>{/if}
  <strong>{title}</strong>
  {#if text}<p>{text}</p>{/if}
  {#if children}<div class="act">{@render children()}</div>{/if}
</div>

<style>
  .empty { display: grid; justify-items: center; gap: var(--s2); padding: var(--s8) var(--s6); text-align: center; color: var(--muted); }
  .glyph { display: grid; place-items: center; width: 40px; height: 40px; border-radius: var(--r-md); background: var(--line); color: var(--text2); margin-bottom: var(--s1); }
  strong { color: var(--text); font-size: var(--t-md); font-weight: 600; }
  p { margin: 0; max-width: 34ch; font-size: var(--t-sm); line-height: 1.5; }
  .act { margin-top: var(--s2); display: flex; gap: var(--s2); }
</style>
