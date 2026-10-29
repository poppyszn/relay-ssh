///<reference types="svelte" />
;
import { onMount, tick } from "svelte";
import { Events } from "@wailsio/runtime";
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
import SettingsPanel from "./SettingsPanel.svelte";
import ToolsPanel from "./ToolsPanel.svelte";
import FilesPanel from "./FilesPanel.svelte";
import ForwardsPanel from "./ForwardsPanel.svelte";
import { settings } from "./settings.svelte";
import { FilesService, ForwardsService, ProfilesService, TerminalService, ToolsService, type Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
function $$render() {

  
  
  
  
  
  
  
  
  
  
  
  
  
  
  
  
  
  

  const mac = navigator.userAgent.includes("Mac");
  let profiles = $state<Profile[]>([]);
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

  const folders = $derived([...new Set(profiles.map((p) => p.folder).filter(Boolean))].sort());
  const open = $derived(new Set(ws.tabs.flatMap((t) => leaves(t.root).map((id) => ws.panes[id]?.target ?? "")).filter(Boolean)));
  const actions = [
    { id: "new-connection", label: "New connection", hint: mac ? "⌘N" : "Ctrl+Shift+N" },
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
      profiles = (await ProfilesService.List()) ?? [];
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
      case "new-connection": return (form = { initial: blankProfile(), oldName: "" });
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
      case "tools": return (toolsOpen = !toolsOpen);
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
    const map: Record<string, string> = { T: "new-tab", N: "new-connection", K: "quick-open", D: "split-right", W: "close", B: "toggle-sidebar", F: "find", ",": "settings", O: "files", P: "forwards" };
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
    } catch (e) {
      out.error = String(e);
    }
    await TerminalService.ReportBench(JSON.stringify(out));
  }
;
async () => {

 { svelteHTML.createElement("svelte:window", {  "onkeydown":keydown,});}

 { svelteHTML.createElement("div", { "class":`app`,});
   { const $$_raBbaT1C = __sveltets_2_ensureComponent(TabBar); new $$_raBbaT1C({ target: __sveltets_2_any(), props: {            "custom":customChrome,"onsidebar":() => (sidebar = !sidebar),"onsettings":() => (settingsOpen = !settingsOpen),"ontools":() => (toolsOpen = !toolsOpen),"onfiles":openFiles,"onforwards":openForwards,}});}
   { svelteHTML.createElement("div", { "class":`body`,});
    if(sidebar){
       { const $$_rabediS2C = __sveltets_2_ensureComponent(Sidebar); new $$_rabediS2C({ target: __sveltets_2_any(), props: {           profiles,"active":open,"onconnect":(p) => connect(p),"onnew":() => run("new-connection"),"onedit":edit,"onpalette":() => (palette = true),}});}
    }
     { svelteHTML.createElement("main", {});
      if(banner){ { svelteHTML.createElement("p", {   "class":`banner`,"role":`alert`,});banner; { svelteHTML.createElement("button", {   "aria-label":`Dismiss`,"onclick":() => (banner = ""),});  } }}
       { const $$_div3 = svelteHTML.createElement("div", {  "class":`stage`,});benchHost = $$_div3;
           for(let t of __sveltets_2_ensureArray(ws.tabs)){t.id;
           { svelteHTML.createElement("div", {  "class":`tabview`,});t.id === ws.activeId;
             { const $$_weiVenaP5C = __sveltets_2_ensureComponent(PaneView); new $$_weiVenaP5C({ target: __sveltets_2_any(), props: {       "node":t.root,"tabId":t.id,"multi":leaves(t.root).length > 1,profiles,}});}
           }
        }
           { svelteHTML.createElement("div", { "class":`welcome`,}); { svelteHTML.createElement("h1", {});  } { svelteHTML.createElement("p", {});         { svelteHTML.createElement("span", { "class":`kbd`,});mac ? "⌘T" : "Ctrl+Shift+T"; }     } }
        
       }
     }
   }
   { const $$_raBsutatS1C = __sveltets_2_ensureComponent(StatusBar); new $$_raBsutatS1C({ target: __sveltets_2_any(), props: {   profiles,"themeName":settings.themeDef.name,}});}
 }

if(forwardsTarget){ { const $$_lenaPsdrawroF0C = __sveltets_2_ensureComponent(ForwardsPanel); new $$_lenaPsdrawroF0C({ target: __sveltets_2_any(), props: {        "target":forwardsTarget,"onchanged":refresh,"onclose":() => (forwardsTarget = ""),"onopen":() => { const t = forwardsTarget; forwardsTarget = ""; ws.openTab(t); },}});}}
if(filesTarget){ { const $$_lenaPseliF0C = __sveltets_2_ensureComponent(FilesPanel); new $$_lenaPseliF0C({ target: __sveltets_2_any(), props: {    "target":filesTarget,"onclose":() => (filesTarget = ""),}});}}
if(toolsOpen){ { const $$_lenaPslooT0C = __sveltets_2_ensureComponent(ToolsPanel); new $$_lenaPslooT0C({ target: __sveltets_2_any(), props: {      "onclose":() => (toolsOpen = false),"onchanged":refresh,"ongenerate":generateKey,}});}}
if(settingsOpen){ { const $$_lenaPsgnitteS0C = __sveltets_2_ensureComponent(SettingsPanel); new $$_lenaPsgnitteS0C({ target: __sveltets_2_any(), props: {  "onclose":() => (settingsOpen = false),}});}}
if(palette){ { const $$_ettelaP0C = __sveltets_2_ensureComponent(Palette); new $$_ettelaP0C({ target: __sveltets_2_any(), props: {          profiles,"active":open,actions,"onconnect":connect,"onaction":run,"onclose":() => (palette = false),}});}}
if(form){
   { const $$_mroFeliforP0C = __sveltets_2_ensureComponent(ProfileForm); new $$_mroFeliforP0C({ target: __sveltets_2_any(), props: {            "initial":form.initial,"oldName":form.oldName,profiles,folders,"onsave":save,"ondelete":() => (confirmDelete = profiles.find((p) => p.name === form!.oldName) ?? null),"oncancel":() => (form = null),}});}
}
if(confirmDelete){
   { svelteHTML.createElement("div", {     "class":`scrim`,"role":`presentation`,"onkeydown":(e) => e.key === "Escape" && (confirmDelete = null),});
     { svelteHTML.createElement("div", {     "class":`confirm`,"role":`alertdialog`,"aria-label":`Delete connection`,});
       { svelteHTML.createElement("p", {});  { svelteHTML.createElement("strong", {});confirmDelete.name; }          }
       { svelteHTML.createElement("div", { "class":`buttons`,}); { svelteHTML.createElement("button", {   "class":`btn`,"onclick":() => (confirmDelete = null),});  } { svelteHTML.createElement("button", {   "class":`btn primary danger-fill`,"onclick":() => remove(confirmDelete!),});  } }
     }
   }
}


};
return { props: {} as Record<string, never>, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const App__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type App__SvelteComponent_ = ReturnType<typeof App__SvelteComponent_>;
/*Ωignore_endΩ*/export default App__SvelteComponent_;