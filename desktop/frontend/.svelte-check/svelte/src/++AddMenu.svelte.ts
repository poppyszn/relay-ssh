///<reference types="svelte" />
;
import { onMount } from "svelte";

;type $$ComponentProps =  {
    /** The selected folder, "" for none: shown in the subfolder item. */
    folder: string;
    onconnection: () => void;
    onfolder: () => void;
    onsubfolder: () => void;
    onimport: () => void;
    onkey: () => void;
    onclose: () => void;
  };function $$render() {

  

  let { folder, onconnection, onfolder, onsubfolder, onimport, onkey, onclose }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  let sel = $state(0);
  const items = $derived([
    { icon: "term", title: "Connection", sub: "A server you SSH into", kbd: "⌘N", run: onconnection },
    { icon: "folder", title: "Folder", sub: "Group connections together", kbd: "⇧⌘N", run: onfolder },
    ...(folder ? [{ icon: "sub", title: `Subfolder in “${folder.split("/").pop()}”`, sub: "Nest a folder inside the selected one", kbd: "", run: onsubfolder }] : []),
    { icon: "import", title: "Import from ~/.ssh/config", sub: "Bring in hosts you already have", kbd: "", run: onimport, gap: true },
    { icon: "key", title: "Key", sub: "Adopt an existing key or generate a new one", kbd: "", run: onkey },
  ]);
  const icons: Record<string, string> = {
    term: "M4 17l6-6-6-6M12 19h8",
    folder: "M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z",
    sub: "M3 8a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v1H7l-3 8H5a2 2 0 0 1-2-2zM7 11h15l-3 8H4z",
    import: "M12 3v12M7 10l5 5 5-5M4 20h16",
    key: "M11 12l9-9M16 7l3 3M8 11a4 4 0 1 0 0 8 4 4 0 0 0 0-8z",
  };

  let box: HTMLElement;
  onMount(() => {
    const away = (e: PointerEvent) => !box.contains(e.target as Node) && !(e.target as HTMLElement).closest("[data-add-button]") && onclose();
    window.addEventListener("pointerdown", away, true);
    box.focus();
    return () => window.removeEventListener("pointerdown", away, true);
  });
  function key(e: KeyboardEvent) {
    if (e.key === "Escape") onclose();
    else if (e.key === "ArrowDown") { sel = (sel + 1) % items.length; e.preventDefault(); }
    else if (e.key === "ArrowUp") { sel = (sel + items.length - 1) % items.length; e.preventDefault(); }
    else if (e.key === "Enter") { items[sel].run(); e.preventDefault(); }
  }
;
async () => {

 { const $$_div0 = svelteHTML.createElement("div", {          "class":`menu`,"role":`menu`,"aria-label":`Add`,"tabindex":-1,"onkeydown":key,});box = $$_div0;
   { svelteHTML.createElement("div", { "class":`label head`,});  }
     for(let it of __sveltets_2_ensureArray(items)){let i = 1;
    if(it.gap){ { svelteHTML.createElement("div", { "class":`sep`,}); }}
     { svelteHTML.createElement("button", {        "class":`item`,"role":`menuitem`,"onmouseenter":() => (sel = i),"onclick":it.run,});i === sel;
       { svelteHTML.createElement("span", { "class":`ico`,}); { svelteHTML.createElement("svg", {                 "width":`16`,"height":`16`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":icons[it.icon],});} } }
       { svelteHTML.createElement("span", { "class":`txt`,}); { svelteHTML.createElement("span", { "class":`t`,});it.title; } { svelteHTML.createElement("span", { "class":`s`,});it.sub; } }
      if(it.kbd){ { svelteHTML.createElement("span", { "class":`kbd`,});it.kbd; }}
     }
  }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const AddMenu__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type AddMenu__SvelteComponent_ = ReturnType<typeof AddMenu__SvelteComponent_>;
/*Ωignore_endΩ*/export default AddMenu__SvelteComponent_;