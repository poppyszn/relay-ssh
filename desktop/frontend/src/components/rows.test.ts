// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { describe, expect, it } from "vitest";
import Branch from "./Branch.svelte";
import ConnRow from "./ConnRow.svelte";
import { sb } from "../state/foldertree.svelte";
import { blankProfile } from "../lib/lib";

// The sidebar's "Open now" list renders rows at depth -1 (flush left, no indent guides).
// Array(-1) throws, which broke the whole sidebar the moment a connection was opened.
describe("ConnRow", () => {
  const props = (depth: number) => ({ profile: { ...blankProfile(), name: "web", host: "h", user: "u" }, depth, live: true, current: false, onconnect: () => {}, onedit: () => {} });

  it.each([-1, 0, 2])("renders at depth %i with one guide per level", (depth) => {
    const target = document.createElement("div");
    const c = mount(ConnRow, { target, props: props(depth) });
    expect(target.querySelectorAll(".guide")).toHaveLength(Math.max(0, depth));
    expect(target.textContent).toContain("web");
    unmount(c);
  });

  it("offers Duplicate next to Edit, and does not connect when it is used", () => {
    const target = document.createElement("div");
    let duplicated = "", connected = 0;
    const profile = { ...blankProfile(), name: "web", host: "h", user: "u" };
    const c = mount(ConnRow, { target, props: { ...props(0), profile, onconnect: () => connected++, onduplicate: (p: typeof profile) => (duplicated = p.name) } });
    const btn = target.querySelector<HTMLButtonElement>('button[aria-label="Duplicate web"]')!;
    expect(btn).not.toBeNull();
    btn.click();
    expect(duplicated).toBe("web");
    expect(connected).toBe(0);
    unmount(c);
  });

  // The name line used to carry tags, the jump host and the edit button, so names truncated to a letter.
  it("keeps the name line to the name and one tag, and puts the jump host with the address", () => {
    const target = document.createElement("div");
    const profile = { ...blankProfile(), name: "dev-app-server", host: "10.0.2.15", user: "deploy", port: 2222, jump: "bastion", tags: ["dev", "eu", "web"] };
    const c = mount(ConnRow, { target, props: { ...props(0), profile } });
    expect([...target.querySelectorAll(".tags .chip")].map((e) => e.textContent)).toEqual(["dev"]);
    expect(target.querySelector(".more")?.textContent).toBe("+2");
    expect(target.querySelector(".l2")?.textContent).toBe("deploy@10.0.2.15:2222via bastion");
    unmount(c);
  });
});

// (MouseEvent stands in for PointerEvent, whose button and coordinates happy-dom ignores.)
// A real mouse presses the row's inner button, not the row element, so dragging has to start from there.
describe("dragging from the buttons a mouse actually hits", () => {
  const press = (el: Element) => el.dispatchEvent(new MouseEvent("pointerdown", { bubbles: true, button: 0, clientX: 50, clientY: 20 }));
  const pull = () => window.dispatchEvent(new MouseEvent("pointermove", { clientX: 90, clientY: 60 }));
  const release = () => window.dispatchEvent(new MouseEvent("pointerup", { clientX: 90, clientY: 60 }));

  it("starts from a connection row's name button, not from its edit button", () => {
    const target = document.createElement("div");
    const c = mount(ConnRow, { target, props: { profile: { ...blankProfile(), name: "web", host: "h", user: "u" }, depth: 0, live: false, current: false, onconnect: () => {}, onedit: () => {} } });
    flushSync(); // actions attach in an effect
    press(target.querySelector(".edit")!); pull();
    expect(sb.drag).toBeNull();
    release();
    press(target.querySelector(".main")!); pull();
    expect(sb.drag).toMatchObject({ kind: "profile", id: "web" });
    release();
    sb.cancelDrag();
    unmount(c);
  });

  it("starts from a folder's name button, not from its caret or menu button", () => {
    const target = document.createElement("div");
    const node = { name: "prod", path: "prod", count: 0, children: [], profiles: [] } as never;
    const c = mount(Branch, { target, props: { node, depth: 0, searching: false, open: new Set<string>(), current: "", onconnect: () => {}, onedit: () => {} } });
    flushSync();
    for (const sel of [".caret", ".more"]) { press(target.querySelector(sel)!); pull(); expect(sb.drag).toBeNull(); release(); }
    press(target.querySelector(".name")!); pull();
    expect(sb.drag).toMatchObject({ kind: "folder", id: "prod" });
    release();
    sb.cancelDrag();
    unmount(c);
  });
});

describe("opening a connection", () => {
  const mk = () => {
    const opened: string[] = [];
    const target = document.createElement("div");
    const c = mount(ConnRow, { target, props: { profile: { ...blankProfile(), name: "web", host: "h", user: "u" }, depth: 0, live: false, current: false, onconnect: (p: { name: string }) => opened.push(p.name), onedit: () => {} } });
    flushSync();
    sb.justDragged = false; // an earlier drag test may have left it set
    return { opened, target, c, main: target.querySelector<HTMLElement>(".main")! };
  };

  // happy-dom ignores `detail` in the MouseEvent init, so it is set on the event.
  const click = (el: Element, detail: number) => {
    const e = new MouseEvent("click", { bubbles: true });
    Object.defineProperty(e, "detail", { value: detail });
    el.dispatchEvent(e);
  };

  it("a single click selects the row and asks to show its open session, without opening anything", () => {
    const { opened, target, c, main } = mk();
    const shown: string[] = [];
    sb.onFocusConnection = (n) => shown.push(n);
    click(main, 1);
    flushSync();
    expect(shown).toEqual(["web"]);
    expect(opened).toEqual([]);
    expect(sb.selectedConn).toBe("web");
    expect(target.querySelector(".row.selected")).not.toBeNull();
    unmount(c);
  });

  it("a double-click opens it, and remembers that the user knows", () => {
    const { opened, c, main } = mk();
    main.dispatchEvent(new MouseEvent("dblclick", { bubbles: true, detail: 2 }));
    expect(opened).toEqual(["web"]);
    expect(sb.learnedOpen).toBe(true);
    unmount(c);
  });

  it("Enter on the focused row opens it (a keyboard click has detail 0)", () => {
    const { opened, c, main } = mk();
    click(main, 0);
    expect(opened).toEqual(["web"]);
    unmount(c);
  });
});

describe("Duplicate in the sidebar tree", () => {
  it("is on connections inside folders too", () => {
    const target = document.createElement("div");
    const web = { ...blankProfile(), name: "web", host: "h", user: "u", folder: "Acme" };
    const node = { name: "Acme", path: "Acme", children: [], profiles: [web], count: 1 };
    const c = mount(Branch, { target, props: { node, depth: 0, searching: false, open: new Set<string>(), current: "", onconnect: () => {}, onedit: () => {}, onduplicate: () => {} } as never });
    flushSync();
    expect(target.querySelector('button[aria-label="Duplicate web"]')).not.toBeNull();
    unmount(c);
  });
});
