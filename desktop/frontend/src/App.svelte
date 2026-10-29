<script lang="ts">
  import { onMount, tick } from "svelte";
  import { Clipboard, Events } from "@wailsio/runtime";
  import { blankProfile, nextSession } from "./lib/lib";
  import { ws } from "./state/workspace.svelte";
  import { leaves } from "./lib/panes";
  import TabBar from "./components/TabBar.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import StatusBar from "./components/StatusBar.svelte";
  import PaneView from "./components/PaneView.svelte";
  import DragGhost from "./components/DragGhost.svelte";
  import Logo from "./components/Logo.svelte";
  import Icon from "./components/Icon.svelte";
  import Modal from "./components/Modal.svelte";
  import BlockMenu from "./components/BlockMenu.svelte";
  import { copyText } from "./lib/clipboard";
  import { settings } from "./state/settings.svelte";
  import { sb } from "./state/foldertree.svelte";
  import { vault } from "./state/vault.svelte";
  import { FilesService, ForwardsService, ProfilesService, SnippetsService, TerminalService, ToolsService, VaultService, type Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

  const mac = navigator.userAgent.includes("Mac");
  let profiles = $state<Profile[]>([]);
  let folders = $state<string[]>([]);
  let toolsTab = $state<"health" | "backup" | "import" | "keys">("health");
  let banner = $state("");
  let palette = $state(false);
  let sidebar = $state(true);
  let settingsOpen = $state(false);
  let toolsOpen = $state(false);
  let settingsSection = $state<"vault" | undefined>(undefined);
  let unlockConnect = $state<{ profile: Profile; split: boolean } | null>(null);
  let VaultPanel = $state<typeof import("./components/VaultPanel.svelte").default | null>(null);
  $effect(() => { if (unlockConnect && !VaultPanel) import("./components/VaultPanel.svelte").then((m) => (VaultPanel = m.default)); });
  let filesTarget = $state("");
  let forwardsTarget = $state("");
  let snippetsOpen = $state(false);
  let form = $state<{ initial: Profile; oldName: string } | null>(null);
  // Loaded the first time the form opens; the {#if} keeps `form` non-null while it renders.
  let ProfileForm = $state<typeof import("./components/ProfileForm.svelte").default | null>(null);
  $effect(() => { if (form && !ProfileForm) import("./components/ProfileForm.svelte").then((m) => (ProfileForm = m.default)); });
  let confirmDelete = $state<Profile | null>(null);
  // A modal keeps rendering while it fades out, after its state is already null: feed it the last value.
  let lastForm: typeof form = null;
  const formShown = $derived.by(() => (form ? (lastForm = form) : lastForm));
  let lastDelete: Profile | null = null;
  const deleting = $derived.by(() => (confirmDelete ? (lastDelete = confirmDelete) : lastDelete));
  let lastUnlock: typeof unlockConnect = null;
  const unlocking = $derived.by(() => (unlockConnect ? (lastUnlock = unlockConnect) : lastUnlock));
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
    { id: "snippets", label: "Snippets", hint: mac ? "⇧⌘S" : "Ctrl+Shift+S" },
    { id: "settings", label: "Open settings", hint: mac ? "⌘," : "Ctrl+Shift+," },
    { id: "vault", label: "Open vault settings", hint: "" },
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
    if (p.auth === "password" && vault.exists && !vault.unlocked && vault.hasRemembered(p.name)) {
      unlockConnect = { profile: p, split };
      return;
    }
    await doConnect(p, split);
  }

  function openSettings(section?: "vault") {
    settingsSection = section;
    settingsOpen = section ? true : !settingsOpen;
  }

  async function doConnect(p: Profile, split = false) {
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
      case "settings": return openSettings();
      case "tools": return (toolsTab = "health", toolsOpen = !toolsOpen);
      case "vault": return openSettings("vault");
      case "files": return openFiles();
      case "forwards": return openForwards();
      case "snippets": return (snippetsOpen = !snippetsOpen);
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

  async function save(prof: Profile, password = "") {
    if (password) await ProfilesService.SaveWithPassword(form!.oldName, prof, password);
    else await ProfilesService.Save(form!.oldName, prof);
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
    const map: Record<string, string> = { T: "new-tab", N: "new-connection", M: "new-folder", K: "quick-open", D: "split-right", W: "close", B: "toggle-sidebar", F: "find", ",": "settings", O: "files", P: "forwards", S: "snippets" };
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
    ws.blocksMode = (name) => profiles.find((p) => p.name === name)?.blocks;
    ws.profileAuth = (name) => profiles.find((p) => p.name === name)?.auth;
    sb.onFocusConnection = (name) => {
      const ids = ws.tabs.flatMap((t) => leaves(t.root)).filter((id) => ws.panes[id]?.target === name);
      const id = nextSession(ids, ws.focusedPane);
      if (id) ws.focus(id);
    };
    sb.onPaneToTab = (id, tab) => ws.movePaneToTab(id, tab);
    sb.onPaneMove = (id, target, edge) => ws.movePane(id, target, edge);
    sb.onPaneDrop = (name, paneId, edge) => { banner = ""; return edge === "center" ? ws.openTab(name) : ws.splitAt(paneId, edge, name); };
    const offVault = Events.On("vault", () => vault.refresh());
    vault.refresh();
    VaultService.SetIdleTimeout(settings.vaultTimeout).catch(() => {});
    (async () => {
      customChrome = await TerminalService.CustomChrome();
      document.documentElement.classList.toggle("custom-chrome", customChrome);
      // RELAY_BENCH is "run", "idle", "smoke", or "smoke|<connection>|<remote dir>" to also test files
      const [mode, box, dir, sshdPort, localPort] = (await TerminalService.BenchMode()).split("|");
      if (mode === "smoke") return smoke(box, dir, sshdPort, localPort);
      if (mode) return (await import("./lib/bench")).runBench(benchHost!, mode);
      await refresh();
      await ws.openTab(null);
    })();
    return () => { off(); offVault(); };
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
    // the Content-Security-Policy must not block anything the app needs
    document.addEventListener("securitypolicyviolation", (e) => jsErrors.push(`CSP blocked ${e.violatedDirective}: ${e.blockedURI}`));
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
      for (let i = 0; i < 30 && !document.querySelector("[role=dialog] .rows li"); i++) await new Promise((r) => setTimeout(r, 100));
      const healthRows = document.querySelectorAll("[role=dialog] .rows li").length;
      [...document.querySelectorAll<HTMLElement>("[aria-label=Tools] [role=tab]")].find((b) => b.textContent === "Keys")?.click();
      for (let i = 0; i < 30 && !document.querySelector("[role=dialog] li.key"); i++) await new Promise((r) => setTimeout(r, 100));
      const keyRows = document.querySelectorAll("[role=dialog] li.key").length;
      toolsOpen = false;
      await tick();
      // backup then restore through the bindings; the profile count must survive the round trip
      const countBefore = (await ProfilesService.List())?.length;
      const dest = "/tmp/relay-smoke-backup";
      await ToolsService.Backup(dest, "smoke-pass");
      await ToolsService.Restore(dest + ".age", "smoke-pass");
      out.tools = { healthRows, keyRows, profilesBefore: countBefore, profilesAfter: (await ProfilesService.List())?.length, needsPass: await ToolsService.NeedsPassphrase(dest + ".age") };
      // keys with a passphrase: adopting asks for it, refuses a wrong one, and the adopted key then shows up in the form's picker
      {
        const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));
        const setVal = (el: HTMLInputElement, v: string) => { el.value = v; el.dispatchEvent(new Event("input", { bubbles: true })); };
        const keyRow = (name: string) => [...document.querySelectorAll<HTMLElement>("[role=dialog] li.key")].find((li) => li.querySelector(".kname b")?.textContent === name);
        toolsTab = "keys";
        toolsOpen = true;
        await tick();
        for (let i = 0; i < 30 && !keyRow("id_locked"); i++) await wait(100);
        const row0 = keyRow("id_locked");
        const chips = [...(row0?.querySelectorAll(".chip") ?? [])].map((c) => c.textContent);
        [...(row0?.querySelectorAll<HTMLElement>("button") ?? [])].find((x) => /^(Adopt|Remember)/.test(x.textContent?.trim() ?? ""))?.click();
        await wait(500);
        const asked = !!keyRow("id_locked")?.querySelector(".pass input[type=password]");
        const submit = async (pass: string) => {
          setVal(keyRow("id_locked")!.querySelector<HTMLInputElement>(".pass input")!, pass);
          await tick();
          keyRow("id_locked")!.querySelector<HTMLElement>(".pass button[type=submit]")?.click();
          await wait(1500);
        };
        await submit("not-the-passphrase");
        const wrong = { error: document.querySelector("[role=dialog] .notice.bad")?.textContent ?? null, stillAsking: !!keyRow("id_locked")?.querySelector(".pass"), adopted: /in backups/i.test(keyRow("id_locked")?.textContent ?? "") };
        await submit("smokepass");
        const right = { adopted: /in backups/i.test(keyRow("id_locked")?.textContent ?? ""), asking: !!keyRow("id_locked")?.querySelector(".pass") };
        toolsOpen = false;
        await tick();
        newConnection("");
        await tick();
        await wait(500);
        [...document.querySelectorAll<HTMLElement>('[role=dialog] .seg button')].find((b) => b.textContent?.includes("Key file"))?.click();
        await tick();
        const sel = document.querySelector<HTMLSelectElement>('select[aria-label=Key]');
        const options = [...(sel?.querySelectorAll("option") ?? [])].map((o) => o.textContent?.trim());
        const groups = [...(sel?.querySelectorAll("optgroup") ?? [])].map((g) => g.label);
        if (sel) { sel.value = "a:id_locked"; sel.dispatchEvent(new Event("change", { bubbles: true })); }
        await tick();
        out.keysWithPassphrase = { chips, asked, wrong, right, picker: { options, groups }, picked: (document.querySelector<HTMLSelectElement>('select[aria-label=Key]'))?.value };
        form = null;
        await tick();
      }
      // vault: status bar lock button, create vault dialog, connection form password UI, lock/unlock
      {
        const wait = (ms: number) => new Promise((r) => setTimeout(r, ms));
        const setVal = (el: HTMLInputElement, v: string) => { el.value = v; el.dispatchEvent(new Event("input", { bubbles: true })); };
        const stBefore = await vault.refresh();

        // 1. Click status bar vault button to open VaultPanel
        const until = async (probe: () => unknown, ms = 8000) => { for (let t = 0; t < ms && !probe(); t += 50) await wait(50); };
        document.querySelector<HTMLElement>(".vault-btn")?.click();
        await tick();
        await until(() => [...document.querySelectorAll("[role=dialog] h2")].some((h) => h.textContent === "Settings")); // a lazy chunk; a form fading out may still be on screen

        // The status-bar chip opens Settings at its Vault section; creating the vault opens the small prompt on top.
        const openTitle = [...document.querySelectorAll("[role=dialog] h2")].map((h) => h.textContent).find((t) => t === "Settings") ?? "";
        let created = false;
        if (!stBefore.exists) {
          [...document.querySelectorAll<HTMLElement>("[role=dialog] button")].find((x) => x.textContent?.trim() === "Create vault")?.click();
          await until(() => document.querySelector('[role=dialog] input[aria-label="Master password"]'));
          const pass1 = document.querySelector<HTMLInputElement>('[role=dialog] input[aria-label="Master password"]');
          const pass2 = document.querySelector<HTMLInputElement>('[role=dialog] input[aria-label="Confirm master password"]');
          if (pass1 && pass2) {
            setVal(pass1, "smoke-master-pass");
            setVal(pass2, "smoke-master-pass");
            await tick();
            document.querySelector<HTMLElement>('[role=dialog] button[type=submit]')?.click();
            await wait(800);
          }
          const stAfterCreate = await vault.refresh();
          created = stAfterCreate.exists && stAfterCreate.unlocked;
        } else {
          created = stBefore.exists;
        }

        // Close vault dialog if open
        settingsOpen = false;
        await tick();
        await wait(200);

        // 2. Open new connection form, switch auth to password
        newConnection("");
        await tick();
        await wait(400);

        [...document.querySelectorAll<HTMLElement>('[role=dialog] .seg button')].find((b) => b.textContent?.includes("Password"))?.click();
        await tick();
        await wait(200);

        const formNote = document.querySelector("[role=dialog] .hint")?.textContent ?? "";
        const hasCorrectNote = formNote.includes("Stored encrypted in Relay's vault") && !formNote.includes("It is never stored");

        const nameInput = document.querySelector<HTMLInputElement>('input[placeholder="web-1"]');
        const passInput = document.querySelector<HTMLInputElement>('input[type=password][aria-label="Password"]');

        let savedBadgeShown = false;
        if (nameInput && passInput) {
          setVal(nameInput, "smoke-vault-conn");
          const inputs = [...document.querySelectorAll<HTMLInputElement>("input")];
          const uIn = inputs.find((i) => i.value === "root" || i.closest("label")?.textContent?.includes("User"));
          const hIn = inputs.find((i) => i.closest("label")?.textContent?.includes("Host"));
          if (hIn) setVal(hIn, "127.0.0.1");
          if (uIn) setVal(uIn, "smokeuser");
          setVal(passInput, "smoke-secret-pass");
          await tick();

          document.querySelector<HTMLFormElement>("form")?.dispatchEvent(new Event("submit", { bubbles: true }));
          await wait(800);

          // 3. Re-open the connection form for editing to verify "A password is saved" badge
          await refresh();
          const savedConn = profiles.find((p) => p.name === "smoke-vault-conn");
          if (savedConn) {
            edit(savedConn);
            await tick();
            await wait(400);
            savedBadgeShown = [...document.querySelectorAll("[role=dialog] .chip.ok")].some((c) => c.textContent?.includes("A password is saved"));
            form = null;
            await tick();
            await wait(200);
          }
        }

        // 4. Lock vault
        await vault.lock();
        const stLocked = await vault.refresh();

        // 5. Cleanup profile
        await ProfilesService.Delete("smoke-vault-conn").catch(() => {});
        await refresh();

        out.vault = {
          openTitle,
          created,
          hasCorrectNote,
          savedBadgeShown,
          lockedAfterLock: !stLocked.unlocked,
          vaultExists: stLocked.exists,
        };
      }
      // files: the real panel and service against a real sshd (only when a connection is given)
      if (box && dir) {
        filesTarget = box;
        await new Promise((r) => setTimeout(r, 1500));
        const rows = [...document.querySelectorAll(".item .name")].map((n) => n.textContent);
        // ssh trusts known_hosts from the passwd home, not $HOME, so a throwaway server
        // is untrusted here: the panel must show the readable host-key message.
        for (let i = 0; i < 60 && !document.querySelector("[role=dialog] .notice.bad"); i++) await new Promise((r) => setTimeout(r, 100));
        const shown = document.querySelector("[role=dialog] .notice.bad")?.textContent ?? null;
        filesTarget = "";
        await tick();
        await new Promise((r) => setTimeout(r, 400)); // the panel fades out
        out.files = { panelRows: rows, error: shown, panelClosed: ![...document.querySelectorAll("[role=dialog] h2")].some((h) => h.textContent === "Files") };
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
          // press the row's own button, as a real mouse does
          (from.querySelector(".main, .name") ?? from).dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, clientX: fx, clientY: fy, pointerId: 1 }));
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
      {
        // the "session ended" banner sits near the top of its pane, and can be closed
        ws.focus(victim); // its tab may not be the visible one
        await tick();
        await new Promise((r) => setTimeout(r, 200));
        const banner = document.querySelector<HTMLElement>(".tabview.on .ended");
        const paneBox = banner?.closest<HTMLElement>("[data-pane]")?.getBoundingClientRect();
        const b = banner?.getBoundingClientRect();
        const shown = { present: !!banner, inUpperHalf: !!(b && paneBox && b.top < paneBox.top + paneBox.height / 2) };
        document.querySelector<HTMLElement>(".tabview.on .ended .dismiss")?.click();
        await new Promise((r) => setTimeout(r, 200));
        out.endedBanner = { ...shown, dismissed: !document.querySelector(".tabview.on .ended"), stillEnded: ws.panes[victim]?.ended === true };
      }
      await ws.reconnect(victim);
      ws.session(victim)?.ws.send(enc.encode("echo again-$((6*7))\n"));
      for (let i = 0; i < 40 && !seen(victim, "again-42"); i++) await new Promise((r) => setTimeout(r, 100));
      out.reconnected = seen(victim, "again-42");
      ws.closePane(panes[1]);
      out.afterClose = ws.tabs.flatMap((t) => leaves(t.root)).length;
      // snippets: save one through the form, then run and insert it into the focused local pane
      {
        const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
        const local = panes[0];
        ws.focus(local);
        await tick();
        run("snippets");
        await tick();
        await sleep(300);
        const setValue = (el: HTMLInputElement | HTMLTextAreaElement, v: string) => { el.value = v; el.dispatchEvent(new Event("input", { bubbles: true })); };
        [...document.querySelectorAll<HTMLElement>("[role=dialog] .btn.primary")].find((b) => b.textContent?.trim() === "New snippet")?.click();
        await tick();
        await sleep(200);
        setValue(document.querySelector<HTMLInputElement>('[role=dialog] .form input')!, "smoke snippet");
        setValue(document.querySelector<HTMLTextAreaElement>('[role=dialog] .form textarea')!, "echo snip-$((6*7))");
        document.querySelector<HTMLElement>('[role=dialog] button[type=submit]')?.click();
        await sleep(500);
        const listed = [...document.querySelectorAll("[role=dialog] .item .name")].map((b) => b.textContent);
        const header = document.querySelector("[role=dialog] header .chip")?.textContent;
        const buttons = [...document.querySelectorAll<HTMLElement>("[role=dialog] .item .btn")];
        buttons.find((b) => b.textContent === "Insert")?.click();
        await sleep(400);
        const tail = () => { const b = ws.session(local)?.term.buffer.active; const out: string[] = []; for (let y = Math.max(0, (b?.length ?? 0) - 40); b && y < b.length; y++) { const t = b.getLine(y)?.translateToString(true); if (t) out.push(t); } return out.slice(-4); };
        const inserted = tail().join("").replace(/\s+/g, "").includes("echosnip-$((6*7))"); // the narrow pane wraps the line
        const ranEarly = seen(local, "snip-42");
        const closedAfterUse = !document.querySelector("[role=dialog]");
        run("snippets");
        await tick();
        await sleep(300);
        [...document.querySelectorAll<HTMLElement>("[role=dialog] .item .btn")].find((b) => b.textContent === "Run")?.click();
        for (let i = 0; i < 40 && !seen(local, "snip-42"); i++) await sleep(100);
        out.snippets = { listed, header, inserted, ranEarly, ranAfterRun: seen(local, "snip-42"), closedAfterUse, saved: (await SnippetsService.List())?.length };
        await SnippetsService.Delete("smoke snippet");
      }
      // the split buttons in the top bar
      const leafCount = () => ws.tabs.flatMap((t) => leaves(t.root)).length;
      const leavesBefore = leafCount();
      document.querySelector<HTMLElement>('[aria-label="Split right"]')?.click();
      document.querySelector<HTMLElement>('[aria-label="Split down"]')?.click();
      await new Promise((r) => setTimeout(r, 800));
      out.splitButtons = leafCount() - leavesBefore;
      // dragging: tabs reorder, and a connection dropped on a pane edge splits it (middle: a new tab)
      if (box) {
        const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
        sidebar = true;
        await tick();
        const press = (el: Element, x: number, y: number) => el.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, clientX: x, clientY: y, pointerId: 1 }));
        const dragTo = async (x: number, y: number) => { window.dispatchEvent(new PointerEvent("pointermove", { clientX: x, clientY: y })); await tick(); };
        const release = async (x: number, y: number) => { window.dispatchEvent(new PointerEvent("pointerup", { clientX: x, clientY: y })); await tick(); await sleep(300); };
        await ws.openTab(null);
        const order = () => ws.tabs.map((t) => t.id);
        const before = order();
        const first = document.querySelector<HTMLElement>('[data-tab] .main')!.getBoundingClientRect();
        const lastTab = [...document.querySelectorAll<HTMLElement>("[data-tab]")].at(-1)!.getBoundingClientRect();
        press(document.querySelector('[data-tab] .main')!, first.left + 40, first.top + 10);
        await dragTo(first.left + 60, first.top + 10);
        await dragTo(lastTab.right - 4, first.top + 10);
        const slotShown = !!document.querySelector(".tabs .slot");
        await release(lastTab.right - 4, first.top + 10);
        const after = order();
        out.tabDrag = { before, after, movedFirstToEnd: after.at(-1) === before[0] && after.length === before.length, slotShown, indicatorGone: !document.querySelector(".tabs .slot") };
        const rowBtn = [...document.querySelectorAll<HTMLElement>(".row .name")].find((n) => n.textContent === box)?.closest(".row")?.querySelector<HTMLElement>(".main");
        const paneEl = document.querySelector<HTMLElement>(".tabview.on [data-pane]");
        if (rowBtn && paneEl) {
          const rr = rowBtn.getBoundingClientRect();
          const pr = paneEl.getBoundingClientRect();
          const leafCount = () => ws.tabs.flatMap((t) => leaves(t.root)).length;
          const dropOn = async (x: number, y: number) => {
            press(rowBtn, rr.left + 40, rr.top + 10);
            await dragTo(rr.left + 60, rr.top + 30);
            await dragTo(x, y);
            const hint = document.querySelector(".dropzone")?.textContent ?? null;
            await release(x, y);
            return hint;
          };
          const tab = ws.active!;
          const n0 = leafCount();
          const hintL = await dropOn(pr.left + 20, pr.top + pr.height / 2);
          const lv = leaves(ws.active!.root);
          const left = { hint: hintL, added: leafCount() - n0, firstIsBox: ws.panes[lv[0]]?.target === box, sameTab: ws.active === tab };
          const tabs0 = ws.tabs.length;
          const pr2 = document.querySelector<HTMLElement>(".tabview.on [data-pane]")!.getBoundingClientRect();
          const hintC = await dropOn(pr2.left + pr2.width / 2, pr2.top + pr2.height / 2);
          const centerTarget = ws.panes[ws.focusedPane]?.target;
          out.paneDrop = { left, center: { hint: hintC, newTab: ws.tabs.length - tabs0, target: centerTarget } };
          // the remote pane just opened waits at ssh's host-key question: blocks are off there, so the opt-in is offered
          await sleep(3200);
          out.blocksOffer = {
            statusButton: [...document.querySelectorAll("footer button")].some((b) => b.textContent?.trim() === "Enable command blocks"),
            pill: document.querySelector(".tabview.on .hint strong")?.textContent ?? null,
          };
          // double-click opens a new tab; a single click does not
          ws.select(ws.tabs.find((t) => leaves(t.root).every((id) => ws.panes[id]?.target === null))!.id); // a tab with only local shells
          await tick();
          const awayFrom = ws.panes[ws.focusedPane]?.target;
          const tabsA = ws.tabs.length;
          rowBtn.dispatchEvent(new MouseEvent("click", { bubbles: true, detail: 1 }));
          await sleep(200);
          const jumped = ws.panes[ws.focusedPane]?.target;
          const afterSingle = ws.tabs.length;
          rowBtn.dispatchEvent(new MouseEvent("dblclick", { bubbles: true, detail: 2 }));
          await sleep(400);
          out.openByClick = { awayFrom, jumpedTo: jumped, single: afterSingle - tabsA, double: ws.tabs.length - afterSingle, tip: !!document.querySelector(".tip") };
          // dragging a pane by its header moves it, and its session keeps running
          ws.select(tab.id);
          await tick();
          await sleep(300);
          const ids = leaves(ws.active!.root);
          const mover = ids.at(-1)!;
          const dest = ids[0];
          const hdr = document.querySelector<HTMLElement>(`[data-pane="${mover}"] header`);
          const destEl = document.querySelector<HTMLElement>(`[data-pane="${dest}"]`);
          if (hdr && destEl) {
            const h = hdr.getBoundingClientRect();
            const d = destEl.getBoundingClientRect();
            press(hdr, h.left + 60, h.top + 10);
            await dragTo(h.left + 80, h.top + 30);
            await dragTo(d.left + 20, d.top + d.height / 2);
            const hint = document.querySelector(".dropzone")?.textContent ?? null;
            await release(d.left + 20, d.top + d.height / 2);
            const now = leaves(ws.active!.root);
            ws.session(mover)?.ws.send(enc.encode("echo moved-$((6*7))\n"));
            for (let i = 0; i < 40 && !seen(mover, "moved-42"); i++) await sleep(100);
            const el = document.querySelector<HTMLElement>(`[data-pane="${mover}"] .xterm-screen`)?.getBoundingClientRect();
            out.paneMove = { hint, before: ids, after: now, movedFirst: now[0] === mover, sameSet: [...now].sort().join() === [...ids].sort().join(), stillRuns: seen(mover, "moved-42"), size: el ? [Math.round(el.width), Math.round(el.height)] : null };
          }
        }
      }
      // across tabs: a pane dragged onto a tab or the + button moves there, and a tab dragged onto a pane merges into it
      if (box) {
        const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
        const press = (el: Element, x: number, y: number) => el.dispatchEvent(new PointerEvent("pointerdown", { bubbles: true, button: 0, clientX: x, clientY: y, pointerId: 1 }));
        const mv = async (x: number, y: number) => { window.dispatchEvent(new PointerEvent("pointermove", { clientX: x, clientY: y })); await tick(); };
        const up = async (x: number, y: number) => { window.dispatchEvent(new PointerEvent("pointerup", { clientX: x, clientY: y })); await tick(); await sleep(400); };
        const center = (el: Element) => { const r = el.getBoundingClientRect(); return [r.left + r.width / 2, r.top + r.height / 2] as const; };
        const tabOf = (id: string) => ws.tabs.find((t) => leaves(t.root).includes(id));
        // start clean: earlier steps leave many tabs, and the ones past the edge of the strip cannot be hit
        for (const t of [...ws.tabs]) ws.closeTab(t.id);
        await tick();
        // a tab with two panes, and a second tab
        await ws.openTab(null);
        await ws.split("row");
        const two = ws.active!;
        const [pa, pb] = leaves(two.root);
        await ws.openTab(null);
        const other = ws.active!;
        ws.select(two.id);
        await tick();
        await sleep(300);
        // pane -> another tab
        const hdr = document.querySelector<HTMLElement>(`[data-pane="${pb}"] header`)!;
        const [hx, hy] = center(hdr);
        press(hdr, hx, hy);
        await mv(hx + 20, hy + 20);
        const [tx, ty] = center(document.querySelector(`[data-tab="${other.id}"]`)!);
        await mv(tx, ty);
        const highlighted = !!document.querySelector(`[data-tab="${other.id}"].drop`);
        await up(tx, ty);
        const toTab = { highlighted, movedTo: tabOf(pb)?.id === other.id, active: ws.active?.id === other.id, sourceKept: tabOf(pa)?.id === two.id, runs: false };
        ws.session(pb)?.ws.send(enc.encode("echo crosstab-$((6*7))\n"));
        for (let i = 0; i < 40 && !seen(pb, "crosstab-42"); i++) await sleep(100);
        toTab.runs = seen(pb, "crosstab-42");
        // pane -> the + button (a new tab)
        ws.select(other.id);
        await tick();
        await sleep(300);
        const h2 = document.querySelector<HTMLElement>(`[data-pane="${pb}"] header`)!;
        const [h2x, h2y] = center(h2);
        const tabsBefore = ws.tabs.length;
        press(h2, h2x, h2y);
        await mv(h2x + 20, h2y + 20);
        const [px, py] = center(document.querySelector("[data-newtab]")!);
        await mv(px, py);
        await up(px, py);
        const toNew = { newTabs: ws.tabs.length - tabsBefore, alone: leaves(tabOf(pb)!.root).length === 1 && tabOf(pb)?.id !== other.id };
        // tab -> a pane of the active tab: the dragged tab's panes land beside it
        const active = ws.active!;
        const victim = ws.tabs.find((t) => t !== active)!;
        const victimPanes = leaves(victim.root);
        const targetPane = leaves(active.root)[0];
        const tabsN = ws.tabs.length;
        const tabBtn = document.querySelector(`[data-tab="${victim.id}"] .main`)!;
        const [bx, by] = center(tabBtn);
        press(tabBtn, bx, by);
        await mv(bx + 30, by);
        const tp = document.querySelector<HTMLElement>(`.tabview.on [data-pane="${targetPane}"]`)!.getBoundingClientRect();
        await mv(tp.right - 20, tp.top + tp.height / 2);
        const hint = document.querySelector(".dropzone")?.textContent ?? null;
        await up(tp.right - 20, tp.top + tp.height / 2);
        out.crossTab = { toTab, toNew, merge: { hint, tabsRemoved: tabsN - ws.tabs.length, together: victimPanes.every((id) => tabOf(id)?.id === active.id), activeIsDest: ws.active?.id === active.id } };
      }
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
        // automatic command blocks: after the login authenticates and goes quiet, the setup line is typed
        const autoBefore = settings.autoBlocks;
        settings.autoBlocks = true;
        await ws.openTab(box);
        const auto = ws.focusedPane;
        for (let i = 0; i < 60 && !seen(auto, "yes/no"); i++) await sleep(100);
        const typedBeforeLogin = ws.panes[auto]?.blocks || ws.session(auto)?.authed === true;
        ws.session(auto)?.ws.send(enc.encode("yes\n"));
        for (let i = 0; i < 150 && !ws.panes[auto]?.blocks; i++) await sleep(100);
        out.autoBlocks = { sawHostKeyQuestion: seen(auto, "yes/no"), authedBeforeAnswer: typedBeforeLogin, authed: ws.session(auto)?.authed, authBy: ws.session(auto)?.authBy, blocks: ws.panes[auto]?.blocks, pill: !!document.querySelector(".tabview.on .hint") };
        settings.autoBlocks = autoBefore;
      }
    } catch (e) {
      out.error = String(e);
    }
    await TerminalService.ReportBench(JSON.stringify(out));
  }
</script>

<svelte:window onkeydown={keydown} />

<div class="app">
  <TabBar custom={customChrome} connected={!!ws.panes[ws.focusedPane]?.target} onsidebar={() => (sidebar = !sidebar)} onsearch={() => (sidebar = true)} onpalette={() => (palette = true)} onsplit={(d) => ws.split(d)} onsettings={() => openSettings()} ontools={() => { toolsTab = "health"; toolsOpen = !toolsOpen; }} onfiles={openFiles} onforwards={openForwards} onsnippets={() => (snippetsOpen = !snippetsOpen)} />
  <div class="body">
    {#if sidebar}
      <Sidebar {profiles} {folders} {open} current={ws.panes[ws.focusedPane]?.target ?? ""} onconnect={(p) => connect(p)} onnew={newConnection} onedit={edit} ontools={(t) => { toolsTab = t; toolsOpen = true; }} onchanged={refresh} />
    {/if}
    <main>
      {#if banner}<div class="notice bad banner" role="alert"><span>{banner}</span><button type="button" class="icon-btn dense" aria-label="Dismiss" onclick={() => (banner = "")}><Icon name="x" size={14} /></button></div>{/if}
      <div class="stage" bind:this={benchHost}>
        {#each ws.tabs as t (t.id)}
          <div class="tabview" class:on={t.id === ws.activeId}>
            <PaneView node={t.root} tabId={t.id} multi={leaves(t.root).length > 1} {profiles} />
          </div>
        {:else}
          <div class="welcome">
            <Logo size={56} />
            <h1>Relay</h1>
            <p>{profiles.length ? "Open a connection from the list, or start a local shell." : "Add your first connection, or bring in the hosts you already have."}</p>
            <div class="actions">
              {#if profiles.length === 0}
                <button type="button" class="btn primary" onclick={() => newConnection("")}>New connection</button>
                <button type="button" class="btn" onclick={() => { toolsTab = "import"; toolsOpen = true; }}>Import from ~/.ssh/config</button>
              {/if}
              <button type="button" class="btn" onclick={() => ws.openTab(null)}>Local shell <span class="kbd">{mac ? "⌘T" : "Ctrl+Shift+T"}</span></button>
            </div>
            <p class="keys"><span class="kbd">{mac ? "⌘K" : "Ctrl+K"}</span> quick open</p>
          </div>
        {/each}
      </div>
    </main>
  </div>
  <StatusBar {profiles} themeName={settings.label} onvault={() => openSettings("vault")} />
</div>

{#if snippetsOpen}{#await import("./components/SnippetsPanel.svelte") then { default: SnippetsPanel }}<SnippetsPanel into={ws.panes[ws.focusedPane] ? (ws.panes[ws.focusedPane].target ?? "Local shell") : ""} onuse={(c, run) => ws.typeSnippet(c, run)} onclose={() => (snippetsOpen = false)} />{/await}{/if}
{#if forwardsTarget}{#await import("./components/ForwardsPanel.svelte") then { default: ForwardsPanel }}<ForwardsPanel target={forwardsTarget} onchanged={refresh} onclose={() => (forwardsTarget = "")} onopen={() => { const t = forwardsTarget; forwardsTarget = ""; ws.openTab(t); }} />{/await}{/if}
{#if filesTarget}{#await import("./components/FilesPanel.svelte") then { default: FilesPanel }}<FilesPanel target={filesTarget} onclose={() => (filesTarget = "")} />{/await}{/if}
{#if toolsOpen}{#await import("./components/ToolsPanel.svelte") then { default: ToolsPanel }}<ToolsPanel initial={toolsTab} onclose={() => (toolsOpen = false)} onchanged={refresh} ongenerate={generateKey} />{/await}{/if}
{#if ws.blockMenu}<BlockMenu block={ws.blockMenu.block} rect={ws.blockMenu.rect} oncopy={copied} onclose={() => (ws.blockMenu = null)} />{/if}
{#if toast}<div class="toast" role="status"><Icon name="check" />{toast}</div>{/if}
{#if settingsOpen}{#await import("./components/SettingsPanel.svelte") then { default: SettingsPanel }}<SettingsPanel section={settingsSection} onclose={() => { settingsOpen = false; settingsSection = undefined; }} />{/await}{/if}
<DragGhost />
{#if palette}{#await import("./components/Palette.svelte") then { default: Palette }}<Palette {profiles} active={open} {actions} onconnect={connect} onaction={run} onclose={() => (palette = false)} />{/await}{/if}
{#if unlockConnect && VaultPanel}
  <VaultPanel
    mode="unlock-connect"
    connName={unlocking?.profile.name}
    onsuccess={async () => {
      const { profile, split } = unlockConnect!;
      unlockConnect = null;
      await doConnect(profile, split);
    }}
    onskip={async () => {
      const { profile, split } = unlockConnect!;
      unlockConnect = null;
      await doConnect(profile, split);
    }}
    onclose={() => {
      unlockConnect = null;
    }}
  />
{/if}
{#if form && ProfileForm}
  <ProfileForm initial={formShown!.initial} oldName={formShown!.oldName} {profiles} {folders} onsave={save} ondelete={() => (confirmDelete = profiles.find((p) => p.name === form!.oldName) ?? null)} oncancel={() => (form = null)} />
{/if}
{#if confirmDelete}
  <Modal title="Delete connection" size="sm" role="alertdialog" onclose={() => (confirmDelete = null)}>
    <p class="confirm-text">Delete <strong>{deleting?.name}</strong>? This removes it from connections.toml and your SSH config.</p>
    {#snippet footer()}
      <button type="button" class="btn" data-autofocus onclick={() => (confirmDelete = null)}>Cancel</button>
      <button type="button" class="btn danger" onclick={() => remove(confirmDelete!)}>Delete</button>
    {/snippet}
  </Modal>
{/if}

<style>
  .app { display: flex; flex-direction: column; height: 100%; }
  .body { flex: 1; min-height: 0; display: flex; }
  main { flex: 1; min-width: 0; display: flex; flex-direction: column; background: var(--term); }
  .stage { flex: 1; min-height: 0; position: relative; }
  .tabview { position: absolute; inset: 0; display: none; }
  .tabview.on { display: flex; }
  .welcome { position: absolute; inset: 0; display: grid; place-content: center; justify-items: center; gap: var(--s2); text-align: center; color: var(--muted); }
  .welcome h1 { color: var(--strong); margin: var(--s3) 0 0; font-size: var(--t-xl); font-weight: 600; }
  .welcome p { margin: 0; max-width: 40ch; }
  .welcome .actions { display: flex; flex-wrap: wrap; justify-content: center; gap: var(--s2); margin-top: var(--s4); }
  .welcome .keys { margin-top: var(--s2); font-size: var(--t-sm); color: var(--dim); }
  .banner { margin: var(--s2) var(--s3) 0; justify-content: space-between; align-items: center; }
  .confirm-text { margin: 0; line-height: 1.5; }
  .toast { position: fixed; left: 50%; bottom: 44px; transform: translateX(-50%); z-index: var(--z-toast); display: flex; align-items: center; gap: var(--s2); padding: var(--s2) var(--s4); background: var(--float); border: 1px solid var(--floatline); border-radius: var(--r-md); box-shadow: var(--e-pop); font-size: var(--t-md); color: var(--text); }
  .toast :global(svg) { color: var(--ok); }
</style>
