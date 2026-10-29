///<reference types="svelte" />
;
import { ws } from "./workspace.svelte";
import { leaves } from "./panes";

;type $$ComponentProps =  { onsidebar: () => void; onsettings: () => void };function $$render() {

  
  

  let { onsidebar, onsettings }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();
  const mac = navigator.userAgent.includes("Mac");

  function status(t: (typeof ws.tabs)[number]) {
    const ids = leaves(t.root);
    return ids.every((id) => ws.panes[id]?.ended) ? "err" : "ok";
  }
;
async () => {

 { svelteHTML.createElement("div", {  "class":`strip drag`,});mac;
   { svelteHTML.createElement("div", {   "class":`tabs nodrag`,"role":`tablist`,});
       for(let t of __sveltets_2_ensureArray(ws.tabs)){t.id;
       { svelteHTML.createElement("div", {    "class":`tab`,"role":`presentation`,});t.id === ws.activeId;
         { svelteHTML.createElement("button", {       "class":`main`,"role":`tab`,"aria-selected":t.id === ws.activeId,"onclick":() => ws.select(t.id),});
           { svelteHTML.createElement("span", { "class":`dot ${status(t)}`,}); }
           { svelteHTML.createElement("span", { "class":`title`,});ws.tabTitle(t); }
          if(leaves(t.root).length > 1){ { svelteHTML.createElement("span", { "class":`chip`,});leaves(t.root).length; }}
         }
         { svelteHTML.createElement("button", {     "class":`x`,"aria-label":`Close tab`,"onclick":() => ws.closeTab(t.id),});  }
       }
    }
     { svelteHTML.createElement("button", {       "class":`plus`,"aria-label":`New local tab`,"title":`New local tab`,"onclick":() => ws.openTab(null),});  }
   }
   { svelteHTML.createElement("div", { "class":`tools nodrag`,});
     { svelteHTML.createElement("button", {     "aria-label":`Toggle connections`,"title":`Toggle connections`,"onclick":onsidebar,});
       { svelteHTML.createElement("svg", {                 "width":`16`,"height":`16`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("rect", {          "x":`3`,"y":`4`,"width":`18`,"height":`16`,"rx":`2`,});} { svelteHTML.createElement("path", {  "d":`M9 4v16`,});} }
     }
     { svelteHTML.createElement("button", {     "aria-label":`Settings`,"title":`Settings`,"onclick":onsettings,});
       { svelteHTML.createElement("svg", {                 "width":`16`,"height":`16`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("circle", {      "cx":`12`,"cy":`12`,"r":`3`,});} { svelteHTML.createElement("path", {  "d":`M12 2v3M12 19v3M2 12h3M19 12h3M4.9 4.9l2.1 2.1M17 17l2.1 2.1M4.9 19.1L7 17M17 7l2.1-2.1`,});} }
     }
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const TabBar__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type TabBar__SvelteComponent_ = ReturnType<typeof TabBar__SvelteComponent_>;
/*Ωignore_endΩ*/export default TabBar__SvelteComponent_;