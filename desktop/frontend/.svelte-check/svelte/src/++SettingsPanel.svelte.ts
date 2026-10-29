///<reference types="svelte" />
;
import { DENSITY, FONTS, THEMES, settings } from "./settings.svelte";

;type $$ComponentProps =  { onclose: () => void };function $$render() {

  

  let { onclose }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();
  const sizes = () => settings.fontSize;
  const bump = (d: number) => (settings.fontSize = Math.min(24, Math.max(10, settings.fontSize + d)));
;
async () => {

 { svelteHTML.createElement("div", {       "class":`scrim`,"role":`presentation`,"onclick":onclose,"onkeydown":(e) => e.key === "Escape" && onclose(),});
   { svelteHTML.createElement("div", {         "class":`panel`,"role":`dialog`,"aria-label":`Settings`,"tabindex":-1,"onclick":(e) => e.stopPropagation(),});
     { svelteHTML.createElement("header", {}); { svelteHTML.createElement("h2", {});  } { svelteHTML.createElement("button", {   "aria-label":`Close`,"onclick":onclose,});  } }

     { svelteHTML.createElement("section", {});
       { svelteHTML.createElement("div", { "class":`label`,});  }
       { svelteHTML.createElement("div", {     "class":`themes`,"role":`radiogroup`,"aria-label":`Theme`,});
          for(let [id, t] of __sveltets_2_ensureArray(Object.entries(THEMES))){
           { svelteHTML.createElement("button", {      "role":`radio`,"aria-checked":settings.theme === id,"onclick":() => (settings.theme = id),});settings.theme === id;
             { svelteHTML.createElement("span", {   "class":`swatch`,"style":`background: ${t.vars.term}; box-shadow: inset 0 -10px ${t.vars.rail}`,}); { svelteHTML.createElement("i", { "style":`background: ${t.vars.accent}`,}); } }
             { svelteHTML.createElement("span", {});t.name; }
           }
        }
       }
     }

     { svelteHTML.createElement("section", {});
       { svelteHTML.createElement("div", { "class":`label`,});  }
       { svelteHTML.createElement("div", { "class":`row`,});
         { svelteHTML.createElement("select", {   "bind:value":settings.font,"aria-label":`Font family`,});/*Ωignore_startΩ*/() => settings.font = __sveltets_2_any(null);/*Ωignore_endΩ*/
            for(let [id, f] of __sveltets_2_ensureArray(Object.entries(FONTS))){ { svelteHTML.createElement("option", { "value":id,});f.name; }}
         }
         { svelteHTML.createElement("div", {   "class":`step`,"aria-label":`Font size`,});
           { svelteHTML.createElement("button", {   "onclick":() => bump(-1),"aria-label":`Smaller`,});  }
           { svelteHTML.createElement("span", { "class":`mono`,});sizes();  }
           { svelteHTML.createElement("button", {   "onclick":() => bump(1),"aria-label":`Larger`,});  }
         }
       }
       { svelteHTML.createElement("div", {     "class":`seg`,"role":`radiogroup`,"aria-label":`Line spacing`,});
          for(let [id, d] of __sveltets_2_ensureArray(Object.entries(DENSITY))){
           { svelteHTML.createElement("button", {      "role":`radio`,"aria-checked":settings.density === id,"onclick":() => (settings.density = id),});settings.density === id;d.name; }
        }
       }
     }

     { svelteHTML.createElement("section", { "class":`last`,});
       { svelteHTML.createElement("div", { "class":`label`,});  }
       { svelteHTML.createElement("label", { "class":`check`,}); { svelteHTML.createElement("input", {    "type":`checkbox`,"bind:checked":settings.copyOnSelect,});/*Ωignore_startΩ*/() => settings.copyOnSelect = __sveltets_2_any(null);/*Ωignore_endΩ*/} { svelteHTML.createElement("span", {});   { svelteHTML.createElement("small", {});       } } }
       { svelteHTML.createElement("label", { "class":`check`,}); { svelteHTML.createElement("input", {    "type":`checkbox`,"bind:checked":settings.cursorBlink,});/*Ωignore_startΩ*/() => settings.cursorBlink = __sveltets_2_any(null);/*Ωignore_endΩ*/} { svelteHTML.createElement("span", {});  } }
     }
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const SettingsPanel__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type SettingsPanel__SvelteComponent_ = ReturnType<typeof SettingsPanel__SvelteComponent_>;
/*Ωignore_endΩ*/export default SettingsPanel__SvelteComponent_;