import { describe, expect, it } from "vitest";
import { leaves, neighbor, removeLeaf, setRatio, splitLeaf, type Node } from "./panes";

const one: Node = { id: "a" };

describe("pane tree", () => {
  it("splits a leaf into a row or column with the new pane second", () => {
    const t = splitLeaf(one, "a", "b", "row", "s1");
    expect(t).toEqual({ sid: "s1", dir: "row", ratio: 0.5, a: { id: "a" }, b: { id: "b" } });
    expect(leaves(t)).toEqual(["a", "b"]);
  });

  it("splits a nested leaf and keeps order", () => {
    const t = splitLeaf(splitLeaf(one, "a", "b", "row", "s1"), "b", "c", "col", "s2");
    expect(leaves(t)).toEqual(["a", "b", "c"]);
  });

  it("removing a pane collapses its parent into the sibling", () => {
    const t = splitLeaf(splitLeaf(one, "a", "b", "row", "s1"), "b", "c", "col", "s2");
    expect(leaves(removeLeaf(t, "b")!)).toEqual(["a", "c"]);
    expect(removeLeaf(removeLeaf(t, "b")!, "c")).toEqual({ id: "a" });
  });

  it("removing the only pane yields null", () => expect(removeLeaf(one, "a")).toBeNull());

  it("neighbor picks the closest remaining pane to focus", () => {
    const t = splitLeaf(splitLeaf(one, "a", "b", "row", "s1"), "b", "c", "col", "s2");
    expect(neighbor(t, "c")).toBe("b");
    expect(neighbor(t, "a")).toBe("b");
    expect(neighbor(one, "a")).toBeUndefined();
  });

  it("setRatio clamps so no pane can be dragged shut", () => {
    const t = splitLeaf(one, "a", "b", "row", "s1");
    expect((setRatio(t, "s1", 0.99) as { ratio: number }).ratio).toBe(0.85);
    expect((setRatio(t, "s1", 0.01) as { ratio: number }).ratio).toBe(0.15);
    expect((setRatio(t, "s1", 0.4) as { ratio: number }).ratio).toBe(0.4);
  });
});
