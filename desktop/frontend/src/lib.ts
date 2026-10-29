import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

export interface FolderNode {
  /** Full folder path, "" for profiles with no folder. */
  path: string;
  profiles: Profile[];
}

/** Case-insensitive match over name, host, user, folder and tags. Every
 * whitespace-separated term must match somewhere; "#tag" matches tags by prefix only. */
export function matches(p: Profile, query: string): boolean {
  const terms = query.toLowerCase().split(/\s+/).filter(Boolean);
  const hay = [p.name, p.host, p.user, p.folder].join(" ").toLowerCase();
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
  return { name: "", host: "", user: "", port: 22, auth: "agent", key: "", folder: "", tags: [], jump: "", forwardAgent: false, tunnels: [] } as Profile;
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
    const p = profiles.find((x) => x.name === id);
    return p && p.folder !== target ? { ok: true } : { ok: false, reason: "Already there" };
  }
  if (within(target, id)) return { ok: false, reason: "A folder cannot go inside itself" };
  if (folderParent(id) === target) return { ok: false, reason: "Already there" };
  if (folders.includes(joinFolder(target, folderBase(id)))) return { ok: false, reason: `A folder named ${folderBase(id)} already exists there` };
  return { ok: true };
}
