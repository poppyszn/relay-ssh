///<reference types="svelte" />
;
import { onMount } from "svelte";
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
import { ProfilesService, TerminalService, type Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
function $$render() {

  
  
  
  
  
  
  
  
  
  
  
  
  

  const mac = navigator.userAgent.includes("Mac");
  let profiles = $state<Profile[]>([]);
  let banner = $state("");
  let palette = $state(false);
  let sidebar = $state(true);
  let form = $state<{ initial: Profile; oldName: string } | null>(null);
  let confirmDelete = $state<Profile | null>(null);
  let benchHost = $state<HTMLElement>();

  const folders = $derived([...new Set(profiles.map((p) => p.folder).filter(Boolean))].sort());
  const open = $derived(new Set(ws.tabs.flatMap((t) => leaves(t.root).map((id) => ws.panes[id]?.target ?? "")).filter(Boolean)));
  const actions = [
    { id: "new-connection", label: "New connection", hint: mac ? "⌘N" : "Ctrl+Shift+N" },
    { id: "new-tab", label: "New local tab", hint: mac ? "⌘T" : "Ctrl+Shift+T" },
    { id: "split-right", label: "Split pane right", hint: mac ? "⌘D" : "Ctrl+Shift+D" },
    { id: "split-down", label: "Split pane down", hint: mac ? "⇧⌘D" : "Ctrl+Alt+D" },
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
      case "enable-blocks": return ws.focusedPane && ws.enableBlocks(ws.focusedPane);
    }
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
    const map: Record<string, string> = { T: "new-tab", N: "new-connection", K: "quick-open", D: "split-right", W: "close", B: "toggle-sidebar" };
    const id = map[e.key.toUpperCase()];
    if (id) { e.preventDefault(); run(id); }
  }

  onMount(() => {
    const off = Events.On("menu", (e: { data: string }) => run(e.data));
    (async () => {
      const mode = await TerminalService.BenchMode();
      if (mode === "smoke") return smoke();
      if (mode) return runBench(benchHost!, mode);
      await refresh();
      await ws.openTab(null);
    })();
    return off;
  });

  // RELAY_BENCH=smoke: drive the real UI path (profiles, open a local tab,
  // split it, run a command in each pane) and report, so a build can be
  // verified without clicking.
  async function smoke() {
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
      out.viewportBg = getComputedStyle(document.querySelector(".tabview.on .xterm-viewport")!).backgroundColor;
      out.fit = { rows: term?.rows, screenH: screen?.clientHeight, innerH: pr ? Math.round(pr.height - 32) : null, cellH: screen && term ? +(screen.clientHeight / term.rows).toFixed(2) : null, dpr: devicePixelRatio, font: term?.options.fontSize, lh: term?.options.lineHeight };
      out.blockEdges = [...document.querySelectorAll<HTMLElement>(".tabview.on .relay-block")].map((e) => {
        const r = e.getBoundingClientRect();
        return { left: Math.round(r.left - (pr?.left ?? 0)), right: Math.round((pr?.right ?? 0) - r.right) };
      });
      out.okBlocks = overlaid(".relay-block.ok");
      out.errBlocks = overlaid(".relay-block.err");
      out.badges = [...document.querySelectorAll(".relay-badge")].map((b) => b.textContent);
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
   { const $$_raBbaT1C = __sveltets_2_ensureComponent(TabBar); new $$_raBbaT1C({ target: __sveltets_2_any(), props: {    "onsidebar":() => (sidebar = !sidebar),"onsettings":() => (banner = "Settings arrive with themes and fonts."),}});}
   { svelteHTML.createElement("div", { "class":`body`,});
    if(sidebar){
       { const $$_rabediS2C = __sveltets_2_ensureComponent(Sidebar); new $$_rabediS2C({ target: __sveltets_2_any(), props: {           profiles,"active":open,"onconnect":(p) => connect(p),"onnew":() => run("new-connection"),"onedit":(p) => (form = { initial: p, oldName: p.name }),"onpalette":() => (palette = true),}});}
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
   { const $$_raBsutatS1C = __sveltets_2_ensureComponent(StatusBar); new $$_raBsutatS1C({ target: __sveltets_2_any(), props: {   profiles,"themeName":`Midnight`,}});}
 }

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