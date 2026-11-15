/** Persistent workspace & UI preferences stored locally so restarts restore user context cleanly. */

export interface SavedTab {
  /** Connection profile ID, or null for a local shell. */
  target: string | null;
}

interface SavedUI {
  sidebarOpen: boolean;
  sidebarWidth: number;
  selectedFolder: string;
  recentIds: string[];
  sessionTabs: SavedTab[];
  sessionActive: number;
}

const KEY = "relay.ui.v1";

function load(): Partial<SavedUI> {
  try {
    return JSON.parse(localStorage.getItem(KEY) ?? "{}");
  } catch {
    return {};
  }
}

export const clampSidebarWidth = (w: number): number => Math.min(400, Math.max(220, Math.round(w)));

class UIState {
  sidebarOpen = $state(true);
  sidebarWidth = $state(268);
  selectedFolder = $state("");
  recentIds = $state<string[]>([]);
  sessionTabs = $state<SavedTab[]>([]);
  sessionActive = $state(0);

  constructor() {
    const s = load();
    if (typeof s.sidebarOpen === "boolean") this.sidebarOpen = s.sidebarOpen;
    if (typeof s.sidebarWidth === "number" && Number.isFinite(s.sidebarWidth)) this.sidebarWidth = clampSidebarWidth(s.sidebarWidth);
    if (typeof s.selectedFolder === "string") this.selectedFolder = s.selectedFolder;
    if (Array.isArray(s.recentIds)) this.recentIds = s.recentIds.filter((x): x is string => typeof x === "string").slice(0, 8);
    if (Array.isArray(s.sessionTabs)) {
      this.sessionTabs = s.sessionTabs
        .filter((x): x is SavedTab => x !== null && typeof x === "object" && (x.target === null || typeof x.target === "string"))
        .slice(0, 24);
    }
    if (typeof s.sessionActive === "number" && s.sessionActive >= 0) this.sessionActive = s.sessionActive;
  }

  private persist() {
    try {
      const payload: SavedUI = {
        sidebarOpen: this.sidebarOpen,
        sidebarWidth: this.sidebarWidth,
        selectedFolder: this.selectedFolder,
        recentIds: this.recentIds,
        sessionTabs: this.sessionTabs,
        sessionActive: this.sessionActive,
      };
      localStorage.setItem(KEY, JSON.stringify(payload));
    } catch {
      // storage may be unavailable
    }
  }

  setSidebarOpen(open: boolean) {
    this.sidebarOpen = open;
    this.persist();
  }

  toggleSidebar() {
    this.setSidebarOpen(!this.sidebarOpen);
  }

  setSidebarWidth(w: number) {
    this.sidebarWidth = clampSidebarWidth(w);
    this.persist();
  }

  setSelectedFolder(path: string) {
    this.selectedFolder = path;
    this.persist();
  }

  recordConnection(id: string) {
    if (!id) return;
    this.recentIds = [id, ...this.recentIds.filter((x) => x !== id)].slice(0, 8);
    this.persist();
  }

  saveSession(tabs: SavedTab[], activeIndex: number) {
    this.sessionTabs = tabs.slice(0, 24);
    this.sessionActive = Math.max(0, activeIndex);
    this.persist();
  }
}

export const ui = new UIState();
