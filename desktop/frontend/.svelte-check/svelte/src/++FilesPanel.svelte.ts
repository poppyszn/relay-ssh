///<reference types="svelte" />
;
import { FilesService, type FileEntry } from "../bindings/github.com/poppyszn/relay-ssh/desktop";

;type $$ComponentProps =  { target: string; onclose: () => void };function $$render() {

  

  let { target, onclose }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  let path = $state("");
  let pathInput = $state("");
  let entries = $state<FileEntry[]>([]);
  let selected = $state("");
  let showHidden = $state(false);
  let busy = $state("");
  let error = $state("");
  let note = $state("");
  let renaming = $state("");
  let renameTo = $state("");
  let creating = $state(false);
  let newName = $state("");
  let confirm = $state<null | { kind: "delete" | "overwrite"; name: string; local?: string }>(null);

  const msg = (e: unknown) => (e as Error)?.message ?? String(e);
  const join = (dir: string, name: string) => (dir === "/" ? "/" + name : dir + "/" + name);
  const parent = (p: string) => p.replace(/\/[^/]+\/?$/, "") || "/";
  const base = (p: string) => p.split(/[\\/]/).pop() ?? p;
  const sel = $derived(entries.find((e) => e.name === selected));
  const shown = $derived(
    entries
      .filter((e) => showHidden || !e.name.startsWith("."))
      .sort((a, b) => Number(b.dir) - Number(a.dir) || a.name.localeCompare(b.name, undefined, { sensitivity: "base" })),
  );
  function size(n: number, dir: boolean) {
    if (dir) return "";
    if (n < 1024) return `${n} B`;
    if (n < 1024 ** 2) return `${(n / 1024).toFixed(1)} KB`;
    if (n < 1024 ** 3) return `${(n / 1024 ** 2).toFixed(1)} MB`;
    return `${(n / 1024 ** 3).toFixed(2)} GB`;
  }

  async function load(dir = path) {
    busy = "Loading…"; error = ""; selected = ""; renaming = ""; creating = false; confirm = null;
    try {
      const l = await FilesService.List(target, dir);
      path = l.path; pathInput = l.path; entries = l.entries ?? [];
    } catch (e) { error = msg(e); } finally { busy = ""; }
  }
  const open = (e: FileEntry) => (e.dir || e.link ? load(join(path, e.name)).then(() => { if (error && e.link) { error = ""; download(e); } }) : download(e));

  async function download(e: FileEntry) {
    error = ""; note = "";
    try {
      const dest = await FilesService.PickDownloadPath(e.name);
      if (!dest) return;
      busy = `Downloading ${e.name}…`;
      await FilesService.Download(target, join(path, e.name), dest);
      note = `Downloaded ${e.name}`;
    } catch (er) { error = msg(er); } finally { busy = ""; }
  }
  async function upload(local: string, overwrite = false) {
    const name = base(local);
    if (!overwrite && entries.some((e) => e.name === name)) return (confirm = { kind: "overwrite", name, local });
    confirm = null; error = ""; note = "";
    try {
      busy = `Uploading ${name}…`;
      await FilesService.Upload(target, local, join(path, name));
      note = `Uploaded ${name}`;
      await load();
    } catch (er) { error = msg(er); busy = ""; }
  }
  async function pickUpload() {
    try { const l = await FilesService.PickUploadPath(); if (l) await upload(l); } catch (e) { error = msg(e); }
  }
  async function mkdir() {
    const name = newName.trim();
    creating = false; newName = "";
    if (!name) return;
    try { await FilesService.Mkdir(target, join(path, name)); await load(); } catch (e) { error = msg(e); }
  }
  async function rename() {
    const from = renaming, to = renameTo.trim();
    renaming = "";
    if (!to || to === from) return;
    try { await FilesService.Rename(target, join(path, from), join(path, to)); await load(); } catch (e) { error = msg(e); }
  }
  async function remove() {
    const e = sel; confirm = null;
    if (!e) return;
    try { await FilesService.Remove(target, join(path, e.name), e.dir); note = `Deleted ${e.name}`; await load(); } catch (er) { error = msg(er); }
  }

  $effect(() => { load(""); });
;
async () => {

 { svelteHTML.createElement("div", {       "class":`scrim`,"role":`presentation`,"onclick":onclose,"onkeydown":(e) => e.key === "Escape" && !renaming && !creating && onclose(),});
   { svelteHTML.createElement("div", {         "class":`panel`,"role":`dialog`,"aria-label":`Files`,"tabindex":-1,"onclick":(e) => e.stopPropagation(),});
     { svelteHTML.createElement("header", {});
       { svelteHTML.createElement("h2", {});  { svelteHTML.createElement("span", { "class":`host mono`,});target; } }
       { svelteHTML.createElement("button", {     "class":`x`,"aria-label":`Close`,"onclick":onclose,});  }
     }

     { svelteHTML.createElement("div", { "class":`bar`,});
       { svelteHTML.createElement("button", {         "class":`icon`,"aria-label":`Up one folder`,"title":`Up`,"onclick":() => load(parent(path)),"disabled":path === "/" || !!busy,});
         { svelteHTML.createElement("svg", {                 "width":`15`,"height":`15`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M12 19V5M6 11l6-6 6 6`,});} }
       }
       { svelteHTML.createElement("button", {         "class":`icon`,"aria-label":`Refresh`,"title":`Refresh`,"onclick":() => load(),"disabled":!!busy,});
         { svelteHTML.createElement("svg", {                 "width":`15`,"height":`15`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M20 11a8 8 0 1 0-2.3 5.7M20 4v7h-7`,});} }
       }
       { svelteHTML.createElement("input", {        "class":`mono`,"bind:value":pathInput,"aria-label":`Folder path`,"onkeydown":(e) => e.key === "Enter" && load(pathInput),});/*Ωignore_startΩ*/() => pathInput = __sveltets_2_any(null);/*Ωignore_endΩ*/}
       { svelteHTML.createElement("label", { "class":`hid`,}); { svelteHTML.createElement("input", {    "type":`checkbox`,"bind:checked":showHidden,});/*Ωignore_startΩ*/() => showHidden = __sveltets_2_any(null);/*Ωignore_endΩ*/}  }
     }

     { svelteHTML.createElement("div", { "class":`tools`,});
       { svelteHTML.createElement("button", {     "class":`btn`,"onclick":pickUpload,"disabled":!!busy,});  }
       { svelteHTML.createElement("button", {     "class":`btn`,"onclick":() => { creating = true; newName = ""; },"disabled":!!busy,});  }
       { svelteHTML.createElement("span", { "class":`sp`,}); }
      if(sel && !confirm){
        if(!sel.dir){ { svelteHTML.createElement("button", {     "class":`btn`,"onclick":() => download(sel),"disabled":!!busy,});  }}
         { svelteHTML.createElement("button", {     "class":`btn`,"onclick":() => { renaming = sel.name; renameTo = sel.name; },"disabled":!!busy,});  }
         { svelteHTML.createElement("button", {     "class":`btn danger`,"onclick":() => (confirm = { kind: "delete", name: sel.name }),"disabled":!!busy,});  }
      }
     }

    if(confirm){
       { svelteHTML.createElement("div", {   "class":`confirm`,"role":`alertdialog`,});
        if(confirm.kind === "delete"){
           { svelteHTML.createElement("span", {});  { svelteHTML.createElement("b", {});confirm.name; }  target; sel?.dir ? "Only an empty folder can be deleted." : "This cannot be undone."; }
           { svelteHTML.createElement("button", {   "class":`btn`,"onclick":() => (confirm = null),});  } { svelteHTML.createElement("button", {   "class":`btn primary danger-fill`,"onclick":remove,});  }
        }else{
           { svelteHTML.createElement("span", {}); { svelteHTML.createElement("b", {});confirm.name; }      }
           { svelteHTML.createElement("button", {   "class":`btn`,"onclick":() => (confirm = null),});  } { svelteHTML.createElement("button", {   "class":`btn primary`,"onclick":() => upload(confirm!.local!, true),});  }
        }
       }
    }
    if(error){ { svelteHTML.createElement("p", {   "class":`msg bad`,"role":`alert`,});error; }}
    if(note && !error){ { svelteHTML.createElement("p", { "class":`msg good`,});note; }}

     { svelteHTML.createElement("div", {     "class":`list`,"role":`grid`,"aria-label":`Files`,});
       { svelteHTML.createElement("div", { "class":`row head`,}); { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {});  } }
      if(creating){
         { svelteHTML.createElement("div", { "class":`row`,}); { svelteHTML.createElement("span", { "class":`n`,}); { svelteHTML.createElement("svg", {               "class":`ico`,"width":`15`,"height":`15`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z`,});} } { svelteHTML.createElement("input", {        "class":`mono edit`,"bind:value":newName,"autofocus":true,"placeholder":`Folder name`,"onkeydown":(e) => { if (e.key === "Enter") mkdir(); if (e.key === "Escape") { e.stopPropagation(); creating = false; } },});/*Ωignore_startΩ*/() => newName = __sveltets_2_any(null);/*Ωignore_endΩ*/} } }
      }
         for(let e of __sveltets_2_ensureArray(shown)){e.name;
         { svelteHTML.createElement("div", {            "class":`row item`,"role":`row`,"tabindex":0,"onclick":() => (selected = e.name),"ondblclick":() => open(e),"onkeydown":(k) => k.key === "Enter" && open(e),});selected === e.name;
           { svelteHTML.createElement("span", { "class":`n`,});
             { svelteHTML.createElement("svg", {                  "class":`ico`,"width":`15`,"height":`15`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linejoin":`round`,"aria-hidden":`true`,});e.dir;
              if(e.dir){ { svelteHTML.createElement("path", {  "d":`M3 7a2 2 0 0 1 2-2h4l2 2h8a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z`,});}}else{ { svelteHTML.createElement("path", {  "d":`M6 3h8l4 4v14H6zM14 3v4h4`,});}}
             }
            if(renaming === e.name){
              
               { svelteHTML.createElement("input", {        "class":`mono edit`,"bind:value":renameTo,"autofocus":true,"onclick":(ev) => ev.stopPropagation(),"onkeydown":(k) => { if (k.key === "Enter") rename(); if (k.key === "Escape") { k.stopPropagation(); renaming = ""; } },});/*Ωignore_startΩ*/() => renameTo = __sveltets_2_any(null);/*Ωignore_endΩ*/}
            }else{ { svelteHTML.createElement("span", { "class":`name`,});e.name; }if(e.link){ { svelteHTML.createElement("span", { "class":`chip`,});  }}}
           }
           { svelteHTML.createElement("span", { "class":`mono dim`,});size(e.size, e.dir); } { svelteHTML.createElement("span", { "class":`dim`,});e.modified; } { svelteHTML.createElement("span", { "class":`mono dim`,});e.perm; }
         }
      }
        if(!busy){ { svelteHTML.createElement("p", { "class":`empty`,});error ? "" : "This folder is empty."; }}
      
     }
     { svelteHTML.createElement("footer", {});busy || `${shown.length} item${shown.length === 1 ? "" : "s"}`; { svelteHTML.createElement("span", { "class":`sp`,}); } { svelteHTML.createElement("span", {});            } }
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const FilesPanel__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type FilesPanel__SvelteComponent_ = ReturnType<typeof FilesPanel__SvelteComponent_>;
/*Ωignore_endΩ*/export default FilesPanel__SvelteComponent_;