<script lang="ts" module>
  type SvgNode =
    | ["path", { d: string }]
    | ["rect", { width: string; height: string; x: string; y: string; rx: string; ry?: string }]
    | ["circle", { cx: string; cy: string; r: string; fill?: string }]
    | ["line", { x1: string; x2: string; y1: string; y2: string }];

  const ICONS = {
    check: [["path", { d: "M20 6 9 17l-5-5" }]],
    x: [["path", { d: "M18 6 6 18" }], ["path", { d: "m6 6 12 12" }]],
    plus: [["path", { d: "M5 12h14" }], ["path", { d: "M12 5v14" }]],
    search: [["path", { d: "m21 21-4.34-4.34" }], ["circle", { cx: "11", cy: "11", r: "8" }]],
    "chevron-right": [["path", { d: "m9 18 6-6-6-6" }]],
    "chevron-down": [["path", { d: "m6 9 6 6 6-6" }]],
    "chevron-up": [["path", { d: "m18 15-6-6-6 6" }]],
    "arrow-right": [["path", { d: "M5 12h14" }], ["path", { d: "m12 5 7 7-7 7" }]],
    "arrow-up": [["path", { d: "m5 12 7-7 7 7" }], ["path", { d: "M12 19V5" }]],
    refresh: [
      ["path", { d: "M3 12a9 9 0 0 1 9-9 9.75 9.75 0 0 1 6.74 2.74L21 8" }],
      ["path", { d: "M21 3v5h-5" }],
      ["path", { d: "M21 12a9 9 0 0 1-9 9 9.75 9.75 0 0 1-6.74-2.74L3 16" }],
      ["path", { d: "M8 16H3v5" }],
    ],
    more: [
      ["circle", { cx: "12", cy: "12", r: "1" }],
      ["circle", { cx: "19", cy: "12", r: "1" }],
      ["circle", { cx: "5", cy: "12", r: "1" }],
    ],
    grip: [
      ["circle", { cx: "9", cy: "12", r: "1" }],
      ["circle", { cx: "9", cy: "5", r: "1" }],
      ["circle", { cx: "9", cy: "19", r: "1" }],
      ["circle", { cx: "15", cy: "12", r: "1" }],
      ["circle", { cx: "15", cy: "5", r: "1" }],
      ["circle", { cx: "15", cy: "19", r: "1" }],
    ],
    folder: [["path", { d: "M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z" }]],
    "folder-open": [["path", { d: "m6 14 1.5-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.54 6a2 2 0 0 1-1.95 1.5H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3.9a2 2 0 0 1 1.69.9l.81 1.2a2 2 0 0 0 1.67.9H18a2 2 0 0 1 2 2v2" }]],
    "folder-plus": [
      ["path", { d: "M12 10v6" }],
      ["path", { d: "M9 13h6" }],
      ["path", { d: "M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z" }],
    ],
    terminal: [["path", { d: "M12 19h8" }], ["path", { d: "m4 17 6-6-6-6" }]],
    pencil: [
      ["path", { d: "M21.174 6.812a1 1 0 0 0-3.986-3.987L3.842 16.174a2 2 0 0 0-.5.83l-1.321 4.352a.5.5 0 0 0 .623.622l4.353-1.32a2 2 0 0 0 .83-.497z" }],
      ["path", { d: "m15 5 4 4" }],
    ],
    trash: [
      ["path", { d: "M10 11v6" }],
      ["path", { d: "M14 11v6" }],
      ["path", { d: "M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6" }],
      ["path", { d: "M3 6h18" }],
      ["path", { d: "M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2" }],
    ],
    copy: [
      ["rect", { width: "14", height: "14", x: "8", y: "8", rx: "2", ry: "2" }],
      ["path", { d: "M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" }],
    ],
    lines: [["path", { d: "M21 5H3" }], ["path", { d: "M15 12H3" }], ["path", { d: "M17 19H3" }]],
    "copy-all": [
      ["path", { d: "m12 15 2 2 4-4" }],
      ["rect", { width: "14", height: "14", x: "8", y: "8", rx: "2", ry: "2" }],
      ["path", { d: "M4 16c-1.1 0-2-.9-2-2V4c0-1.1.9-2 2-2h10c1.1 0 2 .9 2 2" }],
    ],
    import: [
      ["path", { d: "M12 15V3" }],
      ["path", { d: "M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" }],
      ["path", { d: "m7 10 5 5 5-5" }],
    ],
    key: [
      ["path", { d: "M2.586 17.414A2 2 0 0 0 2 18.828V21a1 1 0 0 0 1 1h3a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h1a1 1 0 0 0 1-1v-1a1 1 0 0 1 1-1h.172a2 2 0 0 0 1.414-.586l.814-.814a6.5 6.5 0 1 0-4-4z" }],
      ["circle", { cx: "16.5", cy: "7.5", r: ".5", fill: "currentColor" }],
    ],
    lock: [
      ["rect", { width: "18", height: "11", x: "3", y: "11", rx: "2", ry: "2" }],
      ["path", { d: "M7 11V7a5 5 0 0 1 10 0v4" }],
    ],
    unlock: [
      ["rect", { width: "18", height: "11", x: "3", y: "11", rx: "2", ry: "2" }],
      ["path", { d: "M7 11V7a5 5 0 0 1 9.9-1" }],
    ],
    sidebar: [
      ["rect", { width: "18", height: "18", x: "3", y: "3", rx: "2" }],
      ["path", { d: "M9 3v18" }],
    ],
    "split-right": [
      ["rect", { width: "18", height: "18", x: "3", y: "3", rx: "2" }],
      ["path", { d: "M12 3v18" }],
    ],
    "split-down": [
      ["rect", { width: "18", height: "18", x: "3", y: "3", rx: "2" }],
      ["path", { d: "M3 12h18" }],
    ],
    forwards: [
      ["path", { d: "M8 3 4 7l4 4" }],
      ["path", { d: "M4 7h16" }],
      ["path", { d: "m16 21 4-4-4-4" }],
      ["path", { d: "M20 17H4" }],
    ],
    snippets: [["path", { d: "m16 18 6-6-6-6" }], ["path", { d: "m8 6-6 6 6 6" }]],
    tools: [["path", { d: "M14.7 6.3a1 1 0 0 0 0 1.4l1.6 1.6a1 1 0 0 0 1.4 0l3.106-3.105c.32-.322.863-.22.983.218a6 6 0 0 1-8.259 7.057l-7.91 7.91a1 1 0 0 1-2.999-3l7.91-7.91a6 6 0 0 1 7.057-8.259c.438.12.54.662.219.984z" }]],
    settings: [
      ["path", { d: "M10 5H3" }],
      ["path", { d: "M12 19H3" }],
      ["path", { d: "M14 3v4" }],
      ["path", { d: "M16 17v4" }],
      ["path", { d: "M21 12h-9" }],
      ["path", { d: "M21 19h-5" }],
      ["path", { d: "M21 5h-7" }],
      ["path", { d: "M8 10v4" }],
      ["path", { d: "M8 12H3" }],
    ],
    files: [
      ["path", { d: "M6 22a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h8a2.4 2.4 0 0 1 1.704.706l3.588 3.588A2.4 2.4 0 0 1 20 8v12a2 2 0 0 1-2 2z" }],
      ["path", { d: "M14 2v5a1 1 0 0 0 1 1h5" }],
      ["path", { d: "M10 9H8" }],
      ["path", { d: "M16 13H8" }],
      ["path", { d: "M16 17H8" }],
    ],
    warning: [
      ["path", { d: "m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3" }],
      ["path", { d: "M12 9v4" }],
      ["path", { d: "M12 17h.01" }],
    ],
    info: [
      ["circle", { cx: "12", cy: "12", r: "10" }],
      ["path", { d: "M12 16v-4" }],
      ["path", { d: "M12 8h.01" }],
    ],
    vault: [["path", { d: "M20 13c0 5-3.5 7.5-7.66 8.95a1 1 0 0 1-.67-.01C7.5 20.5 4 18 4 13V6a1 1 0 0 1 1-1c2 0 4.5-1.2 6.24-2.72a1.17 1.17 0 0 1 1.52 0C14.51 3.81 17 5 19 5a1 1 0 0 1 1 1z" }]],
    rdp: [
      ["rect", { width: "20", height: "14", x: "2", y: "3", rx: "2" }],
      ["line", { x1: "8", x2: "16", y1: "21", y2: "21" }],
      ["line", { x1: "12", x2: "12", y1: "17", y2: "21" }],
    ],
    server: [
      ["rect", { width: "20", height: "8", x: "2", y: "2", rx: "2", ry: "2" }],
      ["rect", { width: "20", height: "8", x: "2", y: "14", rx: "2", ry: "2" }],
      ["line", { x1: "6", x2: "6.01", y1: "6", y2: "6" }],
      ["line", { x1: "6", x2: "6.01", y1: "18", y2: "18" }],
    ],
    zap: [["path", { d: "M15.914 4a1.5 1.5 0 00-2.474-1.561l-9 9A1.5 1.5 0 005.5 14h4.002a.5.5 0 01.471.666L8.086 20a1.5 1.5 0 002.475 1.56l9-9A1.5 1.5 0 0018.5 10h-3.997a.5.5 0 01-.472-.667z" }]],
    command: [["path", { d: "M15 6v12a3 3 0 1 0 3-3H6a3 3 0 1 0 3 3V6a3 3 0 1 0-3 3h12a3 3 0 1 0-3-3" }]],
  } satisfies Record<string, SvgNode[]>;

  export type IconName = keyof typeof ICONS;
</script>

<script lang="ts">
  let { name, size = 16 }: { name: IconName; size?: 12 | 14 | 15 | 16 | 17 | 18 | 20 } = $props();
  const nodes = $derived(ICONS[name]);
</script>

<svg
  width={size}
  height={size}
  viewBox="0 0 24 24"
  fill="none"
  stroke="currentColor"
  stroke-width={1.8}
  stroke-linecap="round"
  stroke-linejoin="round"
  aria-hidden="true"
  class="shrink-0"
>
  {#each nodes as [tag, attrs]}
    {#if tag === "path"}
      <path {...attrs} />
    {:else if tag === "rect"}
      <rect {...attrs} />
    {:else if tag === "circle"}
      <circle {...attrs} />
    {:else}
      <line {...attrs} />
    {/if}
  {/each}
</svg>
