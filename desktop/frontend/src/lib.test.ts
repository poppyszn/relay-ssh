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
