<script lang="ts">
  import type { BlockRef } from "../lib/blocks";
  import Menu, { type MenuItem } from "./Menu.svelte";

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
  const entries = [
    { id: "command", label: "Copy command", done: "command", text: texts.command, icon: "copy" },
    { id: "output", label: "Copy output", done: "output", text: texts.output, icon: "lines" },
    { id: "all", label: "Copy command and output", done: "command and output", text: texts.all, icon: "copy-all" },
  ] as const;
  const items: MenuItem[] = entries.map((e) => ({ id: e.id, label: e.label, icon: e.icon, disabled: !e.text, hint: e.text ? "" : "empty" }));

  const W = 252;
  // svelte-ignore state_referenced_locally
  const left = Math.max(8, Math.min(window.innerWidth - W - 8, rect.right - W));
  // svelte-ignore state_referenced_locally
  const top = Math.min(rect.bottom + 6, window.innerHeight - 160);

  function pick(id: string) {
    const e = entries.find((x) => x.id === id);
    if (!e?.text) return;
    oncopy(e.done, e.text);
    onclose();
  }
</script>

<Menu {items} label="Block actions" width={W} style="position: fixed; left: {left}px; top: {top}px" onselect={pick} {onclose} />
