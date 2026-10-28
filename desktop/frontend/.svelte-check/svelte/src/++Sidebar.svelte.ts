///<reference types="svelte" />
;
import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
import { groupByFolder, matches } from "./lib";

;type $$ComponentProps =  {
    profiles: Profile[];
    active: string;
    onconnect: (p: Profile) => void;
    onnew: () => void;
    onedit: (p: Profile) => void;
    ondelete: (p: Profile) => void;
    onpalette: () => void;
  };function $$render() {

  
  

  let { profiles, active, onconnect, onnew, onedit, ondelete, onpalette }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  let query = $state("");
  let collapsed = $state<Record<string, boolean>>({});
  const groups = $derived(groupByFolder(profiles.filter((p) => matches(p, query))));
  const searching = $derived(query.trim() !== "");
;
async () => {

 { svelteHTML.createElement("aside", {});
   { svelteHTML.createElement("div", { "class":`top`,});
     { svelteHTML.createElement("input", {      "bind:value":query,"placeholder":`Search  (#tag to filter by tag)`,"aria-label":`Search connections`,});/*Ωignore_startΩ*/() => query = __sveltets_2_any(null);/*Ωignore_endΩ*/}
     { svelteHTML.createElement("button", {     "class":`btn primary`,"onclick":onnew,"aria-label":`New connection`,});  }
   }
   { svelteHTML.createElement("button", {   "class":`hint`,"onclick":onpalette,});   { svelteHTML.createElement("kbd", {});  } }

   { svelteHTML.createElement("nav", { "aria-label":`Connections`,});
       for(let g of __sveltets_2_ensureArray(groups)){g.path;
      if(g.path !== ""){
         { svelteHTML.createElement("button", {     "class":`folder`,"onclick":() => (collapsed[g.path] = !collapsed[g.path]),"aria-expanded":searching || !collapsed[g.path],});
           { svelteHTML.createElement("span", { "class":`caret`,});searching || !collapsed[g.path] ? "▾" : "▸"; }g.path;
           { svelteHTML.createElement("span", { "class":`count`,});g.profiles.length; }
         }
      }
      if(searching || !collapsed[g.path]){
           for(let p of __sveltets_2_ensureArray(g.profiles)){p.name;
           { svelteHTML.createElement("div", {   "class":`row`,});p.name === active;g.path !== "";
             { svelteHTML.createElement("button", {     "class":`main`,"ondblclick":() => onconnect(p),"onclick":() => onconnect(p),});
               { svelteHTML.createElement("span", { "class":`name`,});p.name; }
               { svelteHTML.createElement("span", { "class":`sub`,});p.user; p.host;p.port !== 22 ? `:${p.port}` : ""; }
              if(p.tags?.length){ { svelteHTML.createElement("span", { "class":`tags`,});  for(let t of __sveltets_2_ensureArray(p.tags)){ { svelteHTML.createElement("span", { "class":`chip`,});t; }} }}
             }
             { svelteHTML.createElement("span", { "class":`actions`,});
               { svelteHTML.createElement("button", {     "title":`Edit`,"aria-label":`Edit ${p.name}`,"onclick":() => onedit(p),});  }
               { svelteHTML.createElement("button", {     "title":`Delete`,"aria-label":`Delete ${p.name}`,"onclick":() => ondelete(p),});  }
             }
           }
        }
      }
    }
       { svelteHTML.createElement("p", { "class":`empty`,});profiles.length === 0 ? "No connections yet. Press + to add one." : "No matches."; }
    
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const Sidebar__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type Sidebar__SvelteComponent_ = ReturnType<typeof Sidebar__SvelteComponent_>;
/*Ωignore_endΩ*/export default Sidebar__SvelteComponent_;