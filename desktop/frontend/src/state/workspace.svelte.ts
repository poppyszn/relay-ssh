import { tick } from "svelte";
import type { Terminal } from "@xterm/xterm";
import { openSession, type Session } from "../lib/terminal";
import * as P from "../lib/panes";
import { blocksWanted, loginQuiet, parsePasswordPrompt, PromptTracker, type PasswordPrompt, reorder, snippetText } from "../lib/lib";
import { settings } from "./settings.svelte";
import type { BlockRef } from "../lib/blocks";
import { TerminalService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

export interface Tab { id: string; root: P.Node; focus: string }
/** `connecting` is true from the start of a saved connection until ssh first writes anything (a prompt, a banner or an error). */
export interface PaneInfo {
  target: string | null;
  ended: boolean;
  error: string;
  blocks: boolean;
  connecting: boolean;
  since: number;
  /** The user closed the "session ended" banner. */
  bannerHidden?: boolean;
  filled?: boolean;
  fillHint?: string;
}

/** The password prompt ssh is showing at the cursor (or the line above when the cursor is on a blank one). */
function promptOnScreen(term: Terminal): PasswordPrompt {
  const buf = term.buffer.active;
  const line = (y: number) => buf.getLine(y)?.translateToString(true) ?? "";
  const here = line(buf.baseY + buf.cursorY);
  const found = parsePasswordPrompt(here);
  if (found || here.trim() || buf.cursorY === 0) return found;
  return parsePasswordPrompt(line(buf.baseY + buf.cursorY - 1));
}

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
  /** A connection's own command-block mode ("on", "off" or ""), supplied by the app. */
  blocksMode: (target: string) => string | undefined = () => undefined;
  /** A connection's auth method ("agent", "key" or "password"), supplied by the app. */
  profileAuth: (target: string) => string | undefined = () => undefined;
  /** A connection's protocol ("ssh" or "rdp"), supplied by the app. */
  profileProtocol: (target: string) => string | undefined = () => undefined;
  private els = new Map<string, HTMLElement>();
  private sessions = new Map<string, Session>();

  isRdp(target: string | null): boolean {
    return target !== null && this.profileProtocol(target) === "rdp";
  }

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
    this.panes[id] = { target, ended: false, error: "", blocks: false, connecting: false, since: 0, filled: false, fillHint: "" };
    this.tabs.push(tab);
    this.activeId = tab.id;
    await this.start(id);
  }

  /** Splits the focused pane of the active tab; the new pane connects to `target` (default: same as the focused one). */
  async split(dir: "row" | "col", target?: string | null) {
    const tab = this.active;
    if (!tab) return this.openTab(target ?? null);
    const id = uid("p");
    this.panes[id] = { target: target !== undefined ? target : (this.panes[tab.focus]?.target ?? null), ended: false, error: "", blocks: false, connecting: false, since: 0, filled: false, fillHint: "" };
    tab.root = P.splitLeaf(tab.root, tab.focus, id, dir, uid("s"));
    tab.focus = id;
    await this.start(id);
  }

  /** Splits pane `paneId` toward an edge; the new pane connects to `target` and sits on that side. */
  async splitAt(paneId: string, edge: "left" | "right" | "top" | "bottom", target: string | null) {
    const tab = this.tabs.find((t) => P.leaves(t.root).includes(paneId));
    if (!tab) return;
    const id = uid("p");
    this.panes[id] = { target, ended: false, error: "", blocks: false, connecting: false, since: 0, filled: false, fillHint: "" };
    tab.root = P.splitLeaf(tab.root, paneId, id, edge === "left" || edge === "right" ? "row" : "col", uid("s"), edge === "left" || edge === "top");
    tab.focus = id;
    this.activeId = tab.id;
    await this.start(id);
  }

  /** Moves pane `id` beside pane `target` (or swaps them for "center"); the session keeps running. */
  movePane(id: string, target: string, edge: "left" | "right" | "top" | "bottom" | "center") {
    const tab = this.tabs.find((t) => P.leaves(t.root).includes(id));
    if (!tab || !P.leaves(tab.root).includes(target)) return;
    const root = edge === "center" ? P.swapLeaves(tab.root, id, target) : P.movePane(tab.root, id, target, edge, () => uid("s"));
    if (!root) return;
    tab.root = root;
    tab.focus = id;
    tick().then(() => this.sessions.get(id)?.term.focus());
  }

  /** Moves pane `id` into another tab (beside that tab's focused pane), or into a new tab. The session keeps running. */
  movePaneToTab(id: string, tabId: string | "new") {
    const src = this.tabs.find((t) => P.leaves(t.root).includes(id));
    if (!src || tabId === src.id) return;
    const rest = P.removeLeaf(src.root, id);
    let dest: Tab | undefined;
    if (tabId === "new") {
      if (!rest) return; // it is already alone in its tab
      dest = { id: uid("t"), root: { id }, focus: id };
      this.tabs.push(dest);
    } else {
      dest = this.tabs.find((t) => t.id === tabId);
      if (!dest) return;
      dest.root = P.insertBeside(dest.root, dest.focus, { id }, "right", () => uid("s"));
      dest.focus = id;
    }
    if (rest) {
      src.root = rest;
      if (src.focus === id) src.focus = P.leaves(rest)[0];
    } else {
      this.tabs.splice(this.tabs.indexOf(src), 1);
    }
    this.activeId = dest.id;
    tick().then(() => this.sessions.get(id)?.term.focus());
  }

  /** Folds tab `tabId`, with all its panes, into the tab that holds pane `targetId`, on the given side of it. */
  mergeTab(tabId: string, targetId: string, edge: "left" | "right" | "top" | "bottom") {
    const src = this.tabs.find((t) => t.id === tabId);
    const dest = this.tabs.find((t) => P.leaves(t.root).includes(targetId));
    if (!src || !dest || src === dest) return;
    dest.root = P.insertBeside(dest.root, targetId, src.root, edge, () => uid("s"));
    dest.focus = src.focus;
    this.tabs.splice(this.tabs.indexOf(src), 1);
    this.activeId = dest.id;
    tick().then(() => this.sessions.get(dest.focus)?.term.focus());
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
    info.bannerHidden = false;
    info.blocks = false;
    // Local shells start at once; only a saved connection has a login to wait for.
    info.connecting = info.target !== null;
    info.since = Date.now();
    this.sessions.get(id)?.dispose();
    this.els.get(id)?.replaceChildren();
    if (this.isRdp(info.target)) {
      info.connecting = false;
      return;
    }
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
      const prompts = new PromptTracker();
      s.onFilled = () => {
        if (this.panes[id]) this.panes[id].filled = true;
      };
      s.onFillRefused = (why) => {
        if (why === "changed" && this.panes[id]) {
          this.panes[id].fillHint = "This connection's host, user or port changed since its password was saved, so it was not filled. Edit the connection and enter the password again.";
        }
      };
      s.term.onWriteParsed(() => {
        if (s.authed || !info.target || this.profileAuth(info.target) !== "password") return;
        const act = prompts.observe(promptOnScreen(s.term));
        if (act === "rejected") {
          if (this.panes[id] && !this.panes[id].fillHint) this.panes[id].fillHint = "The stored password was rejected. Update it in the connection's settings.";
        } else if (act === "send" && s.ws.readyState === WebSocket.OPEN) {
          s.ws.send(JSON.stringify({ t: "fill", ...promptOnScreen(s.term)! }));
        }
      });
      this.autoBlocks(id, s);
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

  /** Whether this pane's session will set up command blocks by itself. */
  autoBlocksFor(target: string | null): boolean {
    return target !== null && blocksWanted(this.blocksMode(target), settings.autoBlocks);
  }

  /**
   * Types the command-block setup line into an ssh session once the login has authenticated and
   * gone quiet: never earlier, because before that the keystrokes could land in a password prompt.
   */
  private autoBlocks(id: string, s: Session) {
    const info = this.panes[id];
    if (!info || !this.autoBlocksFor(info.target)) return;
    const timer = setInterval(() => {
      const live = this.sessions.get(id) === s && this.panes[id] && !this.panes[id].ended;
      if (!live || this.panes[id].blocks || Date.now() - info.since > 120_000) return clearInterval(timer);
      if (loginQuiet(s.authed, s.authedAt, s.lastData, Date.now())) {
        clearInterval(timer);
        this.enableBlocks(id);
      }
    }, 200);
    s.ws.addEventListener("close", () => clearInterval(timer));
  }

  /** Gives up on a connection that is still waiting for the host. */
  cancelConnect(id: string) {
    const info = this.panes[id];
    if (!info?.connecting) return;
    info.error = "Cancelled before the host answered.";
    info.connecting = false;
    this.sessions.get(id)?.dispose();
  }

  /** Types a saved snippet into the focused pane (without Enter unless `run`). False when there is no live pane. */
  typeSnippet(command: string, run: boolean): boolean {
    const id = this.focusedPane;
    const s = this.sessions.get(id);
    if (!s || !this.panes[id] || this.panes[id].ended) return false;
    s.term.paste(snippetText(command)); // bracketed when the shell asks for it, so multi-line text is not run line by line
    if (run) s.term.input("\r", true);
    s.term.focus();
    return true;
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
