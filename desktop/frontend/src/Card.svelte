<script lang="ts">
  let { text, sub = "", confirm, danger = false, onconfirm, oncancel, indent = 14 }: {
    text: string; sub?: string; confirm: string; danger?: boolean; onconfirm: () => void; oncancel: () => void; indent?: number;
  } = $props();
</script>

<div class="wrap" style="padding-left: {indent}px" role="alertdialog" tabindex="-1" aria-label={text} onkeydown={(e) => e.key === "Escape" && oncancel()}>
  <div class="card">
    <span class="arrow"></span>
    <div class="text">{text}</div>
    {#if sub}<div class="sub">{sub}</div>{/if}
    <div class="buttons">
      <button class="btn" onclick={oncancel}>Cancel</button>
      <button class="btn primary" class:danger onclick={onconfirm}>{confirm}</button>
    </div>
  </div>
</div>

<style>
  .wrap { padding-right: 12px; padding-top: 6px; padding-bottom: 8px; }
  .card { position: relative; background: var(--active); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 14px 36px rgba(0, 0, 0, 0.5); padding: 12px 12px 10px; }
  .arrow { position: absolute; top: -6px; left: 22px; width: 10px; height: 10px; background: var(--active); border-left: 1px solid var(--floatline); border-top: 1px solid var(--floatline); transform: rotate(45deg); }
  .text { font-size: 12.5px; color: var(--text); line-height: 1.45; }
  .sub { font-size: 11.5px; color: var(--muted); margin: 3px 0 10px; }
  .text:not(:has(+ .sub)) { margin-bottom: 10px; }
  .buttons { display: flex; justify-content: flex-end; gap: 8px; }
  .btn { padding: 5px 12px; }
  .danger { background: var(--err); border-color: var(--err); color: #1a0508; }
</style>
