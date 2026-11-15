// Shared enter/exit motion for overlays, drawers, and views. Smooth cubic-out deceleration;
// skipped when reduced motion is requested, the document is hidden, or in tests without WAAPI.

const easeOut = (t: number) => 1 - Math.pow(1 - t, 3);

function still(): boolean {
  if (import.meta.env?.MODE === "test") return true; // happy-dom rejects cancelled animations
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") return true;
  if (typeof Element === "undefined" || typeof Element.prototype.animate !== "function") return true;
  if (typeof document !== "undefined" && document.hidden) return true;
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/** Smooth scale + lift pop for modals, the palette, search bar, and menus. */
export function pop(
  _node: Element,
  { duration = 220, y = 10, scale = 0.96 }: { duration?: number; y?: number; scale?: number } = {},
) {
  return {
    duration: still() ? 0 : duration,
    easing: easeOut,
    css: (t: number) =>
      `opacity: ${t}; transform: translateY(${(1 - t) * y}px) scale(${scale + (1 - scale) * t})`,
  };
}

/** Scrim veil behind a modal or the palette. */
export function veil(_node: Element, { duration = 200 }: { duration?: number } = {}) {
  return {
    duration: still() ? 0 : duration,
    easing: easeOut,
    css: (t: number) => `opacity: ${t}`,
  };
}

/** Horizontal slide-out transition for the Command Deck context drawer. */
export function drawerSlide(
  _node: Element,
  { duration = 240, width = 348 }: { duration?: number; width?: number } = {},
) {
  return {
    duration: still() ? 0 : duration,
    easing: easeOut,
    css: (t: number) =>
      `width: ${(t * width).toFixed(2)}px; min-width: 0; opacity: ${Math.min(1, t * 1.4).toFixed(3)}; overflow: hidden`,
  };
}

/** Subtle vertical entrance when switching drawer panels or inner views. */
export function panelSwap(_node: Element, { duration = 180, y = 8 }: { duration?: number; y?: number } = {}) {
  return {
    duration: still() ? 0 : duration,
    easing: easeOut,
    css: (t: number) => `opacity: ${t}; transform: translateY(${(1 - t) * y}px)`,
  };
}

/** Layer transition for Spotlight Home entering/leaving over the workspace stage. */
export function deckLayer(_node: Element, { duration = 220 }: { duration?: number } = {}) {
  return {
    duration: still() ? 0 : duration,
    easing: easeOut,
    css: (t: number) =>
      `opacity: ${t}; transform: translateY(${(1 - t) * 8}px) scale(${0.985 + 0.015 * t})`,
  };
}
