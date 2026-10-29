import type { ITheme } from "@xterm/xterm";

export interface ThemeDef {
  name: string;
  /** CSS custom properties applied to :root (without the leading --). */
  vars: Record<string, string>;
  term: ITheme;
}

const ansi = {
  black: "#1f2a3a", red: "#f07b86", green: "#9ad07a", yellow: "#e6c07b", blue: "#4aa8ee", magenta: "#c792ea", cyan: "#56d4dd", white: "#d3dcea",
  brightBlack: "#5c6b82", brightRed: "#ff939d", brightGreen: "#b4e394", brightYellow: "#f3d398", brightBlue: "#7cc0f5", brightMagenta: "#d9aaf2", brightCyan: "#7ee3ea", brightWhite: "#ffffff",
};

export const THEMES: Record<string, ThemeDef> = {
  midnight: {
    name: "Midnight",
    vars: {
      strip: "#0f1620", rail: "#16202d", bar: "#131c28", bg: "#19232f", term: "#1b2634", line: "#243044", line2: "#2a3649", field: "#111a25", fieldline: "#2a384d",
      float: "#1a2534", floatline: "#34445c", text: "#d3dcea", text2: "#b3bfd0", muted: "#8d9bb0", dim: "#7d8ba0", accent: "#3b9ae1", "accent-text": "#7fb2ee",
      "accent-soft": "rgba(59, 154, 225, 0.14)", hover: "#1a2535", active: "#1f2b3b", foot: "#172130", divider: "#3d4f6b",
    },
    term: { ...ansi, background: "#1b2634", foreground: "#d3dcea", cursor: "#f07b86", selectionBackground: "#2d4466" },
  },
  graphite: {
    name: "Graphite",
    vars: {
      strip: "#111214", rail: "#17181b", bar: "#141517", bg: "#1b1c20", term: "#1e1f24", line: "#2a2c32", line2: "#33353c", field: "#131417", fieldline: "#34363d",
      float: "#202227", floatline: "#3a3d45", text: "#dcdde0", text2: "#b5b7bd", muted: "#8e9098", dim: "#7a7c84", accent: "#4d9de0", "accent-text": "#86b8ec",
      "accent-soft": "rgba(77, 157, 224, 0.15)", hover: "#202126", active: "#272930", foot: "#1a1b1f", divider: "#44474f",
    },
    term: { ...ansi, black: "#2a2c32", background: "#1e1f24", foreground: "#dcdde0", cursor: "#f07b86", selectionBackground: "#3a4150", brightBlack: "#6a6d76" },
  },
  obsidian: {
    name: "Obsidian",
    vars: {
      strip: "#000000", rail: "#0b0b0b", bar: "#050505", bg: "#0a0a0a", term: "#0b0b0c", line: "#1c1c1e", line2: "#262629", field: "#070707", fieldline: "#2a2a2d",
      float: "#141415", floatline: "#303033", text: "#e6e6e6", text2: "#bdbdbd", muted: "#8a8a8e", dim: "#77777b", accent: "#2fa3d9", "accent-text": "#6cc4ec",
      "accent-soft": "rgba(47, 163, 217, 0.15)", hover: "#121213", active: "#19191b", foot: "#0d0d0e", divider: "#2e2e32",
    },
    term: { ...ansi, black: "#1c1c1e", background: "#0b0b0c", foreground: "#e6e6e6", cursor: "#2fa3d9", selectionBackground: "#26384a", brightBlack: "#5e5e63" },
  },
};

export const FONTS: Record<string, { name: string; stack: string }> = {
  jetbrains: { name: "JetBrains Mono", stack: '"JetBrains Mono", "SF Mono", Menlo, Consolas, monospace' },
  system: { name: "System monospace", stack: '"SF Mono", Menlo, "Cascadia Mono", Consolas, monospace' },
};

export const DENSITY: Record<string, { name: string; lineHeight: number }> = {
  compact: { name: "Compact", lineHeight: 1.15 },
  comfortable: { name: "Comfortable", lineHeight: 1.25 },
  airy: { name: "Airy", lineHeight: 1.4 },
};

interface Saved { theme: string; font: string; fontSize: number; density: string; copyOnSelect: boolean; cursorBlink: boolean }
const KEY = "relay.settings.v1";

function load(): Partial<Saved> {
  try {
    return JSON.parse(localStorage.getItem(KEY) ?? "{}");
  } catch {
    return {}; // storage can be unavailable or hold garbage; defaults then
  }
}

class Settings {
  theme = $state("midnight");
  font = $state("jetbrains");
  fontSize = $state(14);
  density = $state("comfortable");
  copyOnSelect = $state(false);
  cursorBlink = $state(true);

  constructor() {
    const s = load();
    if (s.theme && THEMES[s.theme]) this.theme = s.theme;
    if (s.font && FONTS[s.font]) this.font = s.font;
    if (typeof s.fontSize === "number" && s.fontSize >= 10 && s.fontSize <= 24) this.fontSize = s.fontSize;
    if (s.density && DENSITY[s.density]) this.density = s.density;
    if (typeof s.copyOnSelect === "boolean") this.copyOnSelect = s.copyOnSelect;
    if (typeof s.cursorBlink === "boolean") this.cursorBlink = s.cursorBlink;
  }

  get themeDef(): ThemeDef { return THEMES[this.theme]; }

  /** Options xterm needs, for new terminals and live updates alike. */
  get xterm() {
    return {
      theme: this.themeDef.term,
      fontFamily: FONTS[this.font].stack,
      fontSize: this.fontSize,
      lineHeight: DENSITY[this.density].lineHeight,
      cursorBlink: this.cursorBlink,
    };
  }

  /** Applies the theme to the page chrome and persists everything. */
  apply() {
    for (const [k, v] of Object.entries(this.themeDef.vars)) document.documentElement.style.setProperty(`--${k}`, v);
    const saved: Saved = { theme: this.theme, font: this.font, fontSize: this.fontSize, density: this.density, copyOnSelect: this.copyOnSelect, cursorBlink: this.cursorBlink };
    try {
      localStorage.setItem(KEY, JSON.stringify(saved));
    } catch {
      // best effort: settings just will not persist
    }
  }
}

export const settings = new Settings();
