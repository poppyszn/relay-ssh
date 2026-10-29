import type { ITheme } from "@xterm/xterm";

// Palettes as plain data (no framework), so they can be unit tested for contrast.

export interface ThemeDef {
  name: string;
  scheme: "dark" | "light";
  /** CSS custom properties applied to :root (without the leading --). */
  vars: Record<string, string>;
  term: ITheme;
  /** Highlight colours for scrollback search matches (xterm wants #rrggbb). */
  search: { match: string; matchBorder: string; active: string; activeBorder: string };
}

export const hexToRgb = (hex: string): [number, number, number] => [1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)) as [number, number, number];
export const rgba = (hex: string, a: number) => `rgba(${hexToRgb(hex).join(", ")}, ${a})`;

/** WCAG contrast ratio between two #rrggbb colours. */
export function contrast(a: string, b: string): number {
  const lum = (hex: string) => {
    const [r, g, bl] = hexToRgb(hex).map((v) => {
      const c = v / 255;
      return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4;
    });
    return 0.2126 * r + 0.7152 * g + 0.0722 * bl;
  };
  const [hi, lo] = [lum(a), lum(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

/** Variables derived from a few base colours, so tints always follow the theme's own accent. */
function derive(v: Record<string, string>): Record<string, string> {
  return {
    ...v,
    "accent-soft": rgba(v.accent, 0.14),
    "accent-hl": rgba(v.accent, 0.16),
    "accent-ring": rgba(v.accent, 0.18),
    "accent-faint": rgba(v.accent, 0.08),
    "accent-edge": rgba(v.accent, 0.5),
    "ok-bg": rgba(v.ok, 0.08),
    "ok-edge": rgba(v.ok, 0.4),
    "ok-glow": rgba(v.ok, 0.18),
    "err-bg": rgba(v.err, 0.1),
    "err-edge": rgba(v.err, 0.6),
  };
}

const ansiDark = {
  black: "#1f2a3a", red: "#f07b86", green: "#9ad07a", yellow: "#e6c07b", blue: "#4aa8ee", magenta: "#c792ea", cyan: "#56d4dd", white: "#d3dcea",
  brightBlack: "#5c6b82", brightRed: "#ff939d", brightGreen: "#b4e394", brightYellow: "#f3d398", brightBlue: "#7cc0f5", brightMagenta: "#d9aaf2", brightCyan: "#7ee3ea", brightWhite: "#ffffff",
};
// Darker, more saturated colours so prompt and program output stay readable on a light background.
const ansiLight = {
  black: "#2b3647", red: "#c4283e", green: "#1f7a4a", yellow: "#946200", blue: "#1a64b5", magenta: "#8a3bbd", cyan: "#0b7482", white: "#7a8799",
  brightBlack: "#5d6b80", brightRed: "#d63c52", brightGreen: "#2a8f5a", brightYellow: "#a87500", brightBlue: "#2b78cc", brightMagenta: "#9d4fd0", brightCyan: "#12899a", brightWhite: "#1b2634",
};

const darkBase = {
  strong: "#ffffff", "on-accent": "#06131f", "on-err": "#1a0508", scrim: "rgba(6, 10, 16, 0.62)", shadow: "rgba(0, 0, 0, 0.5)", band: "rgba(255, 255, 255, 0.028)",
  ok: "#9ad07a", err: "#f07b86", "err-text": "#f79aa3", warn: "#e6c07b", "stripe-ok": "#4f9b6e", "dot-off": "#3a4a62", "switch-thumb": "#cfd8e6", "accent-hover": "#4aa8ee",
};
const searchDark = { match: "#5d4e2b", matchBorder: "#8a7440", active: "#e6c07b", activeBorder: "#f3d398" };

export const THEMES: Record<string, ThemeDef> = {
  midnight: {
    name: "Midnight", scheme: "dark", search: searchDark,
    vars: derive({
      ...darkBase,
      strip: "#0f1620", rail: "#16202d", bar: "#131c28", bg: "#19232f", term: "#1b2634", line: "#243044", line2: "#2a3649", field: "#111a25", fieldline: "#58667b",
      float: "#1a2534", floatline: "#34445c", text: "#d3dcea", text2: "#b3bfd0", muted: "#8d9bb0", dim: "#7f8da2", accent: "#3b9ae1", "accent-text": "#7fb2ee",
      hover: "#1a2535", active: "#1f2b3b", foot: "#172130", divider: "#3d4f6b",
    }),
    term: { ...ansiDark, background: "#1b2634", foreground: "#d3dcea", cursor: "#f07b86", selectionBackground: "#2d4466" },
  },
  graphite: {
    name: "Graphite", scheme: "dark", search: searchDark,
    vars: derive({
      ...darkBase,
      strip: "#111214", rail: "#17181b", bar: "#141517", bg: "#1b1c20", term: "#1e1f24", line: "#2a2c32", line2: "#33353c", field: "#131417", fieldline: "#60626a",
      float: "#202227", floatline: "#3a3d45", text: "#dcdde0", text2: "#b5b7bd", muted: "#8e9098", dim: "#868890", accent: "#4d9de0", "accent-text": "#86b8ec",
      hover: "#202126", active: "#272930", foot: "#1a1b1f", divider: "#44474f", "accent-hover": "#5eaaea", "dot-off": "#474a52",
    }),
    term: { ...ansiDark, black: "#2a2c32", background: "#1e1f24", foreground: "#dcdde0", cursor: "#f07b86", selectionBackground: "#3a4150", brightBlack: "#6a6d76" },
  },
  obsidian: {
    name: "Obsidian", scheme: "dark", search: searchDark,
    vars: derive({
      ...darkBase,
      strip: "#000000", rail: "#0b0b0b", bar: "#050505", bg: "#0a0a0a", term: "#0b0b0c", line: "#1c1c1e", line2: "#262629", field: "#070707", fieldline: "#5c5c5f",
      float: "#141415", floatline: "#303033", text: "#e6e6e6", text2: "#bdbdbd", muted: "#8a8a8e", dim: "#7e7e82", accent: "#2fa3d9", "accent-text": "#6cc4ec",
      hover: "#121213", active: "#19191b", foot: "#0d0d0e", divider: "#2e2e32", "accent-hover": "#44b2e4", "dot-off": "#3a3a3e",
    }),
    term: { ...ansiDark, black: "#1c1c1e", background: "#0b0b0c", foreground: "#e6e6e6", cursor: "#2fa3d9", selectionBackground: "#26384a", brightBlack: "#5e5e63" },
  },
  daylight: {
    name: "Daylight", scheme: "light",
    search: { match: "#f7e7b0", matchBorder: "#d6b13f", active: "#f2c94c", activeBorder: "#a8790a" },
    vars: derive({
      strong: "#0f1a28", "on-accent": "#ffffff", "on-err": "#ffffff", scrim: "rgba(20, 30, 45, 0.38)", shadow: "rgba(20, 35, 60, 0.2)", band: "rgba(15, 26, 40, 0.035)",
      ok: "#1f7a4a", err: "#c4283e", "err-text": "#b3203a", warn: "#946200", "stripe-ok": "#3da06a", "dot-off": "#b4bfcf", "switch-thumb": "#ffffff", "accent-hover": "#1a69b0",
      strip: "#e4e9f0", rail: "#eef2f7", bar: "#e8edf3", bg: "#f5f7fa", term: "#fbfcfe", line: "#dce3ec", line2: "#d2dae6", field: "#ffffff", fieldline: "#8995a6",
      float: "#ffffff", floatline: "#c9d2df", text: "#1b2634", text2: "#364559", muted: "#536175", dim: "#5c6a7e", accent: "#1f72bd", "accent-text": "#1a64aa",
      hover: "#e8eef5", active: "#dde8f4", foot: "#edf1f6", divider: "#c0cad9",
    }),
    term: { ...ansiLight, background: "#fbfcfe", foreground: "#1b2634", cursor: "#c4283e", cursorAccent: "#fbfcfe", selectionBackground: "#cfe2f6" },
  },
};
