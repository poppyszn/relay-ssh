///<reference types="svelte" />
;
import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
import { groupByFolder, matches } from "./lib";

;type $$ComponentProps =  {
    profiles: Profile[];
    active: Set<string>;
    onconnect: (p: Profile) => void;
    onnew: () => void;
    onedit: (p: Profile) => void;
    onpalette: () => void;
  };function $$render() {

  
  

  let { profiles, active, onconnect, onnew, onedit, onpalette }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  let query = $state("");
  let collapsed = $state<Record<string, boolean>>({});
  const groups = $derived(groupByFolder(profiles.filter((p) => matches(p, query))));
  const searching = $derived(query.trim() !== "");
;
async () => {

 { svelteHTML.createElement("aside", {});
   { svelteHTML.createElement("div", { "class":`top`,});
     { svelteHTML.createElement("div", { "class":`search`,});
       { svelteHTML.createElement("svg", {               "width":`14`,"height":`14`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("circle", {      "cx":`11`,"cy":`11`,"r":`7`,});} { svelteHTML.createElement("path", {  "d":`M20 20l-3.5-3.5`,});} }
       { svelteHTML.createElement("input", {      "bind:value":query,"placeholder":`Search`,"aria-label":`Search connections`,});/*Ωignore_startΩ*/() => query = __sveltets_2_any(null);/*Ωignore_endΩ*/}
       { svelteHTML.createElement("button", {     "class":`kbd`,"onclick":onpalette,"title":`Quick open`,});  }
     }
     { svelteHTML.createElement("button", {       "class":`add`,"onclick":onnew,"aria-label":`New connection`,"title":`New connection`,});  }
   }

   { svelteHTML.createElement("nav", { "aria-label":`Connections`,});
       for(let g of __sveltets_2_ensureArray(groups)){g.path;
       { svelteHTML.createElement("button", {     "class":`bar`,"onclick":() => (collapsed[g.path] = !collapsed[g.path]),"aria-expanded":searching || !collapsed[g.path],});
         { svelteHTML.createElement("span", { "class":`label`,});g.path === "" ? "Unfiled" : g.path; }
         { svelteHTML.createElement("span", { "class":`label count`,});g.profiles.length; }
       }
      if(searching || !collapsed[g.path]){
           for(let p of __sveltets_2_ensureArray(g.profiles)){p.name;
           { svelteHTML.createElement("div", {  "class":`row`,});active.has(p.name);
             { svelteHTML.createElement("button", {   "class":`main`,"onclick":() => onconnect(p),});
               { svelteHTML.createElement("span", { "class":`line1`,}); { svelteHTML.createElement("span", {  "class":`dot`,});active.has(p.name); } { svelteHTML.createElement("span", { "class":`name`,});p.name; }  for(let t of __sveltets_2_ensureArray(p.tags ?? [])){ { svelteHTML.createElement("span", { "class":`chip`,});t; }} }
               { svelteHTML.createElement("span", { "class":`sub mono`,});p.user; p.host;p.port !== 22 ? `:${p.port}` : ""; }
             }
             { svelteHTML.createElement("button", {       "class":`edit`,"aria-label":`Edit ${p.name}`,"title":`Edit`,"onclick":() => onedit(p),});
               { svelteHTML.createElement("svg", {                 "width":`13`,"height":`13`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z`,});} }
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