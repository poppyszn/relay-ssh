import { THEMES, type ThemeDef } from "../lib/themes";
import { VaultService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

export { THEMES };

/** The "system" theme follows the macOS light or dark appearance. */
export const SYSTEM = "system";
const SYSTEM_LIGHT = "daylight";
const SYSTEM_DARK = "midnight";

export const FONTS: Record<string, { name: string; stack: string }> = {
  jetbrains: { name: "JetBrains Mono", stack: '"JetBrains Mono", "SF Mono", Menlo, Consolas, monospace' },
  system: { name: "System monospace", stack: '"SF Mono", Menlo, "Cascadia Mono", Consolas, monospace' },
};

export const DENSITY: Record<string, { name: string; lineHeight: number }> = {
  compact: { name: "Compact", lineHeight: 1.15 },
  comfortable: { name: "Comfortable", lineHeight: 1.25 },
  airy: { name: "Airy", lineHeight: 1.4 },
};

interface Saved {
  theme: string;
  font: string;
  fontSize: number;
  density: string;
  copyOnSelect: boolean;
  cursorBlink: boolean;
  autoBlocks: boolean;
  vaultTimeout?: number;
}
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
  /** Set up command blocks by itself in ssh sessions (a connection can override this). */
  autoBlocks = $state(false);
  /** Inactivity timeout in minutes before auto-locking the secret vault (0 = lock on quit). */
  vaultTimeout = $state(15);
  private sentVaultTimeout = -1;

  constructor() {
    const s = load();
    if (s.theme && (THEMES[s.theme] || s.theme === SYSTEM)) this.theme = s.theme;
    if (s.font && FONTS[s.font]) this.font = s.font;
    if (typeof s.fontSize === "number" && s.fontSize >= 10 && s.fontSize <= 24) this.fontSize = s.fontSize;
    if (s.density && DENSITY[s.density]) this.density = s.density;
    if (typeof s.copyOnSelect === "boolean") this.copyOnSelect = s.copyOnSelect;
    if (typeof s.cursorBlink === "boolean") this.cursorBlink = s.cursorBlink;
    if (typeof s.autoBlocks === "boolean") this.autoBlocks = s.autoBlocks;
    if (typeof s.vaultTimeout === "number" && s.vaultTimeout >= 0) this.vaultTimeout = s.vaultTimeout;
  }

  /** Whether macOS is currently in light mode; only consulted for the "system" theme. */
  systemLight = $state(typeof matchMedia === "function" && matchMedia("(prefers-color-scheme: light)").matches);

  /** The theme actually in use: "system" resolves to Daylight or Midnight. */
  get resolved(): string {
    return this.theme === SYSTEM ? (this.systemLight ? SYSTEM_LIGHT : SYSTEM_DARK) : this.theme;
  }
  get themeDef(): ThemeDef { return THEMES[this.resolved]; }
  /** Name for the status bar. */
  get label(): string {
    return this.theme === SYSTEM ? `System (${THEMES[this.resolved].name})` : this.themeDef.name;
  }

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
    const root = document.documentElement;
    for (const [k, v] of Object.entries(this.themeDef.vars)) root.style.setProperty(`--${k}`, v);
    root.style.colorScheme = this.themeDef.scheme; // native scrollbars, inputs and the selection colour follow
    const saved: Saved = {
      theme: this.theme,
      font: this.font,
      fontSize: this.fontSize,
      density: this.density,
      copyOnSelect: this.copyOnSelect,
      cursorBlink: this.cursorBlink,
      autoBlocks: this.autoBlocks,
      vaultTimeout: this.vaultTimeout,
    };
    try {
      localStorage.setItem(KEY, JSON.stringify(saved));
    } catch {
      // best effort: settings just will not persist
    }
    if (import.meta.env.MODE !== "test" && this.vaultTimeout !== this.sentVaultTimeout) {
      this.sentVaultTimeout = this.vaultTimeout;
      VaultService.SetIdleTimeout(this.vaultTimeout).catch(() => {});
    }
  }
}

export const settings = new Settings();

// Follow macOS appearance changes while the "system" theme is selected.
if (typeof matchMedia === "function") {
  matchMedia("(prefers-color-scheme: light)").addEventListener("change", (e) => (settings.systemLight = e.matches));
}
