<script lang="ts" module>
  // Open modals, outermost first. Only the top one answers Esc and Tab, and each sits above the last.
  const stack: number[] = [];
  let seq = 0;
</script>

<script lang="ts">
  import { onDestroy, onMount, type Snippet } from "svelte";
  import Icon from "./Icon.svelte";
  import { pop, veil } from "../lib/motion";

  /**
   * The one modal shell: scrim, panel, 52px header (title, optional chip, close), scrolling body,
   * optional footer. Handles Esc (top modal only), Tab trapping, first-field focus and giving
   * focus back to whatever opened it.
   */
  let { title, subtitle = "", size = "md", tall = false, flush = false, role = "dialog", dismissOnScrim = true, onclose, children, footer, extra }: {
    title: string;
    /** A small mono chip next to the title, e.g. the connection name. */
    subtitle?: string;
    size?: "sm" | "md" | "lg";
    /** A fixed, roomy height for panels that hold lists or tabs. */
    tall?: boolean;
    /** No padding on the body, for panels that lay themselves out. */
    flush?: boolean;
    role?: "dialog" | "alertdialog";
    /** Click on the scrim closes. Turn off for forms holding unsaved input. */
    dismissOnScrim?: boolean;
    onclose: () => void;
    children: Snippet;
    footer?: Snippet;
    extra?: Snippet;
  } = $props();

  const id = ++seq;
  const depth = stack.length;
  stack.push(id);
  const titleId = `modal-title-${id}`;
  let panel: HTMLElement;
  let pressedScrim = false;
  let opener: HTMLElement | null = null;

  const FOCUSABLE = 'button:not(:disabled), input:not(:disabled), select:not(:disabled), textarea:not(:disabled), [href], [tabindex]:not([tabindex="-1"])';
  const focusable = () => [...panel.querySelectorAll<HTMLElement>(FOCUSABLE)].filter((el) => el.offsetParent !== null || el === document.activeElement);

  onMount(() => {
    opener = document.activeElement as HTMLElement | null;
    const first = panel.querySelector<HTMLElement>("[data-autofocus], [autofocus]") ?? panel.querySelector<HTMLElement>(".modal-body input:not([type=checkbox]), .modal-body select, .modal-body textarea");
    (first ?? panel).focus({ preventScroll: true });
    window.addEventListener("keydown", onWindowKey);
  });
  onDestroy(() => {
    stack.splice(stack.indexOf(id), 1);
    if (typeof window === "undefined") return;
    window.removeEventListener("keydown", onWindowKey);
    if (opener?.isConnected) opener.focus?.({ preventScroll: true });
  });

  function onWindowKey(e: KeyboardEvent) {
    if (stack[stack.length - 1] !== id || e.defaultPrevented) return;
    if (e.key === "Escape") {
      e.preventDefault();
      onclose();
    }
  }

  function onPanelKey(e: KeyboardEvent) {
    if (e.key !== "Tab") return;
    const items = focusable();
    if (!items.length) return e.preventDefault();
    const [a, z] = [items[0], items[items.length - 1]];
    if (e.shiftKey && (document.activeElement === a || document.activeElement === panel)) { e.preventDefault(); z.focus(); }
    else if (!e.shiftKey && document.activeElement === z) { e.preventDefault(); a.focus(); }
  }
</script>

<div
  class="scrim"
  style:z-index="calc(var(--z-modal) + {depth * 2})"
  role="presentation"
  onpointerdown={(e) => (pressedScrim = e.target === e.currentTarget)}
  onclick={(e) => { if (dismissOnScrim && pressedScrim && e.target === e.currentTarget) onclose(); pressedScrim = false; }}
  in:veil|global
  out:veil|global
>
  <div class="panel {size}" class:tall {role} aria-modal="true" aria-labelledby={titleId} tabindex="-1" bind:this={panel} onkeydown={onPanelKey} in:pop|global out:pop|global>
    <header>
      <h2 id={titleId}>{title}</h2>
      {#if subtitle}<span class="chip mono">{subtitle}</span>{/if}
      <span class="grow"></span>
      {#if extra}{@render extra()}{/if}
      <button type="button" class="icon-btn" aria-label="Close" onclick={onclose}><Icon name="x" /></button>
    </header>
    <div class="modal-body" class:flush>{@render children()}</div>
    {#if footer}<footer>{@render footer()}</footer>{/if}
  </div>
</div>

<style>
  .scrim { position: fixed; inset: 0; display: grid; place-items: center; padding: var(--s6); background: var(--scrim); }
  .panel { --w: 680px; width: min(var(--w), 100%); max-height: 100%; display: flex; flex-direction: column; background: var(--float); border: 1px solid var(--floatline); border-radius: var(--r-md); box-shadow: var(--e-modal); outline: none; overflow: hidden; }
  .panel.sm { --w: 440px; }
  .panel.lg { --w: 920px; }
  .panel.tall { height: min(640px, 100%); }
  header { flex: none; height: 52px; display: flex; align-items: center; gap: var(--s3); padding: 0 var(--s3) 0 var(--s5); border-bottom: 1px solid var(--line); }
  h2 { margin: 0; font-size: var(--t-lg); font-weight: 600; color: var(--strong); letter-spacing: -0.005em; }
  .grow { flex: 1; }
  .modal-body { flex: 1; min-height: 0; overflow: auto; padding: var(--s5); }
  .modal-body.flush { padding: 0; display: flex; flex-direction: column; }
  footer { flex: none; display: flex; align-items: center; justify-content: flex-end; gap: var(--s2); padding: var(--s3) var(--s5); border-top: 1px solid var(--line); background: var(--foot); }
</style>
