import { describe, expect, it } from "vitest";
import { blankProfile, blocksWanted, snippetPreview, snippetText, connectingHint, loginQuiet, formatElapsed, groupByFolder, matches, paneEdge, parseTags, reorder, slotAt } from "./lib";

const mk = (o: object) => ({ ...blankProfile(), host: "h", user: "u", ...o }) as ReturnType<typeof blankProfile>;

describe("matches", () => {
  const p = mk({ name: "web-1", host: "10.0.0.5", folder: "prod/eu", tags: ["db", "Critical"] });
  it("matches any field, all terms required", () => {
    expect(matches(p, "web")).toBe(true);
    expect(matches(p, "10.0 prod")).toBe(true);
    expect(matches(p, "web staging")).toBe(false);
  });
  it("#tag matches tags by prefix, case-insensitively, and never other fields", () => {
    expect(matches(p, "#critical")).toBe(true);
    expect(matches(p, "#crit")).toBe(true);
    expect(matches(p, "#web")).toBe(false);
  });
  it("empty query matches everything", () => expect(matches(p, "  ")).toBe(true));
});

describe("groupByFolder", () => {
  it("sorts folders, puts unfiled last, sorts profiles by name", () => {
    const g = groupByFolder([mk({ name: "z" }), mk({ name: "b", folder: "b" }), mk({ name: "a", folder: "a" }), mk({ name: "y", folder: "a" })]);
    expect(g.map((n) => n.path)).toEqual(["a", "b", ""]);
    expect(g[0].profiles.map((p) => p.name)).toEqual(["a", "y"]);
  });
});

describe("parseTags", () => {
  it("trims, drops empties and duplicates", () => expect(parseTags(" a, b ,,a")).toEqual(["a", "b"]));
});

import { buildTree, canDrop, folderBase, folderParent, joinFolder } from "./lib";

const P = (name: string, folder = "") => mk({ name, folder });

describe("folder paths", () => {
  it("splits and joins", () => {
    expect(folderParent("a/b/c")).toBe("a/b");
    expect(folderParent("a")).toBe("");
    expect(folderBase("a/b/c")).toBe("c");
    expect(joinFolder("", "x")).toBe("x");
    expect(joinFolder("a/b", "x")).toBe("a/b/x");
  });
});

describe("buildTree", () => {
  const profiles = [P("web", "prod/eu"), P("bastion", "prod"), P("db", "staging"), P("solo")];
  const folders = ["archive", "prod", "prod/eu", "staging"]; // includes the empty "archive"

  it("nests folders, keeps empty ones, counts the whole subtree, and puts unfiled connections aside", () => {
    const t = buildTree(profiles, folders, "");
    expect(t.roots.map((n) => n.path)).toEqual(["archive", "prod", "staging"]);
    const prod = t.roots.find((n) => n.path === "prod")!;
    expect(prod.children.map((n) => n.path)).toEqual(["prod/eu"]);
    expect(prod.profiles.map((p) => p.name)).toEqual(["bastion"]);
    expect(prod.count).toBe(2); // bastion + web in the subfolder
    expect(t.roots.find((n) => n.path === "archive")!.count).toBe(0);
    expect(t.unfiled.map((p) => p.name)).toEqual(["solo"]);
  });

  it("a search keeps only folders that lead to a match and hides empty ones", () => {
    const t = buildTree(profiles, folders, "web");
    expect(t.roots.map((n) => n.path)).toEqual(["prod"]);
    expect(t.roots[0].children.map((n) => n.path)).toEqual(["prod/eu"]);
    expect(t.roots[0].profiles).toEqual([]);
    expect(t.unfiled).toEqual([]);
  });
});

describe("canDrop", () => {
  const profiles = [P("web", "prod/eu"), P("db", "staging")];
  const folders = ["prod", "prod/eu", "staging"];

  it("moving a connection somewhere new is fine, to where it already is is not", () => {
    expect(canDrop("profile", "web", "staging", profiles, folders).ok).toBe(true);
    expect(canDrop("profile", "web", "", profiles, folders).ok).toBe(true);
    expect(canDrop("profile", "web", "prod/eu", profiles, folders).ok).toBe(false);
  });

  it("a folder cannot go into itself, its own subfolder, its current parent, or onto a name that exists", () => {
    expect(canDrop("folder", "prod", "prod", profiles, folders).ok).toBe(false);
    expect(canDrop("folder", "prod", "prod/eu", profiles, folders).ok).toBe(false);
    expect(canDrop("folder", "prod/eu", "prod", profiles, folders).ok).toBe(false); // already there
    const clash = canDrop("folder", "prod/eu", "staging", profiles, [...folders, "staging/eu"]);
    expect(clash.ok).toBe(false);
    expect(clash.reason).toMatch(/already/);
    expect(canDrop("folder", "prod/eu", "staging", profiles, folders).ok).toBe(true);
    expect(canDrop("folder", "prod/eu", "", profiles, folders).ok).toBe(true); // to the top level
  });
});

describe("connecting progress", () => {
  it("formats elapsed seconds", () => {
    expect(formatElapsed(0)).toBe("0s");
    expect(formatElapsed(4_900)).toBe("4s");
    expect(formatElapsed(59_999)).toBe("59s");
    expect(formatElapsed(61_000)).toBe("1m 1s");
  });

  it("stays quiet at first, then says the host is slow, then that it may be unreachable", () => {
    expect(connectingHint(2_000, false)).toBe("");
    expect(connectingHint(9_000, false)).toMatch(/longer than usual/i);
    expect(connectingHint(31_000, false)).toMatch(/unreachable|offline|firewall/i);
  });

  it("mentions the jump host when the route has one", () => {
    expect(connectingHint(9_000, true)).toMatch(/jump host/i);
    expect(connectingHint(2_000, true)).toBe("");
  });
});

describe("tab reordering", () => {
  it("moves an item to a slot (the gap before index n, so n is the end)", () => {
    expect(reorder(["a", "b", "c", "d"], 0, 3)).toEqual(["b", "c", "a", "d"]);
    expect(reorder(["a", "b", "c", "d"], 3, 1)).toEqual(["a", "d", "b", "c"]);
    expect(reorder(["a", "b", "c"], 0, 3)).toEqual(["b", "c", "a"]);
    expect(reorder(["a", "b", "c"], 2, 0)).toEqual(["c", "a", "b"]);
  });

  it("leaves the order alone when dropped where it already is", () => {
    expect(reorder(["a", "b", "c"], 1, 1)).toEqual(["a", "b", "c"]);
    expect(reorder(["a", "b", "c"], 1, 2)).toEqual(["a", "b", "c"]);
  });

  it("finds the slot under the pointer from the tab centres", () => {
    const centres = [70, 210, 350];
    expect(slotAt(centres, 10)).toBe(0);
    expect(slotAt(centres, 100)).toBe(1);
    expect(slotAt(centres, 300)).toBe(2);
    expect(slotAt(centres, 900)).toBe(3);
    expect(slotAt([], 50)).toBe(0);
  });
});

describe("pane drop zones", () => {
  const r = { left: 100, top: 50, width: 400, height: 300 };
  it("splits toward the nearest edge", () => {
    expect(paneEdge(r, 110, 200)).toBe("left");
    expect(paneEdge(r, 490, 200)).toBe("right");
    expect(paneEdge(r, 300, 55)).toBe("top");
    expect(paneEdge(r, 300, 345)).toBe("bottom");
  });
  it("opens a new tab in the middle", () => {
    expect(paneEdge(r, 300, 200)).toBe("center");
  });
  it("prefers the closer edge in a corner", () => {
    expect(paneEdge(r, 105, 60)).toBe("left"); // 5/400 from the left, 10/300 from the top
    expect(paneEdge(r, 130, 52)).toBe("top");
  });
});

describe("automatic command blocks", () => {
  it("a connection's own mode beats the app setting", () => {
    expect(blocksWanted("on", false)).toBe(true);
    expect(blocksWanted("off", true)).toBe(false);
    expect(blocksWanted("", true)).toBe(true);
    expect(blocksWanted("", false)).toBe(false);
    expect(blocksWanted(undefined, true)).toBe(true);
  });

  it("waits for the login to finish and the output to go quiet before typing", () => {
    // not authenticated yet: never, however quiet (a prompt may be waiting for a password)
    expect(loginQuiet(false, 0, 0, 10_000)).toBe(false);
    // just authenticated, or output still arriving
    expect(loginQuiet(true, 1_000, 1_000, 1_200)).toBe(false);
    expect(loginQuiet(true, 1_000, 1_900, 2_300)).toBe(false);
    // quiet long enough since the later of the marker and the last output
    expect(loginQuiet(true, 1_000, 1_000, 1_800)).toBe(true);
    expect(loginQuiet(true, 1_000, 1_900, 2_700)).toBe(true);
  });
});

describe("snippets", () => {
  it("types a command as written, with the line endings a terminal expects", () => {
    expect(snippetText("ls -la")).toBe("ls -la");
    expect(snippetText("df -h\r\ndu -sh *\r\n\r\n")).toBe("df -h\ndu -sh *");
    expect(snippetText("  echo hi  \n")).toBe("  echo hi");
  });

  it("previews the first line and says how many more there are", () => {
    expect(snippetPreview("ls")).toBe("ls");
    expect(snippetPreview("df -h\ndu -sh *\nls")).toBe("df -h  (+2 lines)");
    expect(snippetPreview("a\nb")).toBe("a  (+1 line)");
    expect(snippetPreview("x".repeat(200)).length).toBeLessThanOrEqual(90);
  });
});

describe("pane drop zones without a middle", () => {
  it("always picks an edge, for a layout that has no 'swap' or 'new tab' to offer", () => {
    const r = { left: 0, top: 0, width: 400, height: 300 };
    expect(paneEdge(r, 200, 150, false)).toMatch(/^(left|right|top|bottom)$/);
    expect(paneEdge(r, 190, 100, false)).toBe("top");
    expect(paneEdge(r, 100, 150, false)).toBe("left");
  });
});
