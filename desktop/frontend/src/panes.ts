// A tab's panes form a binary tree: leaves are terminals, splits divide space.
export interface Leaf { id: string }
export interface Split { sid: string; dir: "row" | "col"; ratio: number; a: Node; b: Node }
export type Node = Leaf | Split;

export const isLeaf = (n: Node): n is Leaf => "id" in n;

export function leaves(n: Node): string[] {
  return isLeaf(n) ? [n.id] : [...leaves(n.a), ...leaves(n.b)];
}

/** Replaces leaf `id` with a split holding it and a new leaf `newId`, which goes second unless `before`. */
export function splitLeaf(n: Node, id: string, newId: string, dir: "row" | "col", sid: string, before = false): Node {
  if (isLeaf(n)) return n.id !== id ? n : { sid, dir, ratio: 0.5, ...(before ? { a: { id: newId }, b: n } : { a: n, b: { id: newId } }) };
  return { ...n, a: splitLeaf(n.a, id, newId, dir, sid, before), b: splitLeaf(n.b, id, newId, dir, sid, before) };
}

/** Removes leaf `id`; its parent collapses into the sibling. Null if it was the last pane. */
export function removeLeaf(n: Node, id: string): Node | null {
  if (isLeaf(n)) return n.id === id ? null : n;
  const a = removeLeaf(n.a, id);
  const b = removeLeaf(n.b, id);
  if (!a) return b;
  if (!b) return a;
  return { ...n, a, b };
}

/** The pane to focus once `id` goes away: the one before it, else the one after. */
export function neighbor(n: Node, id: string): string | undefined {
  const all = leaves(n);
  const i = all.indexOf(id);
  return i < 0 || all.length < 2 ? undefined : (all[i - 1] ?? all[i + 1]);
}

export function setRatio(n: Node, sid: string, ratio: number): Node {
  if (isLeaf(n)) return n;
  const r = Math.min(0.85, Math.max(0.15, ratio));
  return n.sid === sid ? { ...n, ratio: r } : { ...n, a: setRatio(n.a, sid, ratio), b: setRatio(n.b, sid, ratio) };
}
