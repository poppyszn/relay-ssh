///<reference types="svelte" />
;
import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
import { parseTags } from "./lib";

;type $$ComponentProps =  {
    initial: Profile;
    isNew: boolean;
    others: string[];
    folders: string[];
    onsave: (p: Profile) => Promise<void>;
    oncancel: () => void;
  };function $$render() {

  
  

  let { initial, isNew, others, folders, onsave, oncancel }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  let p = $state<Profile>(structuredClone($state.snapshot(initial)));
  let tags = $state((initial.tags ?? []).join(", "));
  let error = $state("");
  let busy = $state(false);

  async function submit(e: Event) {
    e.preventDefault();
    busy = true;
    error = "";
    try {
      await onsave({ ...p, port: Number(p.port), tags: parseTags(tags), key: p.auth === "key" ? p.key : "" } as Profile);
    } catch (err) {
      error = (err as Error).message ?? String(err);
    } finally {
      busy = false;
    }
  }
  const addTunnel = () => (p.tunnels ??= []).push({ type: "local", listen: "", target: "" });
;
async () => {

 { svelteHTML.createElement("div", {     "class":`scrim`,"role":`presentation`,"onkeydown":(e) => e.key === "Escape" && oncancel(),});
   { svelteHTML.createElement("form", {     "role":`dialog`,"aria-label":isNew ? "New connection" : "Edit connection","onsubmit":submit,});
     { svelteHTML.createElement("h2", {});isNew ? "New connection" : `Edit ${initial.name}`; }
     { svelteHTML.createElement("div", { "class":`grid`,});
       { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {    "bind:value":p.name,"required":true,"autofocus":true,"placeholder":`web-1`,});/*Ωignore_startΩ*/() => p.name = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
       { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {      "bind:value":p.folder,"list":`folders`,"placeholder":`prod/eu`,});/*Ωignore_startΩ*/() => p.folder = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
       { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {    "bind:value":p.host,"required":true,"placeholder":`10.0.0.5 or example.com`,});/*Ωignore_startΩ*/() => p.host = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
       { svelteHTML.createElement("label", {});   { svelteHTML.createElement("input", {        "type":`number`,"bind:value":p.port,"min":`1`,"max":`65535`,"required":true,});/*Ωignore_startΩ*/() => p.port = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
       { svelteHTML.createElement("label", {});   { svelteHTML.createElement("input", {  "bind:value":p.user,"required":true,});/*Ωignore_startΩ*/() => p.user = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
       { svelteHTML.createElement("label", {});
         { svelteHTML.createElement("select", { "bind:value":p.auth,});/*Ωignore_startΩ*/() => p.auth = __sveltets_2_any(null);/*Ωignore_endΩ*/
           { svelteHTML.createElement("option", { "value":`agent`,});  }
           { svelteHTML.createElement("option", { "value":`key`,});  }
           { svelteHTML.createElement("option", { "value":`password`,});    }
         }
       }
      if(p.auth === "key"){
         { svelteHTML.createElement("label", { "class":`wide`,});  { svelteHTML.createElement("input", {    "bind:value":p.key,"required":true,"placeholder":`id_ed25519 or ~/.ssh/id_ed25519`,});/*Ωignore_startΩ*/() => p.key = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
      }
       { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {    "bind:value":tags,"placeholder":`db, critical`,});/*Ωignore_startΩ*/() => tags = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
       { svelteHTML.createElement("label", {}); 
         { svelteHTML.createElement("select", { "bind:value":p.jump,});/*Ωignore_startΩ*/() => p.jump = __sveltets_2_any(null);/*Ωignore_endΩ*/
           { svelteHTML.createElement("option", {"value":"",});  }
            for(let o of __sveltets_2_ensureArray(others)){ { svelteHTML.createElement("option", { "value":o,});o; }}
         }
       }
       { svelteHTML.createElement("label", { "class":`check wide`,}); { svelteHTML.createElement("input", {    "type":`checkbox`,"bind:checked":p.forwardAgent,});/*Ωignore_startΩ*/() => p.forwardAgent = __sveltets_2_any(null);/*Ωignore_endΩ*/}    }
     }

     { svelteHTML.createElement("fieldset", {});
       { svelteHTML.createElement("legend", {});  }
         for(let t of __sveltets_2_ensureArray(p.tunnels ?? [])){let i = 1;
         { svelteHTML.createElement("div", { "class":`tunnel`,});
           { svelteHTML.createElement("select", {   "bind:value":t.type,"aria-label":`Tunnel type`,});/*Ωignore_startΩ*/() => t.type = __sveltets_2_any(null);/*Ωignore_endΩ*/
             { svelteHTML.createElement("option", { "value":`local`,});  } { svelteHTML.createElement("option", { "value":`remote`,});  } { svelteHTML.createElement("option", { "value":`dynamic`,});  }
           }
            { svelteHTML.createElement("input", {      "bind:value":t.listen,"placeholder":t.type === "remote" ? "remote port" : "local port","aria-label":`Listen`,"required":true,});/*Ωignore_startΩ*/() => t.listen = __sveltets_2_any(null);/*Ωignore_endΩ*/}
          if(t.type !== "dynamic"){  { svelteHTML.createElement("input", {      "bind:value":t.target,"placeholder":`host:port`,"aria-label":`Target`,"required":true,});/*Ωignore_startΩ*/() => t.target = __sveltets_2_any(null);/*Ωignore_endΩ*/}}
           { svelteHTML.createElement("button", {       "type":`button`,"class":`btn`,"aria-label":`Remove tunnel`,"onclick":() => p.tunnels?.splice(i, 1),});  }
         }
      }
       { svelteHTML.createElement("button", {     "type":`button`,"class":`btn`,"onclick":addTunnel,});   }
     }

     { svelteHTML.createElement("datalist", { "id":`folders`,});  for(let f of __sveltets_2_ensureArray(folders)){ { svelteHTML.createElement("option", { "value":f,}); }} }
    if(error){ { svelteHTML.createElement("p", {   "class":`error`,"role":`alert`,});error; }}
     { svelteHTML.createElement("div", { "class":`buttons`,});
       { svelteHTML.createElement("button", {     "type":`button`,"class":`btn`,"onclick":oncancel,});  }
       { svelteHTML.createElement("button", {   "class":`btn primary`,"disabled":busy,});isNew ? "Add" : "Save"; }
     }
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const ProfileForm__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type ProfileForm__SvelteComponent_ = ReturnType<typeof ProfileForm__SvelteComponent_>;
/*Ωignore_endΩ*/export default ProfileForm__SvelteComponent_;