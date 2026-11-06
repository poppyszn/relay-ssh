import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

export interface FolderNode {
  /** Full folder path, "" for profiles with no folder. */
  path: string;
  profiles: Profile[];
}

/** Case-insensitive match over name, host, user, folder and tags. Every
 * whitespace-separated term must match somewhere; "#tag" matches tags by prefix only. */
export function matches(p: Profile, query: string): boolean {
  const terms = query.toLowerCase().split(/\s+/).filter(Boolean);
  // "rdp" (and the Windows domain) find remote desktops; ssh hosts get no protocol word, so "ssh" stays a plain search.
  const hay = [p.name, p.host, p.user, p.folder, ...(p.protocol === "rdp" ? ["rdp", p.domain ?? ""] : [])].join(" ").toLowerCase();
  return terms.every((t) =>
    t.startsWith("#") ? (p.tags ?? []).some((g) => g.toLowerCase().startsWith(t.slice(1))) : hay.includes(t) || (p.tags ?? []).some((g) => g.toLowerCase().includes(t)),
  );
}

/** Groups profiles by folder, folders sorted, unfiled profiles last. */
export function groupByFolder(profiles: Profile[]): FolderNode[] {
  const by = new Map<string, Profile[]>();
  for (const p of profiles) by.set(p.folder, [...(by.get(p.folder) ?? []), p]);
  return [...by.entries()]
    .map(([path, ps]) => ({ path, profiles: ps.sort((a, b) => a.name.localeCompare(b.name)) }))
    .sort((a, b) => (a.path === "" ? 1 : b.path === "" ? -1 : a.path.localeCompare(b.path)));
}

export function blankProfile(): Profile {
  return { id: "", name: "", host: "", user: "", port: 22, auth: "agent", key: "", folder: "", tags: [], jump: "", forwardAgent: false, blocks: "", tunnels: [] } as Profile;
}

/** A host someone pasted into quick open: `ssh [-p N] [user@]host [-p N]`, `ssh://user@host:port`
 * or `user@host[:port]`. Anything else, including any other ssh option, is an ordinary search (null).
 * The result only prefills the new-connection form, which validates it again. */
export function parseTarget(q: string): { user: string; host: string; port: number } | null {
  let s = q.trim();
  const ssh = /^ssh(:\/\/|\s+)/.test(s);
  s = s.replace(/^ssh:\/\//, "").replace(/^ssh\s+/, "");
  let port = 22;
  s = s.replace(/(^|\s)-p\s+(\d+)(?=\s|$)/, (_, pre, n) => ((port = Number(n)), pre)).trim();
  const m = /^(?:([A-Za-z0-9._-]+)@)?([A-Za-z0-9][A-Za-z0-9.-]*)(?::(\d+))?$/.exec(s);
  if (!m || (!ssh && !m[1])) return null;
  if (m[3]) port = Number(m[3]);
  if (port < 1 || port > 65535) return null;
  return { user: m[1] ?? "", host: m[2], port };
}

/** A copy of a connection under a new name that is free (name-copy, name-copy-2, ...), in the same folder with the same
 * settings and its own lists. A saved password is not carried over; the form for the copy starts without one. */
export function duplicateProfile(p: Profile, existing: string[]): Profile {
  const root = p.name.replace(/-copy(-\d+)?$/, "");
  const taken = new Set(existing);
  let name = `${root}-copy`;
  for (let n = 2; taken.has(name); n++) name = `${root}-copy-${n}`;
  return { ...p, id: "", name, tags: [...(p.tags ?? [])], tunnels: (p.tunnels ?? []).map((t) => ({ ...t })) } as Profile;
}

export function parseTags(s: string): string[] {
  return [...new Set(s.split(",").map((t) => t.trim()).filter(Boolean))];
}

// ---- folder tree ----

export const folderParent = (path: string) => path.slice(0, Math.max(0, path.lastIndexOf("/")));
export const folderBase = (path: string) => path.slice(path.lastIndexOf("/") + 1);
export const joinFolder = (dir: string, name: string) => (dir ? `${dir}/${name}` : name);
const within = (path: string, folder: string) => path === folder || path.startsWith(folder + "/");

export interface TreeNode {
  path: string;
  name: string;
  children: TreeNode[];
  /** Connections directly in this folder. */
  profiles: Profile[];
  /** Connections in this folder and everything beneath it. */
  count: number;
}

/**
 * Builds the sidebar tree. `folders` is every folder (empty ones and ancestors
 * included). With a query only matching connections stay, and folders that lead
 * to none are dropped; otherwise empty folders are kept.
 */
export function buildTree(profiles: Profile[], folders: string[], query: string): { roots: TreeNode[]; unfiled: Profile[] } {
  const visible = profiles.filter((p) => matches(p, query));
  const nodes = new Map<string, TreeNode>();
  const node = (path: string): TreeNode => {
    let n = nodes.get(path);
    if (!n) nodes.set(path, (n = { path, name: folderBase(path), children: [], profiles: [], count: 0 }));
    return n;
  };
  for (const f of folders) node(f);
  for (const p of visible) if (p.folder) node(p.folder).profiles.push(p);

  const roots: TreeNode[] = [];
  for (const path of [...nodes.keys()].sort()) {
    const n = nodes.get(path)!;
    const parent = folderParent(path);
    (parent && nodes.has(parent) ? nodes.get(parent)!.children : roots).push(n);
  }
  const finish = (n: TreeNode): boolean => {
    n.profiles.sort((a, b) => a.name.localeCompare(b.name));
    n.children = n.children.filter(finish);
    n.count = n.profiles.length + n.children.reduce((s, c) => s + c.count, 0);
    return !query.trim() || n.count > 0;
  };
  return { roots: roots.filter(finish), unfiled: visible.filter((p) => !p.folder).sort((a, b) => a.name.localeCompare(b.name)) };
}

/** Whether dropping an item on a folder ("" is the top level) is allowed, and why not. */
export function canDrop(kind: "profile" | "folder", id: string, target: string, profiles: Profile[], folders: string[]): { ok: boolean; reason?: string } {
  if (kind === "profile") {
    const p = profiles.find((x) => x.id === id);
    return p && p.folder !== target ? { ok: true } : { ok: false, reason: "Already there" };
  }
  if (within(target, id)) return { ok: false, reason: "A folder cannot go inside itself" };
  if (folderParent(id) === target) return { ok: false, reason: "Already there" };
  if (folders.includes(joinFolder(target, folderBase(id)))) return { ok: false, reason: `A folder named ${folderBase(id)} already exists there` };
  return { ok: true };
}

/** "4s", "1m 1s": how long a connection attempt has been running. */
export function formatElapsed(ms: number): string {
  const s = Math.floor(ms / 1000);
  return s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`;
}

/** What to tell the user while ssh is silent: nothing for a few seconds, then that it is slow, then why. */
export function connectingHint(ms: number, viaJump: boolean): string {
  if (ms < 8_000) return "";
  if (ms < 30_000) return viaJump ? "Taking longer than usual. The jump host or the server is slow to answer." : "Taking longer than usual.";
  return viaJump
    ? "Still no answer. The jump host or the server may be unreachable, offline or behind a firewall."
    : "Still no answer. The host may be unreachable, offline or behind a firewall.";
}

/** Moves the item at `from` into the gap before index `slot` (so `arr.length` is the end). */
export function reorder<T>(arr: T[], from: number, slot: number): T[] {
  const out = arr.slice();
  const [item] = out.splice(from, 1);
  out.splice(slot > from ? slot - 1 : slot, 0, item);
  return out;
}

/** The gap a pointer at `x` is over, given the horizontal centre of each tab. */
export function slotAt(centres: number[], x: number): number {
  return centres.filter((c) => c < x).length;
}

export type PaneEdge = "left" | "right" | "top" | "bottom" | "center";

/** Where a drop inside a pane lands: toward the nearest edge, or the middle (unless `middle` is false). */
export function paneEdge(r: { left: number; top: number; width: number; height: number }, x: number, y: number, middle = true): PaneEdge {
  const dx = (x - r.left) / r.width;
  const dy = (y - r.top) / r.height;
  if (middle && dx >= 0.3 && dx <= 0.7 && dy >= 0.3 && dy <= 0.7) return "center";
  const d = { left: dx, right: 1 - dx, top: dy, bottom: 1 - dy };
  return (Object.keys(d) as (keyof typeof d)[]).reduce((best, k) => (d[k] < d[best] ? k : best), "left");
}

/** Whether to set up command blocks by itself: the connection's mode ("on", "off", or empty to follow the app). */
export function blocksWanted(mode: string | undefined, appAutomatic: boolean): boolean {
  return mode === "on" ? true : mode === "off" ? false : appAutomatic;
}

/**
 * True once an ssh login has authenticated (so no password prompt can be waiting) and the
 * session has been silent for `quietMs`: the moment to type the setup line.
 */
export function loginQuiet(authed: boolean, authedAt: number, lastData: number, now: number, quietMs = 700): boolean {
  return authed && now - Math.max(authedAt, lastData) >= quietMs;
}

/** A snippet as it is typed: LF line endings, no trailing blank lines or spaces (the Enter key is separate). */
export function snippetText(command: string): string {
  return command.replace(/\r\n?/g, "\n").trimEnd();
}

/** One line for a list: the first line, shortened, and how many more lines follow. */
export function snippetPreview(command: string): string {
  const lines = snippetText(command).split("\n");
  const first = lines[0].length > 80 ? lines[0].slice(0, 79) + "…" : lines[0];
  return lines.length > 1 ? `${first}  (+${lines.length - 1} line${lines.length === 2 ? "" : "s"})` : first;
}

/** Which open session a click on a connection should show: the first, or the one after the focused pane when it is already on one. */
export function nextSession(ids: string[], focused: string): string | undefined {
  const i = ids.indexOf(focused);
  return ids.length === 0 ? undefined : ids[(i + 1) % ids.length];
}

export type PasswordPrompt =
  | { kind: "account"; user: string; host: string }
  | { kind: "generic" }
  | null;

/**
 * Parses an OpenSSH password prompt line from terminal output.
 * Strictly matches account prompts ("alice@10.0.0.5's password: ") and generic
 * prompts ("Password: " / "(alice@host) Password: "), and rejects passphrases,
 * 2FA, sudo, permission errors, and normal output.
 */
export function parsePasswordPrompt(line: string): PasswordPrompt {
  const trimmed = line.trim();
  if (!trimmed) return null;

  // Reject common non-account or non-ssh prompts explicitly
  if (
    trimmed.startsWith("[sudo]") ||
    trimmed.startsWith("sudo:") ||
    trimmed.includes("passphrase") ||
    trimmed.includes("Verification code") ||
    trimmed.includes("continue connecting") ||
    trimmed.includes("Permission denied")
  ) {
    return null;
  }

  // 1. Account prompt: user@host's password:
  const accountMatch = trimmed.match(
    /^([a-zA-Z0-9_.-]+)@(\[[a-fA-F0-9:.]+\]|[a-zA-Z0-9_.-]+)(?::\d+)?'s password:\s*$/i
  );
  if (accountMatch) {
    const user = accountMatch[1];
    const host = accountMatch[2].replace(/^\[|\]$/g, "");
    return { kind: "account", user, host };
  }

  // 2. Generic prompt: "Password: " or "(user@host) Password: "
  if (/^(?:\([^)]+\)\s+)?password:\s*$/i.test(trimmed)) {
    return { kind: "generic" };
  }

  return null;
}

/**
 * Watches the prompt line of a connection that has not logged in yet and decides what to do
 * about each password prompt: ask Go to fill it ("send"), or flag that a prompt we already
 * answered has come back ("rejected": the stored password was wrong). A prompt that stays on
 * screen across many redraws counts once; each distinct prompt (a jump host, then the target)
 * is answered once.
 */
export class PromptTracker {
  private onScreen = false;
  private answered = new Set<string>();

  observe(prompt: PasswordPrompt): "send" | "rejected" | null {
    if (!prompt) {
      this.onScreen = false;
      return null;
    }
    if (this.onScreen) return null;
    this.onScreen = true;
    const key = prompt.kind === "account" ? `${prompt.user}@${prompt.host}` : "generic";
    if (this.answered.has(key)) return "rejected";
    this.answered.add(key);
    return "send";
  }
}
