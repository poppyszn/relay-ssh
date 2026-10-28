///<reference types="svelte" />
;
import { onMount } from "svelte";
import { openSession, type Session } from "./terminal";
import { runBench } from "./bench";
import { blankProfile } from "./lib";
import Sidebar from "./Sidebar.svelte";
import Palette from "./Palette.svelte";
import ProfileForm from "./ProfileForm.svelte";
import { ProfilesService, TerminalService, type Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
function $$render() {

  
  
  
  
  
  
  
  

  let host: HTMLElement;
  let profiles = $state<Profile[]>([]);
  let active = $state<Profile | null>(null);
  let session: Session | undefined;
  let started = $state(false);
  let ended = $state(false);
  let banner = $state("");
  let palette = $state(false);
  let form = $state<{ initial: Profile; oldName: string } | null>(null);
  let confirmDelete = $state<Profile | null>(null);

  const folders = $derived([...new Set(profiles.map((p) => p.folder).filter(Boolean))].sort());

  async function refresh() {
    try {
      profiles = (await ProfilesService.List()) ?? [];
    } catch (e) {
      banner = String(e);
    }
  }

  async function connect(p: Profile | null) {
    palette = false;
    session?.dispose();
    host.replaceChildren();
    active = p;
    ended = false;
    banner = "";
    try {
      session = await openSession(host, p ? p.name : "local");
      session.ws.addEventListener("close", () => (ended = true));
      session.term.focus();
      started = true;
    } catch (e) {
      banner = String(e);
    }
  }

  async function save(prof: Profile) {
    await ProfilesService.Save(form!.oldName, prof);
    if (active && active.name === form!.oldName) active = { ...prof };
    form = null;
    await refresh();
  }

  async function remove(p: Profile) {
    confirmDelete = null;
    try {
      await ProfilesService.Delete(p.name);
      if (active?.name === p.name) active = null;
      await refresh();
    } catch (e) {
      banner = String(e);
    }
  }

  function keydown(e: KeyboardEvent) {
    if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      palette = !palette;
    } else if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "n") {
      e.preventDefault();
      form = { initial: blankProfile(), oldName: "" };
    }
  }

  // RELAY_BENCH=smoke: drive the same path as the UI (load profiles, press
  // "Local shell", run a command) and report what happened, so a build can be
  // verified without clicking.
  async function smoke() {
    const out: Record<string, unknown> = { origin: location.origin };
    try {
      await refresh();
      out.profiles = profiles.length;
      await connect(null);
      out.banner = banner;
      out.connected = session?.ws.readyState === WebSocket.OPEN;
      session?.ws.send(new TextEncoder().encode("echo smoke-$((6*7))\n"));
      for (let i = 0; i < 50 && !out.found; i++) {
        await new Promise((r) => setTimeout(r, 100));
        const b = session?.term.buffer.active;
        for (let y = 0; b && y < b.length; y++) if (b.getLine(y)?.translateToString().includes("smoke-42")) out.found = true;
      }
    } catch (e) {
      out.error = String(e);
    }
    await TerminalService.ReportBench(JSON.stringify(out));
  }

  onMount(async () => {
    const mode = await TerminalService.BenchMode();
    if (mode === "smoke") return smoke();
    if (mode) return runBench(host, mode);
    await refresh();
  });
;
async () => {

 { svelteHTML.createElement("svelte:window", {  "onkeydown":keydown,});}

 { svelteHTML.createElement("div", { "class":`shell`,});
   { const $$_rabediS1C = __sveltets_2_ensureComponent(Sidebar); new $$_rabediS1C({ target: __sveltets_2_any(), props: {              profiles,"active":active?.name ?? "","onconnect":connect,"onnew":() => (form = { initial: blankProfile(), oldName: "" }),"onedit":(p) => (form = { initial: p, oldName: p.name }),"ondelete":(p) => (confirmDelete = p),"onpalette":() => (palette = true),}});}
   { svelteHTML.createElement("main", {});
     { svelteHTML.createElement("header", {});
      if(active){
         { svelteHTML.createElement("strong", {});active.name; } { svelteHTML.createElement("span", { "class":`sub`,});active.user; active.host; }
          for(let t of __sveltets_2_ensureArray(active.tags)){ { svelteHTML.createElement("span", { "class":`chip`,});t; }}
      }else{
         { svelteHTML.createElement("strong", {});  }
      }
       { svelteHTML.createElement("span", { "class":`spacer`,}); }
      if(ended){ { svelteHTML.createElement("span", { "class":`ended`,});  } { svelteHTML.createElement("button", {   "class":`btn`,"onclick":() => connect(active),});  }}
       { svelteHTML.createElement("button", {   "class":`btn`,"onclick":() => connect(null),});  }
     }
    if(banner){ { svelteHTML.createElement("p", {   "class":`banner`,"role":`alert`,});banner; }}
     { const $$_div2 = svelteHTML.createElement("div", {  "class":`term`,});host = $$_div2; }
    if(!started && !banner){
       { svelteHTML.createElement("div", { "class":`welcome`,});
         { svelteHTML.createElement("h1", {});  }
         { svelteHTML.createElement("p", {});         { svelteHTML.createElement("kbd", {});  }     }
       }
    }
   }
 }

if(palette){ { const $$_ettelaP0C = __sveltets_2_ensureComponent(Palette); new $$_ettelaP0C({ target: __sveltets_2_any(), props: {     profiles,"onpick":connect,"onclose":() => (palette = false),}});}}
if(form){
   { const $$_mroFeliforP0C = __sveltets_2_ensureComponent(ProfileForm); new $$_mroFeliforP0C({ target: __sveltets_2_any(), props: {           "initial":form.initial,"isNew":form.oldName === "","others":profiles.map((p) => p.name).filter((n) => n !== form!.oldName),folders,"onsave":save,"oncancel":() => (form = null),}});}
}
if(confirmDelete){
   { svelteHTML.createElement("div", {     "class":`scrim`,"role":`presentation`,"onkeydown":(e) => e.key === "Escape" && (confirmDelete = null),});
     { svelteHTML.createElement("div", {     "class":`confirm`,"role":`alertdialog`,"aria-label":`Delete connection`,});
       { svelteHTML.createElement("p", {});  { svelteHTML.createElement("strong", {});confirmDelete.name; }          }
       { svelteHTML.createElement("div", { "class":`buttons`,});
         { svelteHTML.createElement("button", {   "class":`btn`,"onclick":() => (confirmDelete = null),});  }
         { svelteHTML.createElement("button", {   "class":`btn danger`,"onclick":() => remove(confirmDelete!),});  }
       }
     }
   }
}


};
return { props: {} as Record<string, never>, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const App__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type App__SvelteComponent_ = ReturnType<typeof App__SvelteComponent_>;
/*Ωignore_endΩ*/export default App__SvelteComponent_;