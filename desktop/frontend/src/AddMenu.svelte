<script lang="ts">
  import { onMount } from "svelte";

  let { folder, onconnection, onfolder, onsubfolder, onimport, onkey, onclose }: {
    /** The selected folder, "" for none: shown in the subfolder item. */
    folder: string;
    onconnection: () => void;
    onfolder: () => void;
    onsubfolder: () => void;
    onimport: () => void;
    onkey: () => void;
    onclose: () => void;
  } = $props();

  let sel = $state(0);
  const items = $derived([
    { icon: "term", title: "Connection", sub: "A server you SSH into", kbd: "⌘N", run: onconnection },
    { icon: "folder", title: "Folder", sub: "Group connections together", kbd: "⇧⌘N", run: onfolder },
    ...(folder ? [{ icon: "sub", title: `Subfolder in “${folder.split("/").pop()}”`, sub: "Nest a folder inside the selected one", kbd: "", run: onsubfolder }] : []),
    { icon: "import", title: "Import from ~/.ssh/config", sub: "Bring in hosts you already have", kbd: "", run: onimport, gap: true },
    { icon: "key", title: "Key", sub: "Adopt an existing key or generate a new one", kbd: "", run: onkey },
  ]);
  const icons: Record<string, string> = {
    term: "M4 17l6-6-6-6M12 19h8",
    folder: "M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z",
    sub: "M3 8a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v1H7l-3 8H5a2 2 0 0 1-2-2zM7 11h15l-3 8H4z",
    import: "M12 3v12M7 10l5 5 5-5M4 20h16",
    key: "M11 12l9-9M16 7l3 3M8 11a4 4 0 1 0 0 8 4 4 0 0 0 0-8z",
  };

  let box: HTMLElement;
  onMount(() => {
    const away = (e: PointerEvent) => !box.contains(e.target as Node) && !(e.target as HTMLElement).closest("[data-add-button]") && onclose();
    window.addEventListener("pointerdown", away, true);
    box.focus();
    return () => window.removeEventListener("pointerdown", away, true);
  });
  function key(e: KeyboardEvent) {
    if (e.key === "Escape") onclose();
    else if (e.key === "ArrowDown") { sel = (sel + 1) % items.length; e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = (sel + items.length - 1) % items.length; e.preventDefault(); }
    else if (e.key === "Enter") { items[sel].run(); e.preventDefault(); }
  }
</script>

<div class="menu" role="menu" aria-label="Add" tabindex="-1" bind:this={box} onkeydown={key}>
  <div class="label head">Add</div>
  {#each items as it, i}
    {#if it.gap}<div class="sep"></div>{/if}
    <button class="item" class:hl={i === sel} role="menuitem" onmouseenter={() => (sel = i)} onclick={it.run}>
      <span class="ico"><svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d={icons[it.icon]} /></svg></span>
      <span class="txt"><span class="t">{it.title}</span><span class="s">{it.sub}</span></span>
      {#if it.kbd}<span class="kbd">{it.kbd}</span>{/if}
    </button>
  {/each}
</div>

<style>
  .menu { position: absolute; left: 12px; right: 12px; top: 46px; z-index: 30; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 18px 50px var(--shadow); overflow: hidden; outline: none; }
  .head { padding: 10px 14px 6px; }
  .sep { height: 1px; background: var(--line2); margin: 6px 0; }
  .item { all: unset; box-sizing: border-box; width: 100%; display: flex; align-items: center; gap: 12px; padding: 9px 14px; cursor: pointer; }
  .item.hl { background: var(--accent-hl); box-shadow: inset 3px 0 var(--accent); }
  .ico { width: 30px; height: 30px; display: grid; place-items: center; background: var(--line); border-radius: 6px; color: var(--text2); flex: none; }
  .hl .ico { background: var(--accent-ring); color: var(--accent-text); }
  .txt { display: grid; gap: 1px; flex: 1; min-width: 0; }
  .t { color: var(--strong); font-weight: 500; }
  .s { font-size: 11.5px; color: var(--muted); }
</style>
