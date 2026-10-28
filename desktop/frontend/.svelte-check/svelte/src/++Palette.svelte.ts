///<reference types="svelte" />
;
import { tick } from "svelte";
import type { Profile } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
import { matches } from "./lib";

;type $$ComponentProps =  { profiles: Profile[]; onpick: (p: Profile) => void; onclose: () => void };function $$render() {

  
  
  

  let { profiles, onpick, onclose }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();
  let query = $state("");
  let sel = $state(0);
  let input: HTMLInputElement;
  const results = $derived(profiles.filter((p) => matches(p, query)).slice(0, 12));
  $effect(() => { query; sel = 0; });
  $effect(() => { tick().then(() => input?.focus()); });

  function key(e: KeyboardEvent) {
    if (e.key === "Escape") onclose();
    else if (e.key === "ArrowDown") { sel = Math.min(sel + 1, results.length - 1); e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = Math.max(sel - 1, 0); e.preventDefault(); }
    else if (e.key === "Enter" && results[sel]) onpick(results[sel]);
  }
;
async () => {

 { svelteHTML.createElement("div", {     "class":`scrim`,"onclick":onclose,"role":`presentation`,});
   { svelteHTML.createElement("div", {           "class":`box`,"role":`dialog`,"aria-label":`Quick open`,"tabindex":-1,"onclick":(e) => e.stopPropagation(),"onkeydown":key,});
     { const $$_input2 = svelteHTML.createElement("input", {        "bind:value":query,"placeholder":`Connect to…`,"aria-label":`Connect to`,});input = $$_input2;/*Ωignore_startΩ*/() => query = __sveltets_2_any(null);/*Ωignore_endΩ*/}
     { svelteHTML.createElement("ul", { "role":`listbox`,});
          for(let p of __sveltets_2_ensureArray(results)){let i = 1;p.name;
        
         { svelteHTML.createElement("li", {        "role":`option`,"aria-selected":i === sel,"onmouseenter":() => (sel = i),"onclick":() => onpick(p),});i === sel;
           { svelteHTML.createElement("span", { "class":`name`,});p.name; }
           { svelteHTML.createElement("span", { "class":`sub`,});p.folder ? p.folder + " · " : "";p.user; p.host; }
         }
      }
         { svelteHTML.createElement("li", { "class":`none`,});  }
      
     }
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const Palette__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type Palette__SvelteComponent_ = ReturnType<typeof Palette__SvelteComponent_>;
/*Ωignore_endΩ*/export default Palette__SvelteComponent_;