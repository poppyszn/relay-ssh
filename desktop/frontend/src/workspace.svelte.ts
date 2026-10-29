import { tick } from "svelte";
import { openSession, type Session } from "./terminal";
import * as P from "./panes";
import { reorder } from "./lib";
import type { BlockRef } from "./blocks";
import { TerminalService } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

export interface Tab { id: string; root: P.Node; focus: string }
/** `connecting` is true from the start of a saved connection until ssh first writes anything (a prompt, a banner or an error). */
export interface PaneInfo { target: string | null; ended: boolean; error: string; blocks: boolean; connecting: boolean; since: number }

let seq = 0;
const uid = (p: string) => `${p}${++seq}`;

/** Tabs of split panes. Each pane owns a persistent DOM element and one session. */
class Workspace {
  tabs = $state<Tab[]>([]);
  activeId = $state("");
  /** The pane whose find bar is open, if any. */
  searchPane = $state("");
  /** The open block menu (raw: it holds functions and a DOMRect, which need no reactivity). */
  blockMenu = $state.raw<{ block: BlockRef; rect: DOMRect } | null>(null);
  panes = $state<Record<string, PaneInfo>>({});
  private els = new Map<string, HTMLElement>();
  private sessions = new Map<string, Session>();

  get active(): Tab | undefined {
    return this.tabs.find((t) => t.id === this.activeId);
  }
  get focusedPane(): string {
    return this.active?.focus ?? "";
  }
  session(id: string): Session | undefined {
    return this.sessions.get(id);
  }
  element(id: string): HTMLElement {
    let el = this.els.get(id);
    if (!el) {
      el = document.createElement("div");
      el.className = "pane-term";
      this.els.set(id, el);
    }
    return el;
  }

  tabTitle(t: Tab): string {
    const names = [...new Set(P.leaves(t.root).map((id) => this.panes[id]?.target ?? "Local"))];
    return names.length > 2 ? `${names[0]} +${names.length - 1}` : names.join(" + ");
  }

  async openTab(target: string | null) {
    const id = uid("p");
    const tab: Tab = { id: uid("t"), root: { id }, focus: id };
    this.panes[id] = { target, ended: false, error: "", blocks: false, connecting: false, since: 0 };
    this.tabs.push(tab);
    this.activeId = tab.id;
    await this.start(id);
  }

  /** Splits the focused pane of the active tab; the new pane connects to `target` (default: same as the focused one). */
  async split(dir: "row" | "col", target?: string | null) {
    const tab = this.active;
    if (!tab) return this.openTab(target ?? null);
    const id = uid("p");
    this.panes[id] = { target: target !== undefined ? target : (this.panes[tab.focus]?.target ?? null), ended: false, error: "", blocks: false, connecting: false, since: 0 };
    tab.root = P.splitLeaf(tab.root, tab.focus, id, dir, uid("s"));
    tab.focus = id;
    await this.start(id);
  }

  /** Splits pane `paneId` toward an edge; the new pane connects to `target` and sits on that side. */
  async splitAt(paneId: string, edge: "left" | "right" | "top" | "bottom", target: string | null) {
    const tab = this.tabs.find((t) => P.leaves(t.root).includes(paneId));
    if (!tab) return;
    const id = uid("p");
    this.panes[id] = { target, ended: false, error: "", blocks: false, connecting: false, since: 0 };
    tab.root = P.splitLeaf(tab.root, paneId, id, edge === "left" || edge === "right" ? "row" : "col", uid("s"), edge === "left" || edge === "top");
    tab.focus = id;
    this.activeId = tab.id;
    await this.start(id);
  }

  /** Moves a tab into the gap before position `slot`. */
  moveTab(tabId: string, slot: number) {
    const from = this.tabs.findIndex((t) => t.id === tabId);
    if (from < 0) return;
    this.tabs = reorder(this.tabs, from, slot);
  }

  /** Opens the session once the pane element is attached and has a size. */
  private async start(id: string) {
    await tick();
    const info = this.panes[id];
    if (!info) return;
    info.ended = false;
    info.error = "";
    info.blocks = false;
    // Local shells start at once; only a saved connection has a login to wait for.
    info.connecting = info.target !== null;
    info.since = Date.now();
    this.sessions.get(id)?.dispose();
    this.els.get(id)?.replaceChildren();
    try {
      const s = await openSession(this.element(id), info.target ?? "local");
      s.ws.addEventListener("close", () => {
        if (this.sessions.get(id) === s && this.panes[id]) {
          this.panes[id].ended = true;
          this.panes[id].connecting = false;
        }
      });
      s.onFirstData = () => { if (this.sessions.get(id) === s && this.panes[id]) this.panes[id].connecting = false; };
      if (s.received) info.connecting = false;
      s.term.onData((d) => {
        if (this.sessions.get(id) === s && this.panes[id]?.ended && d.includes("\r")) this.start(id);
      });
      s.onBlockMenu = (block, rect) => { this.blockMenu = { block, rect }; };
      s.onBlocks = () => { if (this.panes[id]) this.panes[id].blocks = true; };
      this.sessions.set(id, s);
      if (this.focusedPane === id) s.term.focus();
    } catch (e) {
      info.error = String(e);
      info.ended = true;
      info.connecting = false;
    }
  }

  /** Gives up on a connection that is still waiting for the host. */
  cancelConnect(id: string) {
    const info = this.panes[id];
    if (!info?.connecting) return;
    info.error = "Cancelled before the host answered.";
    info.connecting = false;
    this.sessions.get(id)?.dispose();
  }

  /** Types the shell-integration snippet into a remote shell, on the user's request. */
  async enableBlocks(id: string) {
    const s = this.sessions.get(id);
    if (!s || this.panes[id]?.ended) return;
    const snippet = await TerminalService.IntegrationSnippet();
    s.ws.send(new TextEncoder().encode(snippet));
    s.term.focus();
  }

  reconnect(id: string) {
    return this.start(id);
  }

  toggleSearch() {
    const id = this.focusedPane;
    this.searchPane = !id || this.searchPane === id ? "" : id;
  }
  closeSearch() {
    const id = this.searchPane;
    this.searchPane = "";
    if (id) this.sessions.get(id)?.term.focus();
  }

  /** Pushes the current theme, font and density to every open terminal. */
  applySettings() {
    for (const s of this.sessions.values()) s.applySettings();
  }

  focus(id: string) {
    const tab = this.tabs.find((t) => P.leaves(t.root).includes(id));
    if (!tab) return;
    tab.focus = id;
    this.activeId = tab.id;
    this.sessions.get(id)?.term.focus();
  }

  closePane(id: string) {
    const tab = this.tabs.find((t) => P.leaves(t.root).includes(id));
    if (!tab) return;
    const next = P.neighbor(tab.root, id);
    const root = P.removeLeaf(tab.root, id);
    this.sessions.get(id)?.dispose();
    this.sessions.delete(id);
    this.els.delete(id);
    delete this.panes[id];
    if (this.searchPane === id) this.searchPane = "";
    if (!root || !next) return this.dropTab(tab.id);
    tab.root = root;
    if (tab.focus === id) tab.focus = next;
    this.focus(tab.focus);
  }

  closeTab(tabId: string) {
    const tab = this.tabs.find((t) => t.id === tabId);
    if (!tab) return;
    for (const id of P.leaves(tab.root)) {
      this.sessions.get(id)?.dispose();
      this.sessions.delete(id);
      this.els.delete(id);
      delete this.panes[id];
    }
    this.dropTab(tabId);
  }

  private dropTab(tabId: string) {
    const i = this.tabs.findIndex((t) => t.id === tabId);
    if (i < 0) return;
    this.tabs.splice(i, 1);
    if (this.activeId === tabId) this.activeId = this.tabs[Math.min(i, this.tabs.length - 1)]?.id ?? "";
    const f = this.active?.focus;
    if (f) tick().then(() => this.sessions.get(f)?.term.focus());
  }

  select(tabId: string) {
    this.activeId = tabId;
    const f = this.active?.focus;
    if (f) tick().then(() => this.sessions.get(f)?.term.focus());
  }

  cycle(delta: number) {
    if (this.tabs.length < 2) return;
    const i = this.tabs.findIndex((t) => t.id === this.activeId);
    this.select(this.tabs[(i + delta + this.tabs.length) % this.tabs.length].id);
  }

  setRatio(sid: string, ratio: number) {
    const tab = this.active;
    if (tab) tab.root = P.setRatio(tab.root, sid, ratio);
  }
}

export const ws = new Workspace();
