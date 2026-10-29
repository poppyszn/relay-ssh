///<reference types="svelte" />
;
import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
import { buildTree } from "./lib";
import { sb } from "./foldertree.svelte";
import Branch from "./Branch.svelte";
import ConnRow from "./ConnRow.svelte";
import AddMenu from "./AddMenu.svelte";
import Card from "./Card.svelte";

;type $$ComponentProps =  {
    profiles: Profile[];
    folders: string[];
    /** Connections that have an open tab. */
    open: Set<string>;
    /** The connection of the focused pane, if any. */
    current: string;
    onconnect: (p: Profile) => void;
    /** Starts a new connection, in the given folder. */
    onnew: (folder: string) => void;
    onedit: (p: Profile) => void;
    onpalette: () => void;
    ontools: (tab: "import" | "keys") => void;
    onchanged: () => void;
  };function $$render() {

  
  
  
  
  
  
  

  let { profiles, folders, open, current, onconnect, onnew, onedit, onpalette, ontools, onchanged }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  let query = $state("");
  let addOpen = $state(false);
  const searching = $derived(query.trim() !== "");
  const tree = $derived(buildTree(profiles, folders, query));
  const live = $derived(profiles.filter((p) => open.has(p.name)));
  const focus = (el: HTMLElement) => el.focus();

  $effect(() => {
    sb.data.profiles = profiles;
    sb.data.folders = folders;
    sb.onchanged = onchanged;
  });

  const add = (fn: () => void) => () => { addOpen = false; fn(); };
;
async () => {

 { svelteHTML.createElement("aside", {});
   { svelteHTML.createElement("div", { "class":`head`,});
     { svelteHTML.createElement("div", { "class":`title`,}); { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", { "class":`n`,});profiles.length; } }
     { svelteHTML.createElement("button", {       "class":`add`,...__sveltets_2_empty({"data-add-button":true}),"aria-haspopup":`menu`,"aria-expanded":addOpen,"onclick":() => (addOpen = !addOpen),});
       { svelteHTML.createElement("svg", {               "width":`14`,"height":`14`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2.4`,"stroke-linecap":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M12 5v14M5 12h14`,});} } { svelteHTML.createElement("span", {});  }
       { svelteHTML.createElement("svg", {                 "width":`11`,"height":`11`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2.4`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M6 9l6 6 6-6`,});} }
     }
    if(addOpen){
       { const $$_uneMddA2C = __sveltets_2_ensureComponent(AddMenu); new $$_uneMddA2C({ target: __sveltets_2_any(), props: {              "folder":sb.selectedFolder,"onclose":() => (addOpen = false),"onconnection":add(() => onnew(sb.selectedFolder)),"onfolder":add(() => sb.startCreate("")),"onsubfolder":add(() => sb.startCreate(sb.selectedFolder)),"onimport":add(() => ontools("import")),"onkey":add(() => ontools("keys")),}});}
    }
   }

   { svelteHTML.createElement("div", { "class":`searchrow`,});
     { svelteHTML.createElement("div", { "class":`search`,});
       { svelteHTML.createElement("svg", {               "width":`14`,"height":`14`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("circle", {      "cx":`11`,"cy":`11`,"r":`7`,});} { svelteHTML.createElement("path", {  "d":`M20 20l-3.5-3.5`,});} }
       { svelteHTML.createElement("input", {      "bind:value":query,"placeholder":`Search`,"aria-label":`Search connections`,});/*Ωignore_startΩ*/() => query = __sveltets_2_any(null);/*Ωignore_endΩ*/}
       { svelteHTML.createElement("button", {     "class":`kbd`,"onclick":onpalette,"title":`Quick open`,});  }
     }
   }

  if(sb.error){ { svelteHTML.createElement("p", {   "class":`err`,"role":`alert`,});sb.error; { svelteHTML.createElement("button", {   "aria-label":`Dismiss`,"onclick":() => (sb.error = ""),});  } }}

   { svelteHTML.createElement("nav", {  "aria-label":`Connections`,...__sveltets_2_empty({"data-scroll":true}),});
    if(live.length && !searching){
       { svelteHTML.createElement("div", { "class":`bar`,}); { svelteHTML.createElement("span", { "class":`label`,});  } { svelteHTML.createElement("span", { "class":`label`,});live.length; } }
         for(let p of __sveltets_2_ensureArray(live)){p.name;
         { const $$_woRnnoC2C = __sveltets_2_ensureComponent(ConnRow); new $$_woRnnoC2C({ target: __sveltets_2_any(), props: {        "profile":p,"depth":-1,"live":true,"current":current === p.name,onconnect,onedit,}});}
      }
    }

    if(tree.roots.length || sb.creatingIn === ""){
       { svelteHTML.createElement("div", { "class":`bar`,}); { svelteHTML.createElement("span", { "class":`label`,});  } { svelteHTML.createElement("span", { "class":`label`,});folders.length; } }
    }
    if(sb.creatingIn === ""){
       { svelteHTML.createElement("div", { "class":`new`,}); { svelteHTML.createElement("svg", {             "width":`15`,"height":`15`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z`,});} }
         {const $$action_0 = __sveltets_2_ensureAction(focus(svelteHTML.mapElementTag('input')));{ svelteHTML.createElement("input", __sveltets_2_union($$action_0), {          "class":`edit`,"placeholder":`Folder name`,"bind:value":sb.newName,"onblur":() => sb.commitCreate(),"onkeydown":(e) => { if (e.key === "Enter") sb.commitCreate(); if (e.key === "Escape") { e.stopPropagation(); sb.creatingIn = null; } },});/*Ωignore_startΩ*/() => sb.newName = __sveltets_2_any(null);/*Ωignore_endΩ*/}} }
    }
       for(let n of __sveltets_2_ensureArray(tree.roots)){n.path;
       { const $$_hcnarB2C = __sveltets_2_ensureComponent(Branch); new $$_hcnarB2C({ target: __sveltets_2_any(), props: {         "node":n,"depth":0,searching,open,current,onconnect,onedit,}});}
    }

    if(tree.unfiled.length){
      if(tree.roots.length){ { svelteHTML.createElement("div", { "class":`bar`,}); { svelteHTML.createElement("span", { "class":`label`,});  } { svelteHTML.createElement("span", { "class":`label`,});tree.unfiled.length; } }}
       { svelteHTML.createElement("div", {...__sveltets_2_empty({"data-drop":""}),});
           for(let p of __sveltets_2_ensureArray(tree.unfiled)){p.name;
           { const $$_woRnnoC3C = __sveltets_2_ensureComponent(ConnRow); new $$_woRnnoC3C({ target: __sveltets_2_any(), props: {          "profile":p,"depth":0,"live":open.has(p.name),"current":current === p.name,onconnect,onedit,}});}
        }
       }
    } else if (!tree.roots.length){
       { svelteHTML.createElement("p", { "class":`empty`,});profiles.length === 0 ? "No connections yet. Use Add to create one." : "No matches."; }
    }

    if(sb.drag){
       { svelteHTML.createElement("div", {    "class":`rootzone`,...__sveltets_2_empty({"data-drop":""}),});sb.dropTarget === "" && sb.dropOk;sb.dropTarget === "" && !sb.dropOk;        }
    }
    if(sb.pending && sb.pending.to === ""){
       { const $$_draC2C = __sveltets_2_ensureComponent(Card); new $$_draC2C({ target: __sveltets_2_any(), props: {            "text":sb.pending.text,"sub":sb.pending.sub,"confirm":`Move`,"indent":14,"onconfirm":() => sb.confirmPending(),"oncancel":() => (sb.pending = null),}});}
    }
   }

   { svelteHTML.createElement("footer", {});
     { svelteHTML.createElement("span", {});profiles.length; profiles.length === 1 ? "" : "s";  folders.length; folders.length === 1 ? "" : "s"; }
     { svelteHTML.createElement("span", { "class":`tools`,});
       { svelteHTML.createElement("button", {     "aria-label":`Import from ssh config`,"title":`Import from ~/.ssh/config`,"onclick":() => ontools("import"),});
         { svelteHTML.createElement("svg", {                 "width":`15`,"height":`15`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M12 3v12M7 10l5 5 5-5M4 20h16`,});} }
       }
       { svelteHTML.createElement("button", {     "aria-label":`Keys`,"title":`Keys`,"onclick":() => ontools("keys"),});
         { svelteHTML.createElement("svg", {                 "width":`15`,"height":`15`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("circle", {      "cx":`8`,"cy":`15`,"r":`4`,});} { svelteHTML.createElement("path", {  "d":`M11 12l9-9M16 7l3 3`,});} }
       }
     }
   }
 }

if(sb.drag){
   { svelteHTML.createElement("div", {    "class":`ghost`,"style":`left: ${sb.drag.x + 14}px; top: ${sb.drag.y + 10}px`,});sb.dropTarget !== null && !sb.dropOk;
     { svelteHTML.createElement("svg", {         "width":`14`,"height":`14`,"viewBox":`0 0 24 24`,"fill":`currentColor`,"aria-hidden":`true`,}); { svelteHTML.createElement("circle", {      "cx":`9`,"cy":`6`,"r":`1`,});} { svelteHTML.createElement("circle", {      "cx":`15`,"cy":`6`,"r":`1`,});} { svelteHTML.createElement("circle", {      "cx":`9`,"cy":`12`,"r":`1`,});} { svelteHTML.createElement("circle", {      "cx":`15`,"cy":`12`,"r":`1`,});} { svelteHTML.createElement("circle", {      "cx":`9`,"cy":`18`,"r":`1`,});} { svelteHTML.createElement("circle", {      "cx":`15`,"cy":`18`,"r":`1`,});} }
     { svelteHTML.createElement("b", {});sb.drag.label; }
    if(sb.dropTarget !== null && !sb.dropOk){ { svelteHTML.createElement("span", {});sb.dropReason; }}
   }
}


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const Sidebar__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type Sidebar__SvelteComponent_ = ReturnType<typeof Sidebar__SvelteComponent_>;
/*Ωignore_endΩ*/export default Sidebar__SvelteComponent_;