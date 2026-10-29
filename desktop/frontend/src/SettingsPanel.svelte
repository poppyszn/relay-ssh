<script lang="ts">
  import { DENSITY, FONTS, SYSTEM, THEMES, settings } from "./settings.svelte";

  let { onclose }: { onclose: () => void } = $props();
  const sizes = () => settings.fontSize;
  const bump = (d: number) => (settings.fontSize = Math.min(24, Math.max(10, settings.fontSize + d)));
</script>

<div class="scrim" role="presentation" onclick={onclose} onkeydown={(e) => e.key === "Escape" && onclose()}>
  <div class="panel" role="dialog" aria-label="Settings" tabindex="-1" onclick={(e) => e.stopPropagation()}>
    <header><h2>Settings</h2><button aria-label="Close" onclick={onclose}>&times;</button></header>

    <section>
      <div class="label">Theme</div>
      <div class="themes" role="radiogroup" aria-label="Theme">
        {#each Object.entries(THEMES) as [id, t]}
          <button role="radio" aria-checked={settings.theme === id} class:on={settings.theme === id} onclick={() => (settings.theme = id)}>
            <span class="swatch" style="background: {t.vars.term}; box-shadow: inset 0 -10px {t.vars.rail}"><i style="background: {t.vars.accent}"></i></span>
            <span>{t.name}</span>
          </button>
        {/each}
        <button role="radio" aria-checked={settings.theme === SYSTEM} class:on={settings.theme === SYSTEM} onclick={() => (settings.theme = SYSTEM)} title="Follow the macOS light or dark appearance">
          <span class="swatch split" style="--l: {THEMES.midnight.vars.term}; --r: {THEMES.daylight.vars.term}; --a: {THEMES.daylight.vars.accent}"><i></i></span>
          <span>System</span>
        </button>
      </div>
    </section>

    <section>
      <div class="label">Terminal font</div>
      <div class="row">
        <select bind:value={settings.font} aria-label="Font family">
          {#each Object.entries(FONTS) as [id, f]}<option value={id}>{f.name}</option>{/each}
        </select>
        <div class="step" aria-label="Font size">
          <button onclick={() => bump(-1)} aria-label="Smaller">−</button>
          <span class="mono">{sizes()}px</span>
          <button onclick={() => bump(1)} aria-label="Larger">+</button>
        </div>
      </div>
      <div class="seg" role="radiogroup" aria-label="Line spacing">
        {#each Object.entries(DENSITY) as [id, d]}
          <button role="radio" aria-checked={settings.density === id} class:on={settings.density === id} onclick={() => (settings.density = id)}>{d.name}</button>
        {/each}
      </div>
    </section>

    <section class="last">
      <div class="label">Behaviour</div>
      <label class="check"><input type="checkbox" bind:checked={settings.copyOnSelect} /><span>Copy on select<small>Selecting text copies it to the clipboard.</small></span></label>
      <label class="check"><input type="checkbox" bind:checked={settings.cursorBlink} /><span>Blinking cursor</span></label>
    </section>
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; background: var(--scrim); z-index: 25; display: flex; justify-content: flex-end; padding: 48px 12px 0 0; }
  .panel { width: 340px; align-self: flex-start; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 24px 70px var(--shadow); overflow: hidden; }
  header { display: flex; align-items: center; justify-content: space-between; height: 48px; padding: 0 12px 0 18px; border-bottom: 1px solid var(--line2); }
  h2 { margin: 0; font-size: 14px; font-weight: 600; color: var(--strong); }
  header button { width: 30px; height: 30px; border: 0; background: none; border-radius: 4px; color: var(--muted); font-size: 18px; }
  header button:hover { background: var(--line); color: var(--strong); }
  section { padding: 14px 18px 16px; border-bottom: 1px solid var(--line2); display: grid; gap: 10px; }
  section.last { border-bottom: 0; }
  .themes { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; }
  .swatch.split { background: linear-gradient(90deg, var(--l) 50%, var(--r) 50%); }
  .swatch.split i { background: var(--a); }
  .themes button { display: grid; gap: 6px; padding: 6px; background: var(--field); border: 1px solid var(--fieldline); border-radius: 4px; color: var(--text2); font-size: 12px; }
  .themes button.on { border-color: var(--accent); color: var(--strong); box-shadow: 0 0 0 1px var(--accent); }
  .swatch { display: block; height: 38px; border-radius: 3px; position: relative; overflow: hidden; }
  .swatch i { position: absolute; left: 6px; top: 8px; width: 22px; height: 3px; }
  .row { display: flex; gap: 8px; }
  .row select { flex: 1; font-family: var(--sans); }
  .step { display: flex; align-items: center; border: 1px solid var(--fieldline); border-radius: 4px; background: var(--field); }
  .step button { width: 30px; height: 34px; border: 0; background: none; font-size: 16px; }
  .step button:hover { background: var(--line); }
  .step span { min-width: 44px; text-align: center; font-size: 12px; }
  .seg { display: grid; grid-template-columns: repeat(3, 1fr); }
  .seg button { padding: 7px 0; background: var(--field); border: 1px solid var(--fieldline); margin-left: -1px; color: var(--text2); font-size: 12px; }
  .seg button:first-child { margin-left: 0; border-radius: 4px 0 0 4px; }
  .seg button:last-child { border-radius: 0 4px 4px 0; }
  .seg button.on { background: var(--accent-soft); border-color: var(--accent); color: var(--strong); position: relative; z-index: 1; }
  .check { display: flex; align-items: flex-start; gap: 10px; cursor: pointer; }
  .check input { width: 16px; height: 16px; margin: 2px 0 0; padding: 0; accent-color: var(--accent); }
  .check span { display: grid; gap: 1px; }
  .check small { color: var(--muted); font-size: 11.5px; }
</style>
