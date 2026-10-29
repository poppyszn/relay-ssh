<script lang="ts">
  let { text, sub = "", confirm, danger = false, onconfirm, oncancel, indent = 14 }: {
    text: string; sub?: string; confirm: string; danger?: boolean; onconfirm: () => void; oncancel: () => void; indent?: number;
  } = $props();
</script>

<div class="wrap" style="padding-left: {indent}px" role="alertdialog" tabindex="-1" aria-label={text} onkeydown={(e) => { if (e.key === "Escape") { e.stopPropagation(); oncancel(); } }}>
  <div class="card">
    <span class="arrow"></span>
    <div class="text">{text}</div>
    {#if sub}<div class="sub">{sub}</div>{/if}
    <div class="buttons">
      <button type="button" class="btn sm" onclick={oncancel}>Cancel</button>
      <button type="button" class="btn sm {danger ? 'danger' : 'primary'}" onclick={onconfirm}>{confirm}</button>
    </div>
  </div>
</div>

<style>
  .wrap { padding-right: var(--s3); padding-top: 6px; padding-bottom: var(--s2); }
  .card { position: relative; background: var(--active); border: 1px solid var(--floatline); border-radius: var(--r-md); box-shadow: var(--e-pop); padding: var(--s3); }
  .arrow { position: absolute; top: -6px; left: 22px; width: 10px; height: 10px; background: var(--active); border-left: 1px solid var(--floatline); border-top: 1px solid var(--floatline); transform: rotate(45deg); }
  .text { font-size: var(--t-md); color: var(--text); line-height: 1.45; }
  .sub { font-size: var(--t-sm); color: var(--muted); margin-top: var(--s1); }
  .buttons { display: flex; justify-content: flex-end; gap: var(--s2); margin-top: var(--s3); }
</style>
