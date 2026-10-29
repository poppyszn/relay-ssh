import { describe, expect, it } from "vitest";
import { insertBeside, leaves, movePane, neighbor, removeLeaf, setRatio, splitLeaf, swapLeaves, type Node } from "./panes";

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

describe("splitLeaf before", () => {
  it("puts the new pane first when asked", () => {
    const t = splitLeaf({ id: "a" }, "a", "b", "row", "s1", true);
    expect(leaves(t)).toEqual(["b", "a"]);
    expect(leaves(splitLeaf({ id: "a" }, "a", "b", "row", "s1"))).toEqual(["a", "b"]);
  });
});

describe("rearranging panes", () => {
  // a | (b / c)
  const tree: Node = { sid: "s1", dir: "row", ratio: 0.5, a: { id: "a" }, b: { sid: "s2", dir: "col", ratio: 0.5, a: { id: "b" }, b: { id: "c" } } };

  it("moves a pane beside another, on the chosen side", () => {
    expect(leaves(movePane(tree, "a", "c", "right", () => "n")!)).toEqual(["b", "c", "a"]);
    expect(leaves(movePane(tree, "a", "b", "top", () => "n")!)).toEqual(["a", "b", "c"]);
    expect(leaves(movePane(tree, "c", "a", "left", () => "n")!)).toEqual(["c", "a", "b"]);
  });

  it("splits the target the right way round", () => {
    const t = movePane({ sid: "s", dir: "row", ratio: 0.5, a: { id: "a" }, b: { id: "b" } }, "a", "b", "bottom", () => "n")!;
    expect(t).toEqual({ sid: "n", dir: "col", ratio: 0.5, a: { id: "b" }, b: { id: "a" } });
  });

  it("keeps every pane exactly once", () => {
    for (const edge of ["left", "right", "top", "bottom"] as const) {
      expect(leaves(movePane(tree, "b", "a", edge, () => "n")!).sort()).toEqual(["a", "b", "c"]);
    }
  });

  it("does nothing for a pane dropped on itself, an unknown pane, or a lone pane", () => {
    expect(movePane(tree, "a", "a", "left", () => "n")).toBeNull();
    expect(movePane(tree, "a", "zzz", "left", () => "n")).toBeNull();
    expect(movePane({ id: "a" }, "a", "b", "left", () => "n")).toBeNull();
  });

  it("swaps two panes in place", () => {
    expect(leaves(swapLeaves(tree, "a", "c"))).toEqual(["c", "b", "a"]);
  });
});

describe("placing a whole layout beside a pane", () => {
  const dest: Node = { sid: "s1", dir: "row", ratio: 0.5, a: { id: "a" }, b: { id: "b" } };
  const incoming: Node = { sid: "x", dir: "col", ratio: 0.4, a: { id: "p" }, b: { id: "q" } };

  it("puts the layout on the chosen side of the pane, keeping its own splits", () => {
    const right = insertBeside(dest, "b", incoming, "right", () => "n");
    expect(leaves(right)).toEqual(["a", "b", "p", "q"]);
    expect(leaves(insertBeside(dest, "a", incoming, "left", () => "n"))).toEqual(["p", "q", "a", "b"]);
    const bottom = insertBeside({ id: "a" }, "a", incoming, "bottom", () => "n");
    expect(bottom).toEqual({ sid: "n", dir: "col", ratio: 0.5, a: { id: "a" }, b: incoming });
  });

  it("works for a single pane too", () => {
    expect(leaves(insertBeside(dest, "a", { id: "z" }, "top", () => "n"))).toEqual(["z", "a", "b"]);
  });

  it("returns the layout unchanged when the target is not in it", () => {
    expect(insertBeside(dest, "nope", incoming, "right", () => "n")).toBe(dest);
  });
});
