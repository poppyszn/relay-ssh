import type { IconName } from "../components/Icon.svelte";

export type CommandGroup = "Connection" | "Workspace" | "Session" | "Appearance" | "Settings";

export interface CommandContext {
  hasPane: boolean;
  hasTarget: boolean;
  isSsh: boolean;
  isRdp: boolean;
  ended: boolean;
  blocks: boolean;
  sidebarOpen: boolean;
}

export interface CommandSpec {
  id: string;
  label: string;
  group: CommandGroup;
  icon?: IconName;
  macHint?: string;
  otherHint?: string;
  /** Optional platform toolbar/menu description. */
  summary?: string;
  when?: (ctx: CommandContext) => boolean;
}

export const COMMANDS: readonly CommandSpec[] = [
  { id: "new-connection", label: "New connection", group: "Connection", icon: "plus", macHint: "⌘N", otherHint: "Ctrl+Shift+N" },
  { id: "new-folder", label: "New folder", group: "Connection", icon: "folder-plus", macHint: "⇧⌘N", otherHint: "Ctrl+Alt+N" },
  { id: "edit-connection", label: "Edit current connection", group: "Connection", icon: "pencil", when: (c) => c.hasTarget },
  { id: "duplicate-connection", label: "Duplicate current connection", group: "Connection", icon: "copy", when: (c) => c.hasTarget },

  { id: "new-tab", label: "New local tab", group: "Workspace", icon: "terminal", macHint: "⌘T", otherHint: "Ctrl+Shift+T" },
  { id: "split-right", label: "Split pane right", group: "Workspace", icon: "split-right", macHint: "⌘D", otherHint: "Ctrl+Shift+D", when: (c) => c.hasPane },
  { id: "split-down", label: "Split pane down", group: "Workspace", icon: "split-down", macHint: "⇧⌘D", otherHint: "Ctrl+Alt+D", when: (c) => c.hasPane },
  { id: "toggle-sidebar", label: "Toggle connections sidebar", group: "Workspace", icon: "sidebar", macHint: "⌘B", otherHint: "Ctrl+Shift+B" },
  { id: "next-tab", label: "Next tab", group: "Workspace", icon: "chevron-right", macHint: "⇧⌘]", when: (c) => c.hasPane },
  { id: "prev-tab", label: "Previous tab", group: "Workspace", icon: "chevron-right", macHint: "⇧⌘[", when: (c) => c.hasPane },
  { id: "close", label: "Close focused pane", group: "Workspace", icon: "x", macHint: "⌘W", otherHint: "Ctrl+Shift+W", when: (c) => c.hasPane },
  { id: "close-tab", label: "Close current tab", group: "Workspace", icon: "x", macHint: "⇧⌘W", otherHint: "Ctrl+Alt+W", when: (c) => c.hasPane },

  { id: "files", label: "Browse files", group: "Session", icon: "files", macHint: "⇧⌘O", otherHint: "Ctrl+Shift+O", when: (c) => c.isSsh },
  { id: "forwards", label: "Port forwards", group: "Session", icon: "forwards", macHint: "⇧⌘P", otherHint: "Ctrl+Shift+P", when: (c) => c.isSsh },
  { id: "snippets", label: "Snippets", group: "Session", icon: "snippets", macHint: "⇧⌘S", otherHint: "Ctrl+Shift+S" },
  { id: "find", label: "Find in terminal scrollback", group: "Session", icon: "search", macHint: "⌘F", otherHint: "Ctrl+Shift+F", when: (c) => c.hasPane && !c.isRdp },
  { id: "enable-blocks", label: "Enable command blocks in this session", group: "Session", icon: "terminal", when: (c) => c.hasPane && c.isSsh && !c.blocks && !c.ended },
  { id: "reconnect", label: "Reconnect focused session", group: "Session", icon: "refresh", when: (c) => c.hasPane && c.ended },

  { id: "tools", label: "Tools: health, backup, import, keys", group: "Settings", icon: "tools" },
  { id: "settings", label: "Settings", group: "Settings", icon: "settings", macHint: "⌘,", otherHint: "Ctrl+Shift+," },
  { id: "vault", label: "Vault settings", group: "Settings", icon: "vault" },

  { id: "theme-obsidian", label: "Theme: Obsidian (Pure dark)", group: "Appearance", icon: "terminal" },
  { id: "theme-midnight", label: "Theme: Midnight", group: "Appearance", icon: "terminal" },
  { id: "theme-graphite", label: "Theme: Graphite", group: "Appearance", icon: "terminal" },
  { id: "theme-daylight", label: "Theme: Daylight", group: "Appearance", icon: "terminal" },
  { id: "theme-system", label: "Theme: Follow system appearance", group: "Appearance", icon: "terminal" },
];

export const commandHint = (spec: Pick<CommandSpec, "macHint" | "otherHint">, mac: boolean): string =>
  (mac ? spec.macHint : spec.otherHint) ?? "";

export const findCommand = (id: string): CommandSpec | undefined => COMMANDS.find((c) => c.id === id);

export function availableCommands(ctx: CommandContext, mac: boolean) {
  return COMMANDS.filter((c) => !c.when || c.when(ctx)).map((c) => ({
    id: c.id,
    label: c.label,
    group: c.group,
    icon: c.icon,
    hint: commandHint(c, mac),
  }));
}
