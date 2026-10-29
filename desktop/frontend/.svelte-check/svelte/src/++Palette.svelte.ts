///<reference types="svelte" />
;
import { tick } from "svelte";
import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
import { matches } from "./lib";
;

  export interface Action { id: string; label: string; hint: string };
;type $$ComponentProps =  {
    profiles: Profile[];
    active: Set<string>;
    actions: Action[];
    onconnect: (p: Profile, split: boolean) => void;
    onaction: (id: string) => void;
    onclose: () => void;
  };function $$render() {

  
  
  

  let { profiles, active, actions, onconnect, onaction, onclose }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  let query = $state("");
  let sel = $state(0);
  let input: HTMLInputElement;

  const conns = $derived(profiles.filter((p) => matches(p, query)).slice(0, 8));
  const acts = $derived(query.trim() ? actions.filter((a) => a.label.toLowerCase().includes(query.trim().toLowerCase())) : actions);
  const total = $derived(conns.length + acts.length);
  $effect(() => { query; sel = 0; });
  $effect(() => { tick().then(() => input?.focus()); });

  function pick(i: number, split = false) {
    if (i < conns.length) onconnect(conns[i], split);
    else if (acts[i - conns.length]) onaction(acts[i - conns.length].id);
  }
  function key(e: KeyboardEvent) {
    if (e.key === "Escape") onclose();
    else if (e.key === "ArrowDown") { sel = Math.min(sel + 1, total - 1); e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = Math.max(sel - 1, 0); e.preventDefault(); }
    else if (e.key === "Enter") { pick(sel, e.metaKey || e.ctrlKey); e.preventDefault(); }
  }
;
async () => {

 { svelteHTML.createElement("div", {     "class":`scrim`,"onclick":onclose,"role":`presentation`,});
   { svelteHTML.createElement("div", {           "class":`box`,"role":`dialog`,"aria-label":`Quick open`,"tabindex":-1,"onclick":(e) => e.stopPropagation(),"onkeydown":key,});
     { svelteHTML.createElement("div", { "class":`field`,});
       { svelteHTML.createElement("svg", {               "width":`18`,"height":`18`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("circle", {      "cx":`11`,"cy":`11`,"r":`7`,});} { svelteHTML.createElement("path", {  "d":`M20 20l-3.5-3.5`,});} }
       { const $$_input3 = svelteHTML.createElement("input", {        "bind:value":query,"placeholder":`Connect to…`,"aria-label":`Connect to`,});input = $$_input3;/*Ωignore_startΩ*/() => query = __sveltets_2_any(null);/*Ωignore_endΩ*/}
       { svelteHTML.createElement("span", { "class":`hint`,});      }
     }
     { svelteHTML.createElement("div", {   "class":`list`,"role":`listbox`,});
      if(conns.length){ { svelteHTML.createElement("div", { "class":`label sect`,});  }}
          for(let p of __sveltets_2_ensureArray(conns)){let i = 1;p.name;
        
         { svelteHTML.createElement("div", {          "class":`row`,"role":`option`,"aria-selected":i === sel,"onmouseenter":() => (sel = i),"onclick":() => pick(i),});i === sel;
           { svelteHTML.createElement("span", {  "class":`dot`,});active.has(p.name); }
           { svelteHTML.createElement("span", { "class":`col`,}); { svelteHTML.createElement("span", { "class":`name`,});p.name; } { svelteHTML.createElement("span", { "class":`sub mono`,});p.folder ? p.folder + " · " : "";p.user; p.host;p.port !== 22 ? ":" + p.port : "";p.jump ? " · via " + p.jump : ""; } }
           { svelteHTML.createElement("span", { "class":`tags`,});  for(let t of __sveltets_2_ensureArray(p.tags ?? [])){ { svelteHTML.createElement("span", { "class":`chip`,});t; }}if(i === sel){ { svelteHTML.createElement("span", { "class":`kbd`,});  }} }
         }
      }
      if(acts.length){ { svelteHTML.createElement("div", { "class":`label sect`,});  }}
          for(let a of __sveltets_2_ensureArray(acts)){let j = 1;a.id;
        const i = conns.length + j;
        
         { svelteHTML.createElement("div", {          "class":`row act`,"role":`option`,"aria-selected":i === sel,"onmouseenter":() => (sel = i),"onclick":() => pick(i),});i === sel;
           { svelteHTML.createElement("span", { "class":`col`,}); { svelteHTML.createElement("span", {});a.label; } } { svelteHTML.createElement("span", { "class":`kbd`,});a.hint; }
         }
      }
      if(!total){ { svelteHTML.createElement("div", { "class":`none`,});  }}
     }
     { svelteHTML.createElement("div", { "class":`foot`,}); { svelteHTML.createElement("span", {}); { svelteHTML.createElement("b", {});  }  } { svelteHTML.createElement("span", {}); { svelteHTML.createElement("b", {});  }     } { svelteHTML.createElement("span", {}); { svelteHTML.createElement("b", {});  }    } { svelteHTML.createElement("span", { "class":`right`,}); { svelteHTML.createElement("b", {});  }  } }
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const Palette__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type Palette__SvelteComponent_ = ReturnType<typeof Palette__SvelteComponent_>;
/*Ωignore_endΩ*/export default Palette__SvelteComponent_;