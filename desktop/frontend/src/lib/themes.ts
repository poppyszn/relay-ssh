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
function derive(v: Record<string, string>, scheme: "dark" | "light" = "dark"): Record<string, string> {
  const light = scheme === "light";
  return {
    ...v,
    "hairline": light ? "rgba(15, 26, 40, 0.08)" : "rgba(255, 255, 255, 0.06)",
    "float-edge": light ? "rgba(15, 26, 40, 0.12)" : "rgba(255, 255, 255, 0.08)",
    "field-edge": light ? "rgba(15, 26, 40, 0.14)" : "transparent",
    "surface-btn": v["surface-btn"] ?? v.line,
    "surface-btn-hover": v["surface-btn-hover"] ?? v.line2,
    "cta-bg": light ? "#111827" : "#f4f4f5",
    "cta-fg": light ? "#ffffff" : "#09090b",
    "cta-hover": light ? "#1f2937" : "#ffffff",
    "accent-soft": rgba(v.accent, 0.14),
    "accent-hl": rgba(v.accent, 0.16),
    "accent-ring": rgba(v.accent, 0.18),
    "accent-faint": rgba(v.accent, 0.08),
    "accent-edge": rgba(v.accent, 0.45),
    "ok-bg": rgba(v.ok, 0.1),
    "ok-edge": rgba(v.ok, 0.36),
    "ok-glow": rgba(v.ok, 0.28),
    "err-bg": rgba(v.err, 0.12),
    "err-edge": rgba(v.err, 0.45),
  };
}

const ansiDark = {
  black: "#18181c", red: "#fb7185", green: "#4ade80", yellow: "#fbbf24", blue: "#318edc", magenta: "#c084fc", cyan: "#38bdf8", white: "#e4e4e7",
  brightBlack: "#63636c", brightRed: "#fda4af", brightGreen: "#86efac", brightYellow: "#fcd34d", brightBlue: "#7ab2ee", brightMagenta: "#d8b4fe", brightCyan: "#7dd3fc", brightWhite: "#ffffff",
};
// Darker, more saturated colours so prompt and program output stay readable on a light background.
const ansiLight = {
  black: "#2b3647", red: "#c4283e", green: "#1f7a4a", yellow: "#946200", blue: "#1a64b5", magenta: "#8a3bbd", cyan: "#0b7482", white: "#7a8799",
  brightBlack: "#5d6b80", brightRed: "#d63c52", brightGreen: "#2a8f5a", brightYellow: "#a87500", brightBlue: "#2b78cc", brightMagenta: "#9d4fd0", brightCyan: "#12899a", brightWhite: "#1b2634",
};

const darkBase = {
  strong: "#ffffff", "on-accent": "#04121f", "on-err": "#1f050a", scrim: "rgba(0, 0, 0, 0.72)", shadow: "rgba(0, 0, 0, 0.65)", band: "rgba(255, 255, 255, 0.025)",
  ok: "#4ade80", err: "#fb7185", "err-text": "#fda4af", warn: "#fbbf24", "stripe-ok": "#4ade80", "dot-off": "#3f3f46", "switch-thumb": "#e4e4e7", "accent-hover": "#4aa0e8",
};
const searchDark = { match: "#5d4e2b", matchBorder: "#8a7440", active: "#fbbf24", activeBorder: "#fcd34d" };

export const THEMES: Record<string, ThemeDef> = {
  obsidian: {
    name: "Obsidian", scheme: "dark", search: searchDark,
    vars: derive({
      ...darkBase,
      strip: "#040406", rail: "#08080c", bar: "#060608", bg: "#020203", term: "#050507",
      line: "#161720", line2: "#222430", field: "#111218", fieldline: "#636675",
      float: "#0c0d14", floatline: "#272936", foot: "#060608", divider: "#222430",
      "surface-btn": "#14151e", "surface-btn-hover": "#1d1f2b",
      text: "#f4f4f5", text2: "#d4d4d8", muted: "#a1a1aa", dim: "#858591",
      accent: "#318edc", "accent-text": "#7ab2ee", "accent-hover": "#4aa0e8",
      hover: "#111219", active: "#191b24", "dot-off": "#3f3f46",
    }),
    term: { ...ansiDark, background: "#050507", foreground: "#f4f4f5", cursor: "#7ab2ee", selectionBackground: "#1b334b" },
  },
  midnight: {
    name: "Midnight", scheme: "dark", search: searchDark,
    vars: derive({
      ...darkBase,
      strip: "#0f1620", rail: "#16202d", bar: "#131c28", bg: "#19232f", term: "#1b2634", line: "#243044", line2: "#2a3649", field: "#111a25", fieldline: "#58667b",
      float: "#1a2534", floatline: "#34445c", "surface-btn": "#1d2a3a", "surface-btn-hover": "#253448",
      text: "#d3dcea", text2: "#b3bfd0", muted: "#8d9bb0", dim: "#7f8da2", accent: "#318edc", "accent-text": "#7ab2ee", "accent-hover": "#4aa0e8",
      hover: "#1a2535", active: "#1f2b3b", foot: "#172130", divider: "#3d4f6b", "dot-off": "#3a4a62",
    }),
    term: { ...ansiDark, black: "#1f2a3a", background: "#1b2634", foreground: "#d3dcea", cursor: "#fb7185", selectionBackground: "#2d4466" },
  },
  graphite: {
    name: "Graphite", scheme: "dark", search: searchDark,
    vars: derive({
      ...darkBase,
      strip: "#111214", rail: "#17181b", bar: "#141517", bg: "#1b1c20", term: "#1e1f24", line: "#2a2c32", line2: "#33353c", field: "#131417", fieldline: "#60626a",
      float: "#202227", floatline: "#3a3d45", "surface-btn": "#22242a", "surface-btn-hover": "#2a2d35",
      text: "#dcdde0", text2: "#b5b7bd", muted: "#8e9098", dim: "#868890", accent: "#318edc", "accent-text": "#7ab2ee", "accent-hover": "#4aa0e8",
      hover: "#202126", active: "#272930", foot: "#1a1b1f", divider: "#44474f", "dot-off": "#474a52",
    }),
    term: { ...ansiDark, black: "#2a2c32", background: "#1e1f24", foreground: "#dcdde0", cursor: "#7ab2ee", selectionBackground: "#3a4150", brightBlack: "#6a6d76" },
  },
  daylight: {
    name: "Daylight", scheme: "light",
    search: { match: "#f7e7b0", matchBorder: "#d6b13f", active: "#f2c94c", activeBorder: "#a8790a" },
    vars: derive({
      strong: "#0f1a28", "on-accent": "#ffffff", "on-err": "#ffffff", scrim: "rgba(20, 30, 45, 0.38)", shadow: "rgba(20, 35, 60, 0.18)", band: "rgba(15, 26, 40, 0.035)",
      ok: "#1f7a4a", err: "#c4283e", "err-text": "#b3203a", warn: "#946200", "stripe-ok": "#3da06a", "dot-off": "#b4bfcf", "switch-thumb": "#ffffff", "accent-hover": "#1a69b0",
      strip: "#e4e9f0", rail: "#eef2f7", bar: "#e8edf3", bg: "#f5f7fa", term: "#fbfcfe", line: "#dce3ec", line2: "#d2dae6", field: "#ffffff", fieldline: "#8995a6",
      float: "#ffffff", floatline: "#c9d2df", "surface-btn": "#e4e9f0", "surface-btn-hover": "#d7dfe9",
      text: "#1b2634", text2: "#364559", muted: "#536175", dim: "#5c6a7e", accent: "#1f72bd", "accent-text": "#1a64aa",
      hover: "#e8eef5", active: "#dde8f4", foot: "#edf1f6", divider: "#c0cad9",
    }, "light"),
    term: { ...ansiLight, background: "#fbfcfe", foreground: "#1b2634", cursor: "#c4283e", cursorAccent: "#fbfcfe", selectionBackground: "#cfe2f6" },
  },
};
