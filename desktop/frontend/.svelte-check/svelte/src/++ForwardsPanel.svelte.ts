///<reference types="svelte" />
;
import ForwardDiagram from "./ForwardDiagram.svelte";
import { ForwardsService, ProfilesService, type ForwardInfo } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";

;type $$ComponentProps =  { target: string; onclose: () => void; onopen: () => void; onchanged: () => void };function $$render() {

  
  

  let { target, onclose, onopen, onchanged }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  let connected = $state(false);
  let message = $state("");
  let forwards = $state<ForwardInfo[]>([]);
  let loaded = $state(false);
  let error = $state("");
  let note = $state("");
  let busy = $state(false);

  let kind = $state<"local" | "remote" | "dynamic">("local");
  let listen = $state("");
  let dest = $state("");
  // Clicking a row shows that forward in the diagram; typing in the form takes over again.
  let viewing = $state<ForwardInfo | null>(null);
  const shown = $derived(viewing ?? { type: kind, listen, target: dest });

  const label = { local: "Local", remote: "Remote", dynamic: "SOCKS" } as const;
  const msg = (e: unknown) => (e as Error)?.message ?? String(e);

  async function refresh() {
    try {
      const s = await ForwardsService.Status(target);
      connected = s.connected; message = s.message; forwards = s.forwards ?? [];
    } catch (e) { error = msg(e); } finally { loaded = true; }
  }
  async function run(fn: () => Promise<void>, ok: string) {
    busy = true; error = ""; note = "";
    try { await fn(); note = ok; await refresh(); } catch (e) { error = msg(e); } finally { busy = false; }
  }
  const add = () => run(async () => { await ForwardsService.Add(target, kind, listen.trim(), dest.trim()); listen = ""; dest = ""; }, "Forward added.");
  const stop = (f: ForwardInfo) => run(() => ForwardsService.Cancel(target, f.type, f.listen, f.target), "Forward stopped.");
  const start = (f: ForwardInfo) => run(() => ForwardsService.Add(target, f.type, f.listen, f.target), "Forward started.");
  // Makes a runtime forward part of the saved connection, so it starts with every login.
  const save = (f: ForwardInfo) => run(async () => {
    const p = (await ProfilesService.List())?.find((x) => x.name === target);
    if (!p) throw new Error("Connection not found.");
    p.tunnels = [...(p.tunnels ?? []), { type: f.type, listen: f.listen, target: f.target }];
    await ProfilesService.Save(target, p);
    onchanged();
  }, "Saved to the connection. It will start automatically next time.");

  $effect(() => {
    refresh();
    const t = setInterval(refresh, 3000);
    return () => clearInterval(t);
  });
;
async () => {

 { svelteHTML.createElement("div", {       "class":`scrim`,"role":`presentation`,"onclick":onclose,"onkeydown":(e) => e.key === "Escape" && onclose(),});
   { svelteHTML.createElement("div", {         "class":`panel`,"role":`dialog`,"aria-label":`Port forwards`,"tabindex":-1,"onclick":(e) => e.stopPropagation(),});
     { svelteHTML.createElement("header", {});
       { svelteHTML.createElement("h2", {});   { svelteHTML.createElement("span", { "class":`host mono`,});target; }
        if(loaded){ { svelteHTML.createElement("span", { "class":`state`,}); { svelteHTML.createElement("span", {  "class":`dot`,});connected; }connected ? "Connected" : "Not connected"; }} }
       { svelteHTML.createElement("button", {     "class":`x`,"aria-label":`Close`,"onclick":onclose,});  }
     }

    if(loaded && !connected){
       { svelteHTML.createElement("div", { "class":`empty`,});
         { svelteHTML.createElement("p", {});message || "This connection is not open."; }
         { svelteHTML.createElement("p", { "class":`dim`,});             }
         { svelteHTML.createElement("button", {   "class":`btn primary`,"onclick":onopen,}); target;    }
       }
    }else{
       { svelteHTML.createElement("div", { "class":`body`,});
        if(forwards.length){
           { svelteHTML.createElement("div", { "class":`row head`,}); { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {}); } }
             for(let f of __sveltets_2_ensureArray(forwards)){f.type + f.listen + f.target;
             { svelteHTML.createElement("div", {          "class":`row pick`,"role":`row`,"tabindex":0,"onclick":() => (viewing = f),"onkeydown":(e) => e.key === "Enter" && (viewing = f),});viewing === f;
               { svelteHTML.createElement("span", {}); { svelteHTML.createElement("span", { "class":`chip`,});label[f.type as keyof typeof label]; } }
               { svelteHTML.createElement("span", { "class":`mono`,}); { svelteHTML.createElement("span", {  "class":`dot`,});f.listening; } f.listen; }
               { svelteHTML.createElement("span", { "class":`mono`,});f.type === "dynamic" ? "SOCKS proxy" : f.target; }
               { svelteHTML.createElement("span", { "class":`dim`,});f.source === "profile" ? "Saved" : "Added now"; }
               { svelteHTML.createElement("span", { "class":`acts`,});
                if(f.source === "runtime"){ { svelteHTML.createElement("button", {       "class":`btn sm`,"onclick":() => save(f),"disabled":busy,"title":`Keep this forward for every login`,});  }}
                if(f.listening){ { svelteHTML.createElement("button", {     "class":`btn sm`,"onclick":() => stop(f),"disabled":busy,});  }}else{ { svelteHTML.createElement("button", {     "class":`btn sm`,"onclick":() => start(f),"disabled":busy,});  }}
               }
             }
          }
        } else if (loaded){
           { svelteHTML.createElement("p", { "class":`none`,});      }
        }
       }

       { svelteHTML.createElement("form", {   "class":`add`,"onsubmit":(e) => { e.preventDefault(); add(); },});
         { const $$_margaiDdrawroF3C = __sveltets_2_ensureComponent(ForwardDiagram); new $$_margaiDdrawroF3C({ target: __sveltets_2_any(), props: {        "kind":shown.type as "local" | "remote" | "dynamic","listen":shown.listen,"dest":shown.target,"server":target,}});}
         { svelteHTML.createElement("div", { "class":`label`,});   }
         { svelteHTML.createElement("div", { "class":`fields`,});
           { svelteHTML.createElement("select", {     "bind:value":kind,"aria-label":`Type`,"onchange":() => (viewing = null),});/*Ωignore_startΩ*/() => kind = __sveltets_2_any(null);/*Ωignore_endΩ*/ { svelteHTML.createElement("option", { "value":`local`,});  } { svelteHTML.createElement("option", { "value":`remote`,});  } { svelteHTML.createElement("option", { "value":`dynamic`,});  } }
            { svelteHTML.createElement("input", {          "class":`mono`,"bind:value":listen,"oninput":() => (viewing = null),"placeholder":kind === "remote" ? "remote port" : "local port","aria-label":`Listen port`,"required":true,});/*Ωignore_startΩ*/() => listen = __sveltets_2_any(null);/*Ωignore_endΩ*/}
          if(kind !== "dynamic"){  { svelteHTML.createElement("input", {          "class":`mono`,"bind:value":dest,"oninput":() => (viewing = null),"placeholder":`host:port`,"aria-label":`Target`,"required":true,});/*Ωignore_startΩ*/() => dest = __sveltets_2_any(null);/*Ωignore_endΩ*/}}else{ { svelteHTML.createElement("span", { "class":`na`,});   }}
           { svelteHTML.createElement("button", {   "class":`btn primary`,"disabled":busy,});  }
         }
       }
    }
    if(error){ { svelteHTML.createElement("p", {   "class":`msg bad`,"role":`alert`,});error; }}
    if(note && !error){ { svelteHTML.createElement("p", { "class":`msg good`,});note; }}
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const ForwardsPanel__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type ForwardsPanel__SvelteComponent_ = ReturnType<typeof ForwardsPanel__SvelteComponent_>;
/*Ωignore_endΩ*/export default ForwardsPanel__SvelteComponent_;