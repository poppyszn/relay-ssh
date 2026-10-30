<script lang="ts" module>
  // One icon set: 24-grid outline glyphs, 1.75 stroke, drawn the same way everywhere.
  const PATHS = {
    check: "M5 12l5 5 9-10",
    x: "M6 6l12 12M18 6L6 18",
    plus: "M12 5v14M5 12h14",
    search: "M11 4a7 7 0 1 0 0 14 7 7 0 0 0 0-14zM20 20l-4-4",
    "chevron-right": "M9 6l6 6-6 6",
    "chevron-down": "M6 9l6 6 6-6",
    "chevron-up": "M6 15l6-6 6 6",
    "arrow-right": "M5 12h12M13 8l4 4-4 4",
    "arrow-up": "M12 19V5M6 11l6-6 6 6",
    refresh: "M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7",
    more: "M5 12h.01M12 12h.01M19 12h.01",
    grip: "M9 6h.01M15 6h.01M9 12h.01M15 12h.01M9 18h.01M15 18h.01",
    folder: "M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z",
    "folder-open": "M3 8a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v1H7l-3 8H5a2 2 0 0 1-2-2zM7 11h15l-3 8H4z",
    "folder-plus": "M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2zM12 10.5v5M9.5 13h5",
    terminal: "M4 17l6-6-6-6M12 19h8",
    pencil: "M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z",
    trash: "M4 7h16M10 11v6M14 11v6M6 7l1 13h10l1-13M9 7V4h6v3",
    copy: "M9 9h10v10H9zM5 15V5h10",
    lines: "M4 6h16M4 12h16M4 18h10",
    "copy-all": "M5 5h14v14H5zM8 9h8M8 13h8M8 17h5",
    import: "M12 3v12M7 10l5 5 5-5M4 20h16",
    key: "M8 11a4 4 0 1 0 0 8 4 4 0 0 0 0-8zM11 12l9-9M16 7l3 3",
    lock: "M5 11h14v10H5zM8 11V7a4 4 0 0 1 8 0v4",
    unlock: "M5 11h14v10H5zM8 11V7a4 4 0 0 1 7.9-1",
    sidebar: "M4 5h16v14H4zM9 5v14",
    "split-right": "M4 5h16v14H4zM12 5v14",
    "split-down": "M4 5h16v14H4zM4 12h16",
    forwards: "M4 8h13M13 4l4 4-4 4M20 16H7M11 12l-4 4 4 4",
    snippets: "M8 4H5v16h3M16 4h3v16h-3M9 9l2 2-2 2M13 13h3",
    tools: "M14.7 6.3a4 4 0 0 0-5.4 5.2L3 17.8V21h3.2l6.3-6.3a4 4 0 0 0 5.2-5.4l-2.6 2.6-2.4-.6-.6-2.4z",
    settings: "M4 7h10M18 7h2M4 17h2M10 17h10M14 4v6M6 14v6",
    files: "M6 3h8l4 4v14H6zM14 3v4h4M9 13h6M9 17h6",
    warning: "M12 4l9 16H3zM12 10v4M12 17.5v.01",
    info: "M12 3a9 9 0 1 0 0 18 9 9 0 0 0 0-18zM12 11v5M12 8v.01",
    vault: "M4 6h16v13H4zM8 6V4h8v2M12 11v3",
    rdp: "M4 5h16v11H4zM8 20h8M12 16v4",
  } as const;
  export type IconName = keyof typeof PATHS;
</script>

<script lang="ts">
  let { name, size = 16 }: { name: IconName; size?: 14 | 16 | 20 } = $props();
</script>

<svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d={PATHS[name]} /></svg>
