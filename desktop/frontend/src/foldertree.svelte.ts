import { ProfilesService, type Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
import { canDrop, folderBase, folderParent, joinFolder, paneEdge, type PaneEdge } from "./lib";

export type DragKind = "profile" | "folder";
export interface Pending { kind: DragKind; id: string; to: string; text: string; sub: string }

const KEY = "relay.sidebar.collapsed.v1";
function loadCollapsed(): Record<string, boolean> {
  try {
    return JSON.parse(localStorage.getItem(KEY) ?? "{}");
  } catch {
    return {}; // storage may be unavailable; everything starts expanded
  }
}
const msg = (e: unknown) => (e as Error)?.message ?? String(e);

/** Shared state and operations for the connection tree: folders, drag and drop, confirmations. */
class SidebarState {
  collapsed = $state<Record<string, boolean>>(loadCollapsed());
  /** The folder new subfolders and connections go into; "" is the top level. */
  selectedFolder = $state("");
  /** Parent path of the inline "new folder" row, or null when none is open. */
  creatingIn = $state<string | null>(null);
  newName = $state("");
  renaming = $state("");
  renameTo = $state("");
  menuFolder = $state("");
  confirmDelete = $state("");
  drag = $state<{ kind: DragKind; id: string; label: string; x: number; y: number } | null>(null);
  dropTarget = $state<string | null>(null);
  /** The terminal pane a dragged connection is over, and which part of it. */
  paneDrop = $state<{ id: string; edge: PaneEdge } | null>(null);
  /** Called when a connection is dropped on a pane (set by the app). */
  onPaneDrop: (connection: string, paneId: string, edge: PaneEdge) => void = () => {};
  dropOk = $state(true);
  dropReason = $state("");
  pending = $state<Pending | null>(null);
  error = $state("");

  data = { profiles: [] as Profile[], folders: [] as string[] };
  onchanged: () => void = () => {};
  /** True just after a drag ends, so the click that follows it is ignored. */
  justDragged = false;

  private save() {
    try {
      localStorage.setItem(KEY, JSON.stringify(this.collapsed));
    } catch {
      // best effort
    }
  }
  expanded(path: string, searching = false) {
    return searching || !this.collapsed[path];
  }
  toggle(path: string) {
    this.collapsed[path] = !this.collapsed[path];
    this.save();
  }
  expand(path: string) {
    // a folder and every ancestor, so something just created or moved is visible
    for (let p = path; p; p = folderParent(p)) this.collapsed[p] = false;
    this.save();
  }
  select(path: string) {
    this.selectedFolder = path;
  }

  private async run(fn: () => Promise<void>) {
    this.error = "";
    try {
      await fn();
      this.onchanged();
    } catch (e) {
      this.error = msg(e);
    }
  }

  // ---- folders ----
  startCreate(parent: string) {
    this.creatingIn = parent;
    this.newName = "";
    this.menuFolder = "";
    if (parent) this.expand(parent);
  }
  async commitCreate() {
    const parent = this.creatingIn;
    const name = this.newName.trim();
    this.creatingIn = null;
    if (parent === null || !name) return;
    if (name.includes("/")) return void (this.error = "A folder name cannot contain '/'. Use Add, then Subfolder, to nest one.");
    const path = joinFolder(parent, name);
    await this.run(async () => {
      await ProfilesService.CreateFolder(path);
      this.expand(path);
      this.selectedFolder = path;
    });
  }
  startRename(path: string) {
    this.renaming = path;
    this.renameTo = folderBase(path);
    this.menuFolder = "";
  }
  async commitRename() {
    const from = this.renaming;
    const name = this.renameTo.trim();
    this.renaming = "";
    if (!from || !name || name === folderBase(from)) return;
    if (name.includes("/")) return void (this.error = "A folder name cannot contain '/'.");
    const to = joinFolder(folderParent(from), name);
    await this.run(async () => {
      await ProfilesService.RenameFolder(from, to);
      if (this.collapsed[from]) this.collapsed[to] = true;
      if (this.selectedFolder === from) this.selectedFolder = to;
    });
  }
  async remove(path: string) {
    this.confirmDelete = "";
    await this.run(async () => {
      await ProfilesService.DeleteFolder(path);
      if (this.selectedFolder === path) this.selectedFolder = "";
    });
  }

  // ---- drag and drop ----
  beginDrag(kind: DragKind, id: string, label: string) {
    this.drag = { kind, id, label, x: 0, y: 0 };
    this.pending = null;
    this.menuFolder = "";
    document.body.classList.add("dragging");
  }
  moveDrag(x: number, y: number) {
    if (!this.drag) return;
    this.drag.x = x;
    this.drag.y = y;
    const scroller = document.querySelector<HTMLElement>("[data-scroll]");
    if (scroller) {
      const r = scroller.getBoundingClientRect();
      if (y > r.top && y < r.top + 36) scroller.scrollTop -= 12;
      else if (y < r.bottom && y > r.bottom - 36) scroller.scrollTop += 12;
    }
    // A connection dropped on a terminal opens in it (split or new tab), instead of moving between folders.
    const pane = this.drag.kind === "profile" ? document.elementFromPoint(x, y)?.closest<HTMLElement>("[data-pane]") : null;
    if (pane) {
      this.paneDrop = { id: pane.dataset.pane ?? "", edge: paneEdge(pane.getBoundingClientRect(), x, y) };
      this.dropTarget = null;
      return;
    }
    this.paneDrop = null;
    const el = document.elementFromPoint(x, y)?.closest<HTMLElement>("[data-drop]");
    if (!el) {
      this.dropTarget = null;
      return;
    }
    const target = el.dataset.drop ?? "";
    const verdict = canDrop(this.drag.kind, this.drag.id, target, this.data.profiles, this.data.folders);
    this.dropTarget = target;
    this.dropOk = verdict.ok;
    this.dropReason = verdict.reason ?? "";
  }
  endDrag() {
    const d = this.drag;
    const target = this.dropTarget;
    const ok = this.dropOk;
    const pane = this.paneDrop;
    this.cancelDrag();
    this.justDragged = true;
    setTimeout(() => (this.justDragged = false), 0);
    if (d && pane) return this.onPaneDrop(d.id, pane.id, pane.edge);
    if (!d || target === null || !ok) return;
    const where = target || "the top level";
    const from = d.kind === "profile" ? this.data.profiles.find((p) => p.name === d.id)?.folder : folderParent(d.id);
    this.pending = {
      kind: d.kind,
      id: d.id,
      to: target,
      text: `Move ${d.label} into ${where}?`,
      sub: from ? `From ${from}. ${d.kind === "profile" ? "Its settings stay the same." : "Everything inside moves with it."}` : d.kind === "profile" ? "Its settings stay the same." : "Everything inside moves with it.",
    };
  }
  cancelDrag() {
    this.drag = null;
    this.dropTarget = null;
    this.paneDrop = null;
    document.body.classList.remove("dragging");
  }
  async confirmPending() {
    const p = this.pending;
    this.pending = null;
    if (!p) return;
    await this.run(async () => {
      if (p.kind === "profile") await ProfilesService.MoveProfile(p.id, p.to);
      else {
        const to = joinFolder(p.to, folderBase(p.id));
        await ProfilesService.RenameFolder(p.id, to);
        if (this.selectedFolder === p.id) this.selectedFolder = to;
      }
      this.expand(p.to);
    });
  }
}

export const sb = new SidebarState();

/** Makes an element draggable by pointer (native drag and drop is avoided so it behaves the same everywhere). */
export function dragSource(node: HTMLElement, opts: { kind: DragKind; id: string; label: string }) {
  let o = opts;
  let start: { x: number; y: number } | null = null;
  let active = false;

  const move = (e: PointerEvent) => {
    if (!start) return;
    if (!active) {
      if (Math.hypot(e.clientX - start.x, e.clientY - start.y) < 6) return;
      active = true;
      sb.beginDrag(o.kind, o.id, o.label);
    }
    sb.moveDrag(e.clientX, e.clientY);
  };
  const stop = () => {
    start = null;
    window.removeEventListener("pointermove", move);
    window.removeEventListener("pointerup", up);
    window.removeEventListener("keydown", key);
  };
  const up = () => {
    const was = active;
    active = false;
    stop();
    if (was) sb.endDrag();
  };
  const key = (e: KeyboardEvent) => {
    if (e.key === "Escape" && active) {
      active = false;
      stop();
      sb.cancelDrag();
    }
  };
  const down = (e: PointerEvent) => {
    // The row's own buttons are the drag handle (a mouse presses them, not the row); controls opt out.
    if (e.button !== 0 || (e.target as HTMLElement).closest("[data-nodrag], input, .pop")) return;
    start = { x: e.clientX, y: e.clientY };
    active = false;
    window.addEventListener("pointermove", move);
    window.addEventListener("pointerup", up);
    window.addEventListener("keydown", key);
  };
  node.addEventListener("pointerdown", down);
  return {
    update(n: typeof opts) { o = n; },
    destroy() { node.removeEventListener("pointerdown", down); stop(); },
  };
}
