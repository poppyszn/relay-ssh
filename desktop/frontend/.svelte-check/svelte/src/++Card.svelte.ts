///<reference types="svelte" />
;
;type $$ComponentProps =  {
    text: string; sub?: string; confirm: string; danger?: boolean; onconfirm: () => void; oncancel: () => void; indent?: number;
  };function $$render() {

  let { text, sub = "", confirm, danger = false, onconfirm, oncancel, indent = 14 }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();
;
async () => {

 { svelteHTML.createElement("div", {           "class":`wrap`,"style":`padding-left: ${indent}px`,"role":`alertdialog`,"tabindex":-1,"aria-label":text,"onkeydown":(e) => e.key === "Escape" && oncancel(),});
   { svelteHTML.createElement("div", { "class":`card`,});
     { svelteHTML.createElement("span", { "class":`arrow`,}); }
     { svelteHTML.createElement("div", { "class":`text`,});text; }
    if(sub){ { svelteHTML.createElement("div", { "class":`sub`,});sub; }}
     { svelteHTML.createElement("div", { "class":`buttons`,});
       { svelteHTML.createElement("button", {   "class":`btn`,"onclick":oncancel,});  }
       { svelteHTML.createElement("button", {   "class":`btn primary`,"onclick":onconfirm,});danger;confirm; }
     }
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const Card__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type Card__SvelteComponent_ = ReturnType<typeof Card__SvelteComponent_>;
/*Ωignore_endΩ*/export default Card__SvelteComponent_;