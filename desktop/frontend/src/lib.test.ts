import { describe, expect, it } from "vitest";
import { blankProfile, groupByFolder, matches, parseTags } from "./lib";

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
