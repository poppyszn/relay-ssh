import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";

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
