///<reference types="svelte" />
;
import type { Node } from "./panes";
import { isLeaf } from "./panes";
import { ws } from "./workspace.svelte";
import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
import PaneView from "./PaneView.svelte";
import SearchBar from "./SearchBar.svelte";

;type $$ComponentProps =  { node: Node; tabId: string; multi: boolean; profiles: Profile[] };function $$render() {

  
  
  
  
  
  

  let { node, tabId, multi, profiles }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  // Moves the pane's persistent terminal element into this leaf.
  function attach(host: HTMLElement, id: string) {
    host.appendChild(ws.element(id));
    return { destroy() { /* element stays alive in the workspace; the next leaf re-adopts it */ } };
  }

  function drag(e: PointerEvent, sid: string, dir: "row" | "col") {
    const handle = e.currentTarget as HTMLElement;
    const box = handle.parentElement!.getBoundingClientRect();
    handle.setPointerCapture(e.pointerId);
    const move = (ev: PointerEvent) =>
      ws.setRatio(sid, dir === "row" ? (ev.clientX - box.left) / box.width : (ev.clientY - box.top) / box.height);
    const up = () => { handle.removeEventListener("pointermove", move); handle.removeEventListener("pointerup", up); };
    handle.addEventListener("pointermove", move);
    handle.addEventListener("pointerup", up);
  }

  const tab = $derived(ws.tabs.find((t) => t.id === tabId));
;
async () => {

if(isLeaf(node)){
  const info = ws.panes[node.id];
  const prof = profiles.find((p) => p.name === info?.target);
   { svelteHTML.createElement("section", {    "class":`leaf`,"onpointerdowncapture":() => ws.focus(node.id),});multi && tab?.focus === node.id;
    if(multi){
       { svelteHTML.createElement("header", {});
         { svelteHTML.createElement("span", {   "class":`dot`,});info && !info.ended;info?.ended; }
         { svelteHTML.createElement("strong", {});info?.target ?? "Local"; }
        if(prof){ { svelteHTML.createElement("span", { "class":`sub`,});prof.user; prof.host; }}
        if(info?.ended){ { svelteHTML.createElement("span", { "class":`gone`,});  }}
         { svelteHTML.createElement("button", {     "class":`close`,"aria-label":`Close pane`,"onclick":() => ws.closePane(node.id),});  }
       }
    }
     {const $$action_0 = __sveltets_2_ensureAction(attach(svelteHTML.mapElementTag('div'),(node.id)));{ svelteHTML.createElement("div", __sveltets_2_union($$action_0), {   "class":`term`,}); }}
    if(ws.searchPane === node.id && ws.session(node.id)){ { const $$_raBhcraeS1C = __sveltets_2_ensureComponent(SearchBar); new $$_raBhcraeS1C({ target: __sveltets_2_any(), props: {    "session":ws.session(node.id)!,"onclose":() => ws.closeSearch(),}});}}
    if(info?.ended){
       { svelteHTML.createElement("div", {   "class":`ended`,"role":`alert`,});
         { svelteHTML.createElement("div", {}); { svelteHTML.createElement("strong", {});info.error ? "Could not connect" : "Session ended"; } { svelteHTML.createElement("span", {});info.error || `Press Enter to reconnect to ${info.target ?? "local shell"}`; } }
         { svelteHTML.createElement("button", {   "class":`btn primary`,"onclick":() => ws.reconnect(node.id),});  }
       }
    }
   }
}else{
   { svelteHTML.createElement("div", {  "class":`split`,});node.dir === "col";
     { svelteHTML.createElement("div", {   "class":`side`,"style":`flex: ${node.ratio} 1 0`,}); { const $$_weiVenaP2C = __sveltets_2_ensureComponent(PaneView); new $$_weiVenaP2C({ target: __sveltets_2_any(), props: {     "node":node.a,tabId,multi,profiles,}});} }
     { svelteHTML.createElement("div", {       "class":`handle`,"role":`separator`,"aria-orientation":node.dir === "row" ? "vertical" : "horizontal","onpointerdown":(e) => drag(e, node.sid, node.dir),}); }
     { svelteHTML.createElement("div", {   "class":`side`,"style":`flex: ${1 - node.ratio} 1 0`,}); { const $$_weiVenaP2C = __sveltets_2_ensureComponent(PaneView); new $$_weiVenaP2C({ target: __sveltets_2_any(), props: {     "node":node.b,tabId,multi,profiles,}});} }
   }
}


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const PaneView__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type PaneView__SvelteComponent_ = ReturnType<typeof PaneView__SvelteComponent_>;
/*Ωignore_endΩ*/export default PaneView__SvelteComponent_;