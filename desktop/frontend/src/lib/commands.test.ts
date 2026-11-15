import { describe, expect, it } from "vitest";
import { COMMANDS, availableCommands, commandHint, findCommand, type CommandContext } from "./commands";

const baseCtx: CommandContext = {
  hasPane: false,
  hasTarget: false,
  isSsh: false,
  isRdp: false,
  ended: false,
  blocks: false,
  sidebarOpen: true,
};

const ids = (ctx: CommandContext) => new Set(availableCommands(ctx, true).map((c) => c.id));

describe("commands registry", () => {
  it("looks up commands by id and selects platform shortcut hints", () => {
    const quick = COMMANDS.find((c) => c.id === "new-tab")!;
    expect(findCommand("new-tab")?.label).toBe("New local tab");
    expect(commandHint(quick, true)).toBe("⌘T");
    expect(commandHint(quick, false)).toBe("Ctrl+Shift+T");
    expect(findCommand("nonexistent")).toBeUndefined();
  });

  it("hides pane and SSH-only actions when on the launcher canvas", () => {
    const visible = ids(baseCtx);
    expect(visible.has("new-connection")).toBe(true);
    expect(visible.has("settings")).toBe(true);
    expect(visible.has("theme-obsidian")).toBe(true);
    expect(visible.has("split-right")).toBe(false);
    expect(visible.has("files")).toBe(false);
    expect(visible.has("forwards")).toBe(false);
    expect(visible.has("reconnect")).toBe(false);
  });

  it("exposes SSH tools on an active SSH pane without blocks", () => {
    const visible = ids({
      ...baseCtx,
      hasPane: true,
      hasTarget: true,
      isSsh: true,
    });
    expect(visible.has("files")).toBe(true);
    expect(visible.has("forwards")).toBe(true);
    expect(visible.has("edit-connection")).toBe(true);
    expect(visible.has("find")).toBe(true);
    expect(visible.has("enable-blocks")).toBe(true);
    expect(visible.has("reconnect")).toBe(false);
  });

  it("hides SSH file/forward/search actions on RDP panes and exposes reconnect when ended", () => {
    const rdpVisible = ids({
      ...baseCtx,
      hasPane: true,
      hasTarget: true,
      isRdp: true,
      ended: true,
    });
    expect(rdpVisible.has("edit-connection")).toBe(true);
    expect(rdpVisible.has("reconnect")).toBe(true);
    expect(rdpVisible.has("files")).toBe(false);
    expect(rdpVisible.has("forwards")).toBe(false);
    expect(rdpVisible.has("find")).toBe(false);
    expect(rdpVisible.has("enable-blocks")).toBe(false);
  });
});
