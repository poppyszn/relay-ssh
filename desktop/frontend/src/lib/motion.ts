// Shared enter/exit motion for overlays. Exponential ease-out; skipped entirely when the
// user asks for reduced motion (and where the Web Animations API is missing, as in tests).

const expoOut = (t: number) => (t === 1 ? 1 : 1 - Math.pow(2, -10 * t));

function still(): boolean {
  if (import.meta.env?.MODE === "test") return true; // happy-dom rejects cancelled animations
  if (typeof window === "undefined" || typeof window.matchMedia !== "function") return true;
  if (typeof Element === "undefined" || typeof Element.prototype.animate !== "function") return true;
  return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}

/** A modal or popover appearing: fades in and settles a few pixels up. */
export function pop(_node: Element, { duration = 160, y = 6 }: { duration?: number; y?: number } = {}) {
  return { duration: still() ? 0 : duration, easing: expoOut, css: (t: number) => `opacity: ${t}; transform: translateY(${(1 - t) * y}px) scale(${0.985 + 0.015 * t})` };
}

/** The scrim behind a modal. */
export function veil(_node: Element, { duration = 160 }: { duration?: number } = {}) {
  return { duration: still() ? 0 : duration, easing: expoOut, css: (t: number) => `opacity: ${t}` };
}
