<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Clipboard, Events } from "@wailsio/runtime";
  import { runBench } from "./bench";
  import { blankProfile } from "./lib";
  import { ws } from "./workspace.svelte";
  import { leaves } from "./panes";
  import TabBar from "./TabBar.svelte";
  import Sidebar from "./Sidebar.svelte";
  import StatusBar from "./StatusBar.svelte";
  import PaneView from "./PaneView.svelte";
  import Palette from "./Palette.svelte";
  import ProfileForm from "./ProfileForm.svelte";
  import Logo from "./Logo.svelte";
  import BlockMenu from "./BlockMenu.svelte";
  import { copyText } from "./clipboard";
  import SettingsPanel from "./SettingsPanel.svelte";
  import ToolsPanel from "./ToolsPanel.svelte";
  import FilesPanel from "./FilesPanel.svelte";
  import ForwardsPanel from "./ForwardsPanel.svelte";
  import { settings } from "./settings.svelte";
  import { sb } from "./foldertree.svelte";
  import { FilesService, ForwardsService, ProfilesService, TerminalService, ToolsService, type Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

  const mac = navigator.userAgent.includes("Mac");
  let profiles = $state<Profile[]>([]);
  let folders = $state<string[]>([]);
  let toolsTab = $state<"health" | "backup" | "import" | "keys">("health");
  let banner = $state("");
  let palette = $state(false);
  let sidebar = $state(true);
  let settingsOpen = $state(false);
  let toolsOpen = $state(false);
  let filesTarget = $state("");
  let forwardsTarget = $state("");
  let form = $state<{ initial: Profile; oldName: string } | null>(null);
  let confirmDelete = $state<Profile | null>(null);
  let benchHost = $state<HTMLElement>();
  let customChrome = $state(false);
  let toast = $state("");
  let toastTimer: ReturnType<typeof setTimeout> | undefined;
  async function copied(what: string, text: string) {
    const ok = await copyText(text);
    toast = ok ? `Copied ${what}` : "Could not copy to the clipboard";
    clearTimeout(toastTimer);
    toastTimer = setTimeout(() => (toast = ""), 1800);
  }

  const open = $derived(new Set(ws.tabs.flatMap((t) => leaves(t.root).map((id) => ws.panes[id]?.target ?? "")).filter(Boolean)));
  const actions = [
    { id: "new-connection", label: "New connection", hint: mac ? "⌘N" : "Ctrl+Shift+N" },
    { id: "new-folder", label: "New folder", hint: mac ? "⇧⌘N" : "" },
    { id: "new-tab", label: "New local tab", hint: mac ? "⌘T" : "Ctrl+Shift+T" },
    { id: "split-right", label: "Split pane right", hint: mac ? "⌘D" : "Ctrl+Shift+D" },
    { id: "split-down", label: "Split pane down", hint: mac ? "⇧⌘D" : "Ctrl+Alt+D" },
    { id: "files", label: "Browse files on this connection", hint: mac ? "⇧⌘O" : "Ctrl+Shift+O" },
    { id: "forwards", label: "Manage port forwards on this connection", hint: mac ? "⇧⌘P" : "Ctrl+Shift+P" },
    { id: "tools", label: "Open tools: health, backup, import, keys", hint: "" },
    { id: "enable-blocks", label: "Enable command blocks in this session", hint: "" },
  ];

  async function refresh() {
    try {
      [profiles, folders] = [(await ProfilesService.List()) ?? [], (await ProfilesService.Folders()) ?? []];
    } catch (e) {
      banner = String(e);
    }
  }

  async function connect(p: Profile, split = false) {
    palette = false;
    banner = "";
    if (split && ws.active) await ws.split("row", p.name);
    else await ws.openTab(p.name);
  }

  function run(id: string) {
    palette = false;
    switch (id) {
      case "new-tab": return ws.openTab(null);
      case "new-connection": return newConnection(sb.selectedFolder);
      case "new-folder": return (sidebar = true, sb.startCreate(""));
      case "quick-open": return (palette = !palette);
      case "split-right": return ws.split("row");
      case "split-down": return ws.split("col");
      case "close": return ws.focusedPane && ws.closePane(ws.focusedPane);
      case "close-tab": return ws.activeId && ws.closeTab(ws.activeId);
      case "next-tab": return ws.cycle(1);
      case "prev-tab": return ws.cycle(-1);
      case "toggle-sidebar": return (sidebar = !sidebar);
      case "find": return ws.toggleSearch();
      case "settings": return (settingsOpen = !settingsOpen);
      case "tools": return (toolsTab = "health", toolsOpen = !toolsOpen);
      case "files": return openFiles();
      case "forwards": return openForwards();
      case "enable-blocks": return ws.focusedPane && ws.enableBlocks(ws.focusedPane);
    }
  }

  // Opens a local tab with the ssh-keygen command typed but not run, so the user
  // answers its prompts (including the passphrase) in the terminal and presses Enter.
  async function generateKey() {
    toolsOpen = false;
    await ws.openTab(null);
    ws.session(ws.focusedPane)?.ws.send(new TextEncoder().encode("ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519_relay"));
  }

  // The file browser works on a saved connection (it reuses that tab's login).
  function openFiles() {
    const t = ws.panes[ws.focusedPane]?.target;
    if (!t) return (banner = "Open a saved connection in this pane to browse its files.");
    banner = "";
    filesTarget = t;
  }

  // Always edit the stored profile, not a cached copy: forwards saved from their own
  // panel, a restore or an import may have changed it since the list was loaded.
  async function edit(p: Profile) {
    await refresh();
    form = { initial: profiles.find((x) => x.name === p.name) ?? p, oldName: p.name };
  }

  // Forwards are managed on a saved connection's live login.
  function openForwards() {
    const t = ws.panes[ws.focusedPane]?.target;
    if (!t) return (banner = "Open a saved connection in this pane to manage its port forwards.");
    banner = "";
    forwardsTarget = t;
  }

  // A new connection starts in the selected folder, if there is one.
  function newConnection(folder: string) {
    form = { initial: { ...blankProfile(), folder }, oldName: "" };
  }

  async function save(prof: Profile) {
    await ProfilesService.Save(form!.oldName, prof);
    form = null;
    await refresh();
  }
  async function remove(p: Profile) {
    confirmDelete = null;
    try {
      await ProfilesService.Delete(p.name);
      form = null;
      await refresh();
    } catch (e) {
      banner = String(e);
    }
  }

  // Windows and Linux have no app menu: Ctrl+Shift combos avoid clashing with
  // terminal control keys. On macOS the native menu owns the shortcuts.
  function keydown(e: KeyboardEvent) {
    if (mac || !e.ctrlKey || !e.shiftKey) return;
    const map: Record<string, string> = { T: "new-tab", N: "new-connection", M: "new-folder", K: "quick-open", D: "split-right", W: "close", B: "toggle-sidebar", F: "find", ",": "settings", O: "files", P: "forwards" };
    const id = map[e.key.toUpperCase()];
    if (id) { e.preventDefault(); run(id); }
  }

  // Theme, font and density apply to the page and to every open terminal, and persist.
  $effect(() => {
    settings.apply();
    ws.applySettings();
  });

  onMount(() => {
    const off = Events.On("menu", (e: { data: string }) => run(e.data));
    (async () => {
      customChrome = await TerminalService.CustomChrome();
      document.documentElement.classList.toggle("custom-chrome", customChrome);
      // RELAY_BENCH is "run", "idle", "smoke", or "smoke|<connection>|<remote dir>" to also test files
      const [mode, box, dir, sshdPort, localPort] = (await TerminalService.BenchMode()).split("|");
      if (mode === "smoke") return smoke(box, dir, sshdPort, localPort);
      if (mode) return runBench(benchHost!, mode);
      await refresh();
      await ws.openTab(null);
    })();
    return off;
  });

  // RELAY_BENCH=smoke: drive the real UI path (profiles, open a local tab,
  // split it, run a command in each pane) and report, so a build can be
  // verified without clicking.
  async function smoke(box = "", dir = "", sshdPort = "", localPort = "") {
    const out: Record<string, unknown> = { origin: location.origin };
    const enc = new TextEncoder();
    const jsErrors: string[] = [];
    window.addEventListener("error", (e) => jsErrors.push(e.message));
    window.addEventListener("unhandledrejection", (e) => jsErrors.push(String(e.reason)));
    out.jsErrors = jsErrors;
    const seen = (id: string, text: string) => {
      const b = ws.session(id)?.term.buffer.active;
      for (let y = 0; b && y < b.length; y++) if (b.getLine(y)?.translateToString().includes(text)) return true;
      return false;
    };
    try {
      await refresh();
      out.profiles = profiles.length;
      await ws.openTab(null);
      await ws.split("row");
      await ws.split("col");
      await ws.openTab(null);
      out.tabs = ws.tabs.length;
      const panes = ws.tabs.flatMap((t) => leaves(t.root));
      out.panes = panes.length;
      panes.forEach((id, i) => ws.session(id)?.ws.send(enc.encode(`echo smoke-${i}-$((6*7))\n`)));
      const deadline = Date.now() + 8000;
      while (Date.now() < deadline && !panes.every((id, i) => seen(id, `smoke-${i}-42`))) await new Promise((r) => setTimeout(r, 100));
      out.found = panes.map((id, i) => seen(id, `smoke-${i}-42`));
      // command blocks: the active tab's pane runs a good and a failing command
      const live = panes[panes.length - 1];
      // through xterm, like real typing, so the tracker sees Enter
      const send = (cmd: string) => ws.session(live)?.term.input(cmd + "\r", true);
      send("echo block-ok");
      send("false");
      for (let i = 0; i < 60 && !document.querySelector(".relay-block.err"); i++) await new Promise((r) => setTimeout(r, 100));
      out.blocksActive = ws.panes[live]?.blocks === true;
      // visible means positioned over the terminal, not merely present in the DOM
      const overlaid = (sel: string) => [...document.querySelectorAll<HTMLElement>(sel)].filter((e) => getComputedStyle(e).position === "absolute").length;
      // geometry and stacking: is each overlay inside the terminal and topmost at its centre?
      const probe = (e: HTMLElement) => {
        const r = e.getBoundingClientRect();
        const prev = e.style.pointerEvents;
        e.style.pointerEvents = "auto";
        const top = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
        e.style.pointerEvents = prev;
        return { x: Math.round(r.left), y: Math.round(r.top), w: Math.round(r.width), h: Math.round(r.height), topmost: top === e, covered: top ? top.tagName + "." + top.className : null, z: getComputedStyle(e).zIndex };
      };
      const term = ws.session(live)?.term;
      const screen = document.querySelector<HTMLElement>(".tabview.on .xterm-screen");
      const sr = screen?.getBoundingClientRect();
      out.webgl = ws.session(live)?.webgl;
      out.screen = sr ? { x: Math.round(sr.left), y: Math.round(sr.top), w: Math.round(sr.width), h: Math.round(sr.height) } : null;
      out.cols = term?.cols;
      out.headers = [...document.querySelectorAll<HTMLElement>(".tabview.on .relay-head")].map((e) => e.textContent);
      out.probes = [...document.querySelectorAll<HTMLElement>(".tabview.on .relay-block, .tabview.on .relay-badge")].map((e) => ({ cls: e.className, ...probe(e) }));
      out.canvases = [...document.querySelectorAll<HTMLElement>(".tabview.on .xterm-screen canvas")].map((c) => ({ cls: c.className, z: getComputedStyle(c).zIndex, pos: getComputedStyle(c).position }));
      // layout: bottom-anchored with padding below the live prompt, blocks bleeding edge to edge
      const pane = document.querySelector<HTMLElement>(".tabview.on .pane-term");
      const pr = pane?.getBoundingClientRect();
      // gap under the last used row (the live prompt), not under the screen's empty rows
      out.bottomGap = sr && pr && term ? Math.round(pr.bottom - (sr.top + (term.buffer.active.cursorY + 1) * (sr.height / term.rows))) : null;
      out.anchorShift = screen?.style.transform ?? null;
      const slider = document.querySelector<HTMLElement>(".tabview.on .scrollbar.vertical > .slider");
      const srr = slider?.getBoundingClientRect();
      out.scrollbar = srr && pr ? { width: Math.round(srr.width), fromRightEdge: Math.round(pr.right - srr.right) } : null;
      out.viewportBg = getComputedStyle(document.querySelector(".tabview.on .xterm-viewport")!).backgroundColor;
      out.fit = { rows: term?.rows, screenH: screen?.clientHeight, innerH: pr ? Math.round(pr.height - 32) : null, cellH: screen && term ? +(screen.clientHeight / term.rows).toFixed(2) : null, dpr: devicePixelRatio, font: term?.options.fontSize, lh: term?.options.lineHeight };
      out.blockEdges = [...document.querySelectorAll<HTMLElement>(".tabview.on .relay-block")].map((e) => {
        const r = e.getBoundingClientRect();
        return { left: Math.round(r.left - (pr?.left ?? 0)), right: Math.round((pr?.right ?? 0) - r.right) };
      });
      out.okBlocks = overlaid(".relay-block.ok");
      out.errBlocks = overlaid(".relay-block.err");
      out.badges = [...document.querySelectorAll(".relay-badge")].map((b) => b.textContent);
      // scrollback search through the real find bar
      ws.focus(live);
      ws.toggleSearch();
      await tick();
      const find = document.querySelector<HTMLInputElement>('[aria-label="Find in scrollback"]');
      if (find) {
        find.value = "block-ok";
        find.dispatchEvent(new Event("input", { bubbles: true }));
        for (let i = 0; i < 30 && !/of/.test(document.querySelector("[role=search] .count")?.textContent ?? ""); i++) await new Promise((r) => setTimeout(r, 100));
      }
      out.search = { opened: !!find, count: document.querySelector("[role=search] .count")?.textContent ?? null };
      ws.closeSearch();
      await tick();
      out.searchClosed = !document.querySelector('[aria-label="Find in scrollback"]');
      // settings apply to the page chrome and to live terminals, then restore
      const before = { theme: settings.theme, size: settings.fontSize };
      const colsBefore = ws.session(live)?.term.cols;
      settings.theme = "obsidian";
      settings.fontSize = 17;
      await tick();
      await new Promise((r) => setTimeout(r, 300));
      const t2 = ws.session(live)?.term;
      out.settings = {
        cssTerm: getComputedStyle(document.documentElement).getPropertyValue("--term").trim(),
        xtermBg: (t2?.options.theme as { background?: string } | undefined)?.background,
        fontSize: t2?.options.fontSize,
        colsChanged: t2?.cols !== colsBefore,
      };
      // the light theme and "system" (which follows the OS) reach CSS, colour-scheme and xterm
      const probeTheme = async (name: string) => {
        settings.theme = name;
        await tick();
        await new Promise((r) => setTimeout(r, 200));
        const root = document.documentElement;
        return {
          term: getComputedStyle(root).getPropertyValue("--term").trim(),
          scheme: root.style.colorScheme,
          xtermBg: (ws.session(live)?.term.options.theme as { background?: string } | undefined)?.background,
          resolved: settings.resolved,
        };
      };
      out.themes = { daylight: await probeTheme("daylight"), system: await probeTheme("system") };
      settings.theme = before.theme;
      settings.fontSize = before.size;
      await tick();
      // tools: the real panel against the real backend
      toolsOpen = true;
      await tick();
      for (let i = 0; i < 30 && !document.querySelector(".checks li"); i++) await new Promise((r) => setTimeout(r, 100));
      const healthRows = document.querySelectorAll(".checks li").length;
      [...document.querySelectorAll<HTMLElement>("[aria-label=Tools] nav button")].find((b) => b.textContent === "Keys")?.click();
      for (let i = 0; i < 30 && !document.querySelector(".keys li"); i++) await new Promise((r) => setTimeout(r, 100));
      const keyRows = document.querySelectorAll(".keys li").length;
      toolsOpen = false;
      await tick();
      // backup then restore through the bindings; the profile count must survive the round trip
      const countBefore = (await ProfilesService.List())?.length;
      const dest = "/tmp/relay-smoke-backup";
      await ToolsService.Backup(dest, "smoke-pass");
      await ToolsService.Restore(dest + ".age", "smoke-pass");
      out.tools = { healthRows, keyRows, profilesBefore: countBefore, profilesAfter: (await ProfilesService.List())?.length, needsPass: await ToolsService.NeedsPassphrase(dest + ".age") };
      // files: the real panel and service against a real sshd (only when a connection is given)
      if (box && dir) {
        filesTarget = box;
        await new Promise((r) => setTimeout(r, 1500));
        const rows = [...document.querySelectorAll(".item .name")].map((n) => n.textContent);
        // ssh trusts known_hosts from the passwd home, not $HOME, so a throwaway server
        // is untrusted here: the panel must show the readable host-key message.
        for (let i = 0; i < 60 && !document.querySelector(".msg.bad"); i++) await new Promise((r) => setTimeout(r, 100));
        const shown = document.querySelector(".msg.bad")?.textContent ?? null;
        filesTarget = "";
        await tick();
        out.files = { panelRows: rows, error: shown, panelClosed: !document.querySelector("[aria-label=Files][role=dialog]") };
      }
      // port forwards: the real panel and service against a live multiplex master started by the harness
      if (box && sshdPort && localPort) {
        const before = await ForwardsService.Status(box);
        forwardsTarget = box;
        await tick();
        await ForwardsService.Add(box, "local", localPort, "127.0.0.1:" + sshdPort);
        for (let i = 0; i < 60 && !document.querySelector(".panel .row:not(.head)"); i++) await new Promise((r) => setTimeout(r, 100));
        const after = await ForwardsService.Status(box);
        const rows = [...document.querySelectorAll(".panel .row:not(.head)")].map((r) => r.textContent?.replace(/\s+/g, " ").trim());
        await ForwardsService.Cancel(box, "local", localPort, "127.0.0.1:" + sshdPort);
        const gone = await ForwardsService.Status(box);
        forwardsTarget = "";
        out.forwards = { connectedBefore: before.connected, connectedState: !!document.querySelector(".state"), listening: after.forwards?.[0]?.listening, uiRows: rows, afterCancel: gone.forwards?.length };
      }
      // edit form shows a forward saved elsewhere (regression: it used a stale cached profile)
      if (profiles.length) {
        const target0 = profiles[0];
        const fresh = (await ProfilesService.List())!.find((x) => x.name === target0.name)!;
        await ProfilesService.Save(target0.name, { ...fresh, tunnels: [...(fresh.tunnels ?? []), { type: "local", listen: "7777", target: "localhost:7" }] });
        await edit(target0); // the sidebar list was NOT refreshed by the save above
        await tick();
        out.editShowsSavedForward = [...document.querySelectorAll(".tun input")].some((i) => (i as HTMLInputElement).value === "7777");
        form = null;
        await ProfilesService.Save(target0.name, fresh); // restore
        await refresh();
      }
      // window chrome: frameless means no native title bar, and our buttons sit centred in the strip
      const strip = document.querySelector<HTMLElement>(".strip")?.getBoundingClientRect();
      const btns = [...document.querySelectorAll<HTMLElement>(".controls button")].map((b) => b.getBoundingClientRect());
      out.chrome = {
        custom: customChrome,
        titleBarHeight: outerHeight - innerHeight,
        buttons: btns.length,
        offCentre: strip && btns.length ? Math.max(...btns.map((r) => Math.abs(r.top + r.height / 2 - (strip.top + strip.height / 2)))) : null,
        inset: btns[0] && strip ? Math.round(btns[0].left - strip.left) : null,
        appRadius: getComputedStyle(document.getElementById("app")!).borderTopLeftRadius,
        rootClass: document.documentElement.classList.contains("custom-chrome"),
        htmlBg: getComputedStyle(document.documentElement).backgroundColor,
      };
      // sidebar: create folders and move things by dragging, through the real UI
      {
        const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));
        sidebar = true;
        await tick();
        const makeFolder = async (name: string) => {
          document.querySelector<HTMLElement>("[data-add-button]")!.click();
          await tick();
          [...document.querySelectorAll<HTMLElement>("[role=menuitem]")].find((b) => b.querySelector(".t")?.textContent === "Folder")!.click();
          await tick();
          const input = document.querySelector<HTMLInputElement>('input[placeholder="Folder name"]')!;
          input.value = name;
          input.dispatchEvent(new Event("input", { bubbles: true }));
          input.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter", bubbles: true }));
          await wait(500);
        };
        const drag = async (from: Element, to: Element) => {
          const f = from.getBoundingClientRect(), t = to.getBoundingClientRect();
          const [fx, fy, tx, ty] = [f.left + 60, f.top + f.height / 2, t.left + 80, t.top + t.height / 2];
          from.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, clientX: fx, clientY: fy, pointerId: 1 }));
          window.dispatchEvent(new PointerEvent("pointermove", { clientX: fx + 12, clientY: fy + 12 }));
          window.dispatchEvent(new PointerEvent("pointermove", { clientX: tx, clientY: ty }));
          await tick();
          window.dispatchEvent(new PointerEvent("pointerup", { clientX: tx, clientY: ty }));
          await tick();
        };
        const confirmMove = async () => {
          const card = document.querySelector('[role=alertdialog]');
          const text = card?.querySelector(".text")?.textContent ?? null;
          card?.querySelector<HTMLElement>(".btn.primary")?.click();
          await wait(600);
          return text;
        };
        const rowOf = (name: string) => [...document.querySelectorAll(".row")].find((r) => r.querySelector(".name")?.textContent === name)!;
        const folderOf = (name: string) => [...document.querySelectorAll(".head")].find((r) => r.querySelector(".label-text")?.textContent === name)!;

        const victim = profiles.find((p) => !p.folder) ?? profiles[0];
        const originalFolder = victim.folder;
        await makeFolder("smoke-a");
        await makeFolder("smoke-b");
        const created = await ProfilesService.Folders();
        await drag(rowOf(victim.name), folderOf("smoke-a"));
        const sawConfirm = await confirmMove();
        const movedTo = (await ProfilesService.List())!.find((p) => p.name === victim.name)!.folder;
        await drag(folderOf("smoke-b"), folderOf("smoke-a"));
        const sawFolderConfirm = await confirmMove();
        const nested = await ProfilesService.Folders();
        // a folder cannot be dropped into itself: no confirmation appears
        await drag(folderOf("smoke-a"), folderOf("smoke-a"));
        const selfDropCard = !!document.querySelector("[role=alertdialog]");
        sb.cancelDrag();
        out.sidebar = { created, sawConfirm, movedTo, sawFolderConfirm, nested, selfDropCard };
        // clean up
        await ProfilesService.MoveProfile(victim.name, originalFolder);
        for (const f of ["smoke-a/smoke-b", "smoke-a", "smoke-b"]) await ProfilesService.DeleteFolder(f).catch(() => {});
        await refresh();
      }
      // block menu: hover a block, open its menu, copy three ways, and read the real clipboard back
      {
        const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));
        const termEl = ws.session(live)?.term.element;
        const okBlock = document.querySelector<HTMLElement>(".tabview.on .relay-block.ok");
        const menu: Record<string, unknown> = {};
        if (termEl && okBlock) {
          const r = okBlock.getBoundingClientRect();
          termEl.dispatchEvent(new MouseEvent("mousemove", { bubbles: true, clientX: r.left + 80, clientY: r.top + 8 }));
          await wait(100);
          menu.hotBadges = document.querySelectorAll(".tabview.on .relay-badge.hot").length;
          const copies: Record<string, string> = {};
          for (const label of ["Copy command", "Copy output", "Copy command and output"]) {
            const hot = document.querySelector<HTMLElement>(".tabview.on .relay-badge.hot .block-menu");
            hot?.click();
            await wait(150);
            const item = [...document.querySelectorAll<HTMLButtonElement>('[aria-label="Block actions"] [role=menuitem]')].find((b) => b.textContent?.includes(label) && !b.textContent.includes(label + " and"));
            const exact = label === "Copy command and output" ? [...document.querySelectorAll<HTMLButtonElement>('[aria-label="Block actions"] [role=menuitem]')].find((b) => b.textContent?.includes("and output")) : item;
            exact?.click();
            await wait(200);
            copies[label] = await Clipboard.Text();
          }
          menu.copies = copies;
          menu.toast = document.querySelector(".toast")?.textContent?.trim() ?? null;
          menu.menuClosed = !document.querySelector('[aria-label="Block actions"]');
        }
        out.blockMenu = menu;
      }
      // ended pane -> reconnect
      const victim = panes[0];
      ws.session(victim)?.ws.close();
      await new Promise((r) => setTimeout(r, 400));
      out.endedDetected = ws.panes[victim]?.ended === true;
      await ws.reconnect(victim);
      ws.session(victim)?.ws.send(enc.encode("echo again-$((6*7))\n"));
      for (let i = 0; i < 40 && !seen(victim, "again-42"); i++) await new Promise((r) => setTimeout(r, 100));
      out.reconnected = seen(victim, "again-42");
      ws.closePane(panes[1]);
      out.afterClose = ws.tabs.flatMap((t) => leaves(t.root)).length;
      // connecting: a saved connection shows a loading screen until ssh first writes, and can be cancelled
      if (box) {
        const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
        const sample = (id: string) => ({
          connecting: ws.panes[id]?.connecting,
          overlay: !!document.querySelector(".tabview.on .connecting"),
          status: [...document.querySelectorAll("footer")].at(-1)?.querySelector(".seg")?.textContent?.trim(),
        });
        await ws.openTab(box);
        const fast = ws.focusedPane;
        const t0 = Date.now();
        for (let i = 0; i < 100 && ws.panes[fast]?.connecting; i++) await sleep(50);
        out.connectFast = { ms: Date.now() - t0, ...sample(fast), received: ws.session(fast)?.received };
        await ws.openTab("slow");
        const slow = ws.focusedPane;
        await sleep(1000);
        out.dbg = { views: [...document.querySelectorAll(".tabview")].map((e) => e.className + ":" + e.querySelectorAll(".connecting").length), footer: [...document.querySelectorAll("footer")].map((f) => f.textContent), active: ws.activeId, focused: ws.focusedPane, slow, pane: JSON.stringify(ws.panes[slow]) };
        const early = { ...sample(slow), time: document.querySelector(".tabview.on .connecting .time")?.textContent, hint: document.querySelector(".tabview.on .hint.on")?.textContent ?? null };
        await sleep(9000);
        const hint = document.querySelector(".tabview.on .hint.on")?.textContent ?? null;
        const stillWaiting = sample(slow);
        document.querySelector<HTMLButtonElement>(".tabview.on .connecting .btn")?.click();
        await sleep(600);
        out.connectSlow = { early, hint, stillWaiting, cancelled: { ...sample(slow), ended: ws.panes[slow]?.ended, error: ws.panes[slow]?.error, banner: document.querySelector(".tabview.on .ended")?.textContent?.replace(/\s+/g, " ").trim() } };
      }
    } catch (e) {
      out.error = String(e);
    }
    await TerminalService.ReportBench(JSON.stringify(out));
  }
</script>

<svelte:window onkeydown={keydown} />

<div class="app">
  <TabBar custom={customChrome} onsidebar={() => (sidebar = !sidebar)} onsettings={() => (settingsOpen = !settingsOpen)} ontools={() => { toolsTab = "health"; toolsOpen = !toolsOpen; }} onfiles={openFiles} onforwards={openForwards} />
  <div class="body">
    {#if sidebar}
      <Sidebar {profiles} {folders} {open} current={ws.panes[ws.focusedPane]?.target ?? ""} onconnect={(p) => connect(p)} onnew={newConnection} onedit={edit} onpalette={() => (palette = true)} ontools={(t) => { toolsTab = t; toolsOpen = true; }} onchanged={refresh} />
    {/if}
    <main>
      {#if banner}<p class="banner" role="alert">{banner}<button aria-label="Dismiss" onclick={() => (banner = "")}>&times;</button></p>{/if}
      <div class="stage" bind:this={benchHost}>
        {#each ws.tabs as t (t.id)}
          <div class="tabview" class:on={t.id === ws.activeId}>
            <PaneView node={t.root} tabId={t.id} multi={leaves(t.root).length > 1} {profiles} />
          </div>
        {:else}
          <div class="welcome"><Logo size={72} /><h1>Relay</h1><p>Open a connection from the list, or press <span class="kbd">{mac ? "⌘T" : "Ctrl+Shift+T"}</span> for a local shell.</p></div>
        {/each}
      </div>
    </main>
  </div>
  <StatusBar {profiles} themeName={settings.label} />
</div>

{#if forwardsTarget}<ForwardsPanel target={forwardsTarget} onchanged={refresh} onclose={() => (forwardsTarget = "")} onopen={() => { const t = forwardsTarget; forwardsTarget = ""; ws.openTab(t); }} />{/if}
{#if filesTarget}<FilesPanel target={filesTarget} onclose={() => (filesTarget = "")} />{/if}
{#if toolsOpen}<ToolsPanel initial={toolsTab} onclose={() => (toolsOpen = false)} onchanged={refresh} ongenerate={generateKey} />{/if}
{#if ws.blockMenu}<BlockMenu block={ws.blockMenu.block} rect={ws.blockMenu.rect} oncopy={copied} onclose={() => (ws.blockMenu = null)} />{/if}
{#if toast}<div class="toast" role="status"><svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M5 12l5 5 9-10" /></svg>{toast}</div>{/if}
{#if settingsOpen}<SettingsPanel onclose={() => (settingsOpen = false)} />{/if}
{#if palette}<Palette {profiles} active={open} {actions} onconnect={connect} onaction={run} onclose={() => (palette = false)} />{/if}
{#if form}
  <ProfileForm initial={form.initial} oldName={form.oldName} {profiles} {folders} onsave={save} ondelete={() => (confirmDelete = profiles.find((p) => p.name === form!.oldName) ?? null)} oncancel={() => (form = null)} />
{/if}
{#if confirmDelete}
  <div class="scrim" role="presentation" onkeydown={(e) => e.key === "Escape" && (confirmDelete = null)}>
    <div class="confirm" role="alertdialog" aria-label="Delete connection">
      <p>Delete <strong>{confirmDelete.name}</strong>? This removes it from connections.toml and your SSH config.</p>
      <div class="buttons"><button class="btn" onclick={() => (confirmDelete = null)}>Cancel</button><button class="btn primary danger-fill" onclick={() => remove(confirmDelete!)}>Delete</button></div>
    </div>
  </div>
{/if}

<style>
  .app { display: flex; flex-direction: column; height: 100%; }
  .body { flex: 1; min-height: 0; display: flex; }
  main { flex: 1; min-width: 0; display: flex; flex-direction: column; background: var(--term); }
  .stage { flex: 1; min-height: 0; position: relative; }
  .tabview { position: absolute; inset: 0; display: none; }
  .tabview.on { display: flex; }
  .welcome { position: absolute; inset: 0; display: grid; place-content: center; text-align: center; color: var(--muted); }
  .welcome h1 { color: var(--text); margin: 14px 0 6px; font-weight: 600; }
  .welcome { justify-items: center; }
  .banner { margin: 0; padding: 8px 14px; display: flex; align-items: center; color: var(--err-text); background: var(--err-bg); border-bottom: 1px solid var(--err); }
  .banner button { all: unset; margin-left: auto; cursor: pointer; font-size: 16px; }
  .scrim { position: fixed; inset: 0; background: var(--scrim); display: flex; align-items: center; justify-content: center; z-index: 30; }
  .confirm { width: min(420px, 92vw); background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; padding: 18px 20px; }
  .confirm p { margin: 0 0 16px; }
  .buttons { display: flex; justify-content: flex-end; gap: 8px; }
  .toast { position: fixed; left: 50%; bottom: 44px; transform: translateX(-50%); z-index: 60; display: flex; align-items: center; gap: 8px; padding: 9px 16px; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 10px 30px var(--shadow); font-size: 12.5px; color: var(--text); }
  .toast svg { color: var(--ok); }
  .danger-fill { background: var(--err); border-color: var(--err); color: var(--on-err); }
</style>
