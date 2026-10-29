import { describe, expect, it } from "vitest";
import { blockText, rowAt, type BufLike } from "./blocktext";

// A fake buffer: each entry is a row's text and whether it continues the row above (a wrapped line).
function buf(rows: Array<string | [string, boolean]>): BufLike {
  return {
    length: rows.length,
    getLine: (y) => {
      const r = rows[y];
      if (r === undefined) return undefined;
      const [text, wrapped] = typeof r === "string" ? [r, false] : r;
      return { isWrapped: wrapped, translateToString: (_trimRight = false, start = 0) => text.slice(start) };
    },
  };
}

describe("blockText", () => {
  // rows: 0 blank padding, 1 path header, 2 prompt + command, 3.. output, then padding
  const rows = ["", "~/app git:(main)", "❯ ls -la", "total 8", "drwxr-xr-x  5 me  staff  160 .", "", ""];

  it("splits the command (after the prompt) from the output (trailing blank rows dropped)", () => {
    const t = blockText(buf(rows), { cmdLine: 2, cmdCol: 2, endLine: rows.length });
    expect(t.command).toBe("ls -la");
    expect(t.output).toBe("total 8\ndrwxr-xr-x  5 me  staff  160 .");
  });

  it("joins a long command that wrapped across rows, and starts the output after it", () => {
    const wrapped = ["", "~/app", ["❯ echo aaaaaaaa", false], ["bbbbbbbb done", true], "aaaaaaaabbbbbbbb done", ""];
    const t = blockText(buf(wrapped as never), { cmdLine: 2, cmdCol: 2, endLine: wrapped.length });
    expect(t.command).toBe("echo aaaaaaaabbbbbbbb done");
    expect(t.output).toBe("aaaaaaaabbbbbbbb done");
  });

  it("keeps blank rows inside the output, and joins wrapped output rows without a newline", () => {
    const r = ["", "❯ cat f", "one", "", "two-", ["continued", true], ""];
    const t = blockText(buf(r as never), { cmdLine: 1, cmdCol: 2, endLine: r.length });
    expect(t.output).toBe("one\n\ntwo-continued");
  });

  it("ignores rows of explicit spaces, like the one zsh draws after output with no trailing newline", () => {
    const r = ["", "❯ printf hi", "hi", " ".repeat(80), ""];
    const t = blockText(buf(r), { cmdLine: 1, cmdCol: 2, endLine: r.length });
    expect(t.output).toBe("hi");
  });

  it("a command with no output has an empty output", () => {
    const r = ["", "❯ true", ""];
    expect(blockText(buf(r), { cmdLine: 1, cmdCol: 2, endLine: r.length })).toEqual({ command: "true", output: "" });
  });

  it("copes with rows that no longer exist (scrolled out of the buffer)", () => {
    expect(blockText(buf([]), { cmdLine: 5, cmdCol: 0, endLine: 9 })).toEqual({ command: "", output: "" });
  });
});

describe("rowAt", () => {
  it("maps a pointer position to a buffer row, accounting for the scroll offset", () => {
    expect(rowAt(100, 40, 20, 0)).toBe(3); // (100-40)/20 = row 3
    expect(rowAt(100, 40, 20, 50)).toBe(53); // scrolled by 50 rows
    expect(rowAt(30, 40, 20, 0)).toBe(-1); // above the screen
  });
});
