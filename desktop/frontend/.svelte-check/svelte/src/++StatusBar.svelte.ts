///<reference types="svelte" />
;
import { ws } from "./workspace.svelte";
import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

;type $$ComponentProps =  { profiles: Profile[]; themeName: string };function $$render() {

  
  

  let { profiles, themeName }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();
  const info = $derived(ws.panes[ws.focusedPane]);
  const prof = $derived(info?.target ? profiles.find((p) => p.name === info.target) : undefined);
  const size = $derived.by(() => {
    const t = ws.session(ws.focusedPane)?.term;
    return t ? `${t.cols}×${t.rows}` : "";
  });
;
async () => {

 { svelteHTML.createElement("footer", {});
  if(info){
     { svelteHTML.createElement("span", { "class":`seg`,}); { svelteHTML.createElement("span", {   "class":`dot`,});!info.ended;info.ended; }info.ended ? "Disconnected" : "Connected"; }
     { svelteHTML.createElement("span", { "class":`seg mono`,});prof ? `${prof.user}@${prof.host}:${prof.port}` : "local shell"; }
    if(prof?.jump){ { svelteHTML.createElement("span", { "class":`seg`,}); prof.jump; }}
    if(info.blocks){ { svelteHTML.createElement("span", { "class":`seg ok`,});  }}
  }else{
     { svelteHTML.createElement("span", { "class":`seg`,});  }
  }
   { svelteHTML.createElement("span", { "class":`spacer`,}); }
   { svelteHTML.createElement("span", { "class":`seg mono`,}); size; }
   { svelteHTML.createElement("span", { "class":`seg`,});themeName; }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const StatusBar__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type StatusBar__SvelteComponent_ = ReturnType<typeof StatusBar__SvelteComponent_>;
/*Ωignore_endΩ*/export default StatusBar__SvelteComponent_;