///<reference types="svelte" />
;
import type { Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
import type { TreeNode } from "./lib";
import { dragSource, sb } from "./foldertree.svelte";
import ConnRow from "./ConnRow.svelte";
import Card from "./Card.svelte";
import Branch from "./Branch.svelte";

;type $$ComponentProps =  {
    node: TreeNode; depth: number; searching: boolean; open: Set<string>; current: string;
    onconnect: (p: Profile) => void; onedit: (p: Profile) => void;
  };function $$render() {

  
  
  
  
  
  

  let { node, depth, searching, open, current, onconnect, onedit }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  const expanded = $derived(sb.expanded(node.path, searching));
  const over = $derived(sb.dropTarget === node.path);
  const focus = (el: HTMLElement) => { el.focus(); (el as HTMLInputElement).select?.(); };
  const pad = $derived(14 + 14 * depth);
;
async () => {

 { svelteHTML.createElement("div", {   "class":`branch`,...__sveltets_2_empty({"data-drop":node.path}),});
   {const $$action_0 = __sveltets_2_ensureAction(dragSource(svelteHTML.mapElementTag('div'),({ kind: "folder", id: node.path, label: node.name })));{ svelteHTML.createElement("div", __sveltets_2_union($$action_0), {          "class":`head`,"style":`padding-left: ${pad}px`,"title":over && !sb.dropOk ? sb.dropReason : "",});sb.selectedFolder === node.path;over && sb.dropOk;over && !sb.dropOk;
       for(let _ of __sveltets_2_ensureArray(Array(depth))){let i = 1; { svelteHTML.createElement("span", {   "class":`guide`,"style":`left: ${18 + 14 * i}px`,}); }}
     { svelteHTML.createElement("button", {       "class":`caret`,"aria-label":expanded ? "Collapse" : "Expand","onclick":() => sb.toggle(node.path),"style":`transform: rotate(${expanded ? 90 : 0}deg)`,});
       { svelteHTML.createElement("svg", {                 "width":`11`,"height":`11`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2.4`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M9 6l6 6-6 6`,});} }
     }
     { svelteHTML.createElement("button", {     "class":`name`,"onclick":() => !sb.justDragged && (sb.select(node.path), searching || sb.toggle(node.path)),"ondblclick":() => sb.startRename(node.path),});
       { svelteHTML.createElement("svg", {                    "class":`ico`,"width":`15`,"height":`15`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,});node.count === 0;
        if(expanded){ { svelteHTML.createElement("path", {  "d":`M3 8a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v1H7l-3 8H5a2 2 0 0 1-2-2zM7 11h15l-3 8H4z`,});}}else{ { svelteHTML.createElement("path", {  "d":`M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z`,});}}
       }
      if(sb.renaming === node.path){
         {const $$action_0 = __sveltets_2_ensureAction(focus(svelteHTML.mapElementTag('input')));{ svelteHTML.createElement("input", __sveltets_2_union($$action_0), {          "class":`edit`,"bind:value":sb.renameTo,"onclick":(e) => e.stopPropagation(),"onblur":() => sb.commitRename(),"onkeydown":(e) => { if (e.key === "Enter") sb.commitRename(); if (e.key === "Escape") { e.stopPropagation(); sb.renaming = ""; } },});/*Ωignore_startΩ*/() => sb.renameTo = __sveltets_2_any(null);/*Ωignore_endΩ*/}}
      }else{ { svelteHTML.createElement("span", { "class":`label-text`,});node.name; }}
     }
     { svelteHTML.createElement("span", { "class":`count`,});node.count; }
     { svelteHTML.createElement("button", {     "class":`more`,"aria-label":`Folder options`,"onclick":() => (sb.menuFolder = sb.menuFolder === node.path ? "" : node.path),});
       { svelteHTML.createElement("svg", {         "width":`14`,"height":`14`,"viewBox":`0 0 24 24`,"fill":`currentColor`,"aria-hidden":`true`,}); { svelteHTML.createElement("circle", {      "cx":`5`,"cy":`12`,"r":`1.5`,});} { svelteHTML.createElement("circle", {      "cx":`12`,"cy":`12`,"r":`1.5`,});} { svelteHTML.createElement("circle", {      "cx":`19`,"cy":`12`,"r":`1.5`,});} }
     }
    if(sb.menuFolder === node.path){
       { svelteHTML.createElement("div", {   "class":`pop`,"role":`menu`,});
         { svelteHTML.createElement("button", {   "role":`menuitem`,"onclick":() => sb.startCreate(node.path),});  }
         { svelteHTML.createElement("button", {   "role":`menuitem`,"onclick":() => sb.startRename(node.path),});  }
         { svelteHTML.createElement("button", {     "role":`menuitem`,"class":`danger`,"onclick":() => { sb.confirmDelete = node.path; sb.menuFolder = ""; },});  }
       }
    }
   }}

  if(sb.confirmDelete === node.path){
     { const $$_draC1C = __sveltets_2_ensureComponent(Card); new $$_draC1C({ target: __sveltets_2_any(), props: {            "text":`Delete the folder ${node.name}?`,"sub":node.count ? `It still holds ${node.count} connection${node.count === 1 ? "" : "s"}, so this will be refused. Move them out first.` : "It is empty. Nothing else is touched.","confirm":`Delete`,"danger":true,"indent":pad + 8,"onconfirm":() => sb.remove(node.path),"oncancel":() => (sb.confirmDelete = ""),}});}
  }
  if(sb.pending && sb.pending.to === node.path){
     { const $$_draC1C = __sveltets_2_ensureComponent(Card); new $$_draC1C({ target: __sveltets_2_any(), props: {            "text":sb.pending.text,"sub":sb.pending.sub,"confirm":`Move`,"indent":pad + 8,"onconfirm":() => sb.confirmPending(),"oncancel":() => (sb.pending = null),}});}
  }

  if(expanded){
    if(sb.creatingIn === node.path){
       { svelteHTML.createElement("div", {   "class":`new`,"style":`padding-left: ${pad + 14 + 8}px`,}); { svelteHTML.createElement("svg", {             "width":`15`,"height":`15`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z`,});} }
         {const $$action_0 = __sveltets_2_ensureAction(focus(svelteHTML.mapElementTag('input')));{ svelteHTML.createElement("input", __sveltets_2_union($$action_0), {          "class":`edit`,"placeholder":`Folder name`,"bind:value":sb.newName,"onblur":() => sb.commitCreate(),"onkeydown":(e) => { if (e.key === "Enter") sb.commitCreate(); if (e.key === "Escape") { e.stopPropagation(); sb.creatingIn = null; } },});/*Ωignore_startΩ*/() => sb.newName = __sveltets_2_any(null);/*Ωignore_endΩ*/}} }
    }
       for(let c of __sveltets_2_ensureArray(node.children)){c.path;
       { const $$_hcnarB1C = __sveltets_2_ensureComponent(Branch); new $$_hcnarB1C({ target: __sveltets_2_any(), props: {         "node":c,"depth":depth + 1,searching,open,current,onconnect,onedit,}});}
    }
       for(let p of __sveltets_2_ensureArray(node.profiles)){p.name;
       { const $$_woRnnoC1C = __sveltets_2_ensureComponent(ConnRow); new $$_woRnnoC1C({ target: __sveltets_2_any(), props: {          "profile":p,"depth":depth + 1,"live":open.has(p.name),"current":current === p.name,onconnect,onedit,}});}
    }
  }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const Branch__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type Branch__SvelteComponent_ = ReturnType<typeof Branch__SvelteComponent_>;
/*Ωignore_endΩ*/export default Branch__SvelteComponent_;