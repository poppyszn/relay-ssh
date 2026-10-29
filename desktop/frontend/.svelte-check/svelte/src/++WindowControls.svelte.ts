///<reference types="svelte" />
;
import { Window } from "@wailsio/runtime";
function $$render() {

  

  // Relay's own window buttons for the frameless window, drawn like macOS's and
  // centred in the 40px strip. Glyphs appear while the pointer is over the group.
  let hover = $state(false);
;
async () => {

 { svelteHTML.createElement("div", {         "class":`controls nodrag`,"role":`group`,"aria-label":`Window controls`,"onmouseenter":() => (hover = true),"onmouseleave":() => (hover = false),});
   { svelteHTML.createElement("button", {       "class":`close`,"aria-label":`Close window`,"title":`Close`,"onclick":() => Window.Close(),});
    if(hover){ { svelteHTML.createElement("svg", {       "width":`8`,"height":`8`,"viewBox":`0 0 8 8`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {        "d":`M1 1l6 6M7 1L1 7`,"stroke":`#4d0000`,"stroke-width":`1.3`,"stroke-linecap":`round`,});} }}
   }
   { svelteHTML.createElement("button", {       "class":`min`,"aria-label":`Minimise window`,"title":`Minimise`,"onclick":() => Window.Minimise(),});
    if(hover){ { svelteHTML.createElement("svg", {       "width":`8`,"height":`8`,"viewBox":`0 0 8 8`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {        "d":`M1 4h6`,"stroke":`#5a3a00`,"stroke-width":`1.3`,"stroke-linecap":`round`,});} }}
   }
   { svelteHTML.createElement("button", {       "class":`zoom`,"aria-label":`Zoom window`,"title":`Zoom`,"onclick":() => Window.ToggleMaximise(),});
    if(hover){ { svelteHTML.createElement("svg", {       "width":`8`,"height":`8`,"viewBox":`0 0 8 8`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {    "d":`M1.2 1.2h3.3L1.2 4.5zM6.8 6.8H3.5l3.3-3.3z`,"fill":`#0b3d12`,});} }}
   }
 }


};
return { props: {} as Record<string, never>, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const WindowControls__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type WindowControls__SvelteComponent_ = ReturnType<typeof WindowControls__SvelteComponent_>;
/*Ωignore_endΩ*/export default WindowControls__SvelteComponent_;