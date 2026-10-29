///<reference types="svelte" />
;
import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
import { dragSource, sb } from "./foldertree.svelte";

;type $$ComponentProps =  {
    profile: Profile;
    depth: number;
    live: boolean;
    current: boolean;
    onconnect: (p: Profile) => void;
    onedit: (p: Profile) => void;
  };function $$render() {

  
  

  let { profile, depth, live, current, onconnect, onedit }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  const tags = $derived((profile.tags ?? []).slice(0, 2));
;
async () => {

 {const $$action_0 = __sveltets_2_ensureAction(dragSource(svelteHTML.mapElementTag('div'),({ kind: "profile", id: profile.name, label: profile.name })));{ svelteHTML.createElement("div", __sveltets_2_union($$action_0), {       "class":`row`,"style":`padding-left: ${14 + 14 * depth + 8}px`,});current;sb.drag?.kind === "profile" && sb.drag.id === profile.name;
     for(let _ of __sveltets_2_ensureArray(Array(depth))){let i = 1; { svelteHTML.createElement("span", {   "class":`guide`,"style":`left: ${18 + 14 * i}px`,}); }}
   { svelteHTML.createElement("button", {     "class":`main`,"onclick":() => !sb.justDragged && onconnect(profile),"title":`Open ${profile.name}`,});
     { svelteHTML.createElement("span", { "class":`l1`,});
       { svelteHTML.createElement("span", {  "class":`dot`,});live; }
       { svelteHTML.createElement("span", { "class":`name`,});profile.name; }
        for(let t of __sveltets_2_ensureArray(tags)){ { svelteHTML.createElement("span", { "class":`tag`,});t; }}
      if(profile.jump){ { svelteHTML.createElement("span", {   "class":`jump`,"title":`Connects through ${profile.jump}`,}); { svelteHTML.createElement("svg", {                 "width":`10`,"height":`10`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2.2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M5 12h12M13 8l4 4-4 4`,});} }profile.jump; }}
     }
     { svelteHTML.createElement("span", { "class":`host mono`,});profile.user; profile.host;profile.port !== 22 ? `:${profile.port}` : ""; }
   }
   { svelteHTML.createElement("button", {       "class":`edit`,"aria-label":`Edit ${profile.name}`,"title":`Edit`,"onclick":() => onedit(profile),});
     { svelteHTML.createElement("svg", {                 "width":`13`,"height":`13`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z`,});} }
   }
 }}


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const ConnRow__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type ConnRow__SvelteComponent_ = ReturnType<typeof ConnRow__SvelteComponent_>;
/*Ωignore_endΩ*/export default ConnRow__SvelteComponent_;