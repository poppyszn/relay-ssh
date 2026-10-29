///<reference types="svelte" />
;
import { ToolsService, type HealthCheck, type ImportReport, type KeyInfo } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
;

  type Tab = "health" | "backup" | "import" | "keys";;
;type $$ComponentProps =  { initial?: Tab; onclose: () => void; onchanged: () => void; ongenerate: () => void };function $$render() {

  

  let { initial = "health", onclose, onchanged, ongenerate }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  // svelte-ignore state_referenced_locally
  let tab = $state<Tab>(initial);
  const tabs: { id: Tab; label: string }[] = [
    { id: "health", label: "Health" },
    { id: "backup", label: "Backup & restore" },
    { id: "import", label: "Import" },
    { id: "keys", label: "Keys" },
  ];

  const msg = (e: unknown) => (e as Error)?.message ?? String(e);

  // Health
  let checks = $state<HealthCheck[]>([]);
  let fixErrors = $state<string[]>([]);
  let busy = $state(false);
  const problems = $derived(checks.filter((c) => c.status !== "ok").length);
  async function recheck() { busy = true; try { checks = (await ToolsService.Health()) ?? []; } catch (e) { fixErrors = [msg(e)]; } finally { busy = false; } }
  async function fix() {
    busy = true;
    try {
      const r = await ToolsService.FixHealth();
      checks = r.checks ?? [];
      fixErrors = r.errors ?? [];
      onchanged();
    } catch (e) { fixErrors = [msg(e)]; } finally { busy = false; }
  }

  // Backup and restore
  let pass = $state("");
  let pass2 = $state("");
  let backupNote = $state<{ ok: boolean; text: string } | null>(null);
  async function backup() {
    backupNote = null;
    if (pass.length < 1) return (backupNote = { ok: false, text: "Enter a passphrase." });
    if (pass !== pass2) return (backupNote = { ok: false, text: "The passphrases do not match." });
    try {
      const path = await ToolsService.PickBackupPath();
      if (!path) return;
      await ToolsService.Backup(path, pass);
      backupNote = { ok: true, text: `Backup saved to ${path}` };
      pass = pass2 = "";
    } catch (e) { backupNote = { ok: false, text: msg(e) }; }
  }
  let restorePath = $state("");
  let restorePass = $state("");
  let needsPass = $state(false);
  let confirmRestore = $state(false);
  let restoreNote = $state<{ ok: boolean; text: string } | null>(null);
  async function chooseRestore() {
    restoreNote = null; confirmRestore = false;
    try {
      const path = await ToolsService.PickRestorePath();
      if (!path) return;
      restorePath = path;
      needsPass = await ToolsService.NeedsPassphrase(path);
    } catch (e) { restoreNote = { ok: false, text: msg(e) }; }
  }
  async function restore() {
    if (!confirmRestore) return (confirmRestore = true);
    confirmRestore = false;
    try {
      await ToolsService.Restore(restorePath, restorePass);
      restoreNote = { ok: true, text: "Backup restored. Your connections and keys are back." };
      restorePass = "";
      onchanged();
    } catch (e) { restoreNote = { ok: false, text: msg(e) }; }
  }

  // Import
  let report = $state<ImportReport | null>(null);
  const added = $derived(report?.added ?? []);
  const skipped = $derived(report?.skipped ?? []);
  let importError = $state("");
  async function doImport() {
    importError = ""; report = null; busy = true;
    try { report = await ToolsService.ImportSSHConfig(); if ((report.added ?? []).length) onchanged(); } catch (e) { importError = msg(e); } finally { busy = false; }
  }

  // Keys
  let keyList = $state<KeyInfo[]>([]);
  let keyError = $state("");
  async function loadKeys() { try { keyList = (await ToolsService.Keys()) ?? []; } catch (e) { keyError = msg(e); } }
  async function adopt(path: string) {
    keyError = "";
    try { await ToolsService.AdoptKey(path); onchanged(); await loadKeys(); } catch (e) { keyError = msg(e); }
  }
  async function adoptOther() {
    try { const p = await ToolsService.PickKeyToAdopt(); if (p) await adopt(p); } catch (e) { keyError = msg(e); }
  }

  $effect(() => { if (tab === "health") recheck(); else if (tab === "keys") loadKeys(); });
;
async () => {

 { svelteHTML.createElement("div", {       "class":`scrim`,"role":`presentation`,"onclick":onclose,"onkeydown":(e) => e.key === "Escape" && onclose(),});
   { svelteHTML.createElement("div", {         "class":`panel`,"role":`dialog`,"aria-label":`Tools`,"tabindex":-1,"onclick":(e) => e.stopPropagation(),});
     { svelteHTML.createElement("header", {}); { svelteHTML.createElement("h2", {});  } { svelteHTML.createElement("button", {     "class":`x`,"aria-label":`Close`,"onclick":onclose,});  } }
     { svelteHTML.createElement("div", { "class":`body`,});
       { svelteHTML.createElement("nav", { "aria-label":`Tools`,});
          for(let t of __sveltets_2_ensureArray(tabs)){ { svelteHTML.createElement("button", {   "onclick":() => (tab = t.id),});tab === t.id;t.label; }}
       }
       { svelteHTML.createElement("div", { "class":`content`,});
        if(tab === "health"){
           { svelteHTML.createElement("div", { "class":`head`,}); { svelteHTML.createElement("div", {}); { svelteHTML.createElement("h3", {});  } { svelteHTML.createElement("p", {});            } }
             { svelteHTML.createElement("div", { "class":`actions`,}); { svelteHTML.createElement("button", {     "class":`btn`,"onclick":recheck,"disabled":busy,});  }if(problems){ { svelteHTML.createElement("button", {     "class":`btn primary`,"onclick":fix,"disabled":busy,}); problems; problems === 1 ? "" : "s"; }} } }
           { svelteHTML.createElement("ul", { "class":`checks`,});
              for(let c of __sveltets_2_ensureArray(checks)){
               { svelteHTML.createElement("li", {}); { svelteHTML.createElement("span", {    "class":`dot`,});c.status === "ok";c.status === "warn";c.status === "fail"; } { svelteHTML.createElement("b", {});c.name; } { svelteHTML.createElement("span", { "class":`detail`,});c.detail; } }
            }
           }
            for(let e of __sveltets_2_ensureArray(fixErrors)){ { svelteHTML.createElement("p", { "class":`note bad`,});e; }}
          if(checks.length && !problems){ { svelteHTML.createElement("p", { "class":`note good`,});   }}
        } else if (tab === "backup"){
           { svelteHTML.createElement("h3", {});  }
           { svelteHTML.createElement("p", {});                         }
           { svelteHTML.createElement("div", { "class":`fields`,}); { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {      "type":`password`,"bind:value":pass,"autocomplete":`off`,});/*Ωignore_startΩ*/() => pass = __sveltets_2_any(null);/*Ωignore_endΩ*/} } { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {      "type":`password`,"bind:value":pass2,"autocomplete":`off`,});/*Ωignore_startΩ*/() => pass2 = __sveltets_2_any(null);/*Ωignore_endΩ*/} } }
           { svelteHTML.createElement("div", { "class":`actions`,}); { svelteHTML.createElement("button", {   "class":`btn primary`,"onclick":backup,});     } }
          if(backupNote){ { svelteHTML.createElement("p", {   "class":`note`,});backupNote.ok;!backupNote.ok;backupNote.text; }}

           { svelteHTML.createElement("h3", { "class":`gap`,});  }
           { svelteHTML.createElement("p", {});             }
           { svelteHTML.createElement("div", { "class":`actions`,}); { svelteHTML.createElement("button", {   "class":`btn`,"onclick":chooseRestore,});  } { svelteHTML.createElement("span", { "class":`path mono`,});restorePath || "No file chosen"; } }
          if(restorePath){
            if(needsPass){ { svelteHTML.createElement("div", { "class":`fields one`,}); { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {      "type":`password`,"bind:value":restorePass,"autocomplete":`off`,});/*Ωignore_startΩ*/() => restorePass = __sveltets_2_any(null);/*Ωignore_endΩ*/} } }}
             { svelteHTML.createElement("div", { "class":`actions`,}); { svelteHTML.createElement("button", {    "class":`btn`,"onclick":restore,});confirmRestore;confirmRestore ? "Click again to replace current data" : "Restore"; } }
          }
          if(restoreNote){ { svelteHTML.createElement("p", {   "class":`note`,});restoreNote.ok;!restoreNote.ok;restoreNote.text; }}
        } else if (tab === "import"){
           { svelteHTML.createElement("h3", {});   }
           { svelteHTML.createElement("p", {});                                            }
           { svelteHTML.createElement("div", { "class":`actions`,}); { svelteHTML.createElement("button", {     "class":`btn primary`,"onclick":doImport,"disabled":busy,});  } }
          if(importError){ { svelteHTML.createElement("p", { "class":`note bad`,});importError; }}
          if(report){
             { svelteHTML.createElement("p", {  "class":`note`,});added.length > 0;added.length ? `Imported ${added.length}: ${added.join(", ")}` : "No new connections found."; }
              for(let s of __sveltets_2_ensureArray(skipped)){ { svelteHTML.createElement("p", { "class":`note bad`,}); s; }}
          }
        }else{
           { svelteHTML.createElement("div", { "class":`head`,}); { svelteHTML.createElement("div", {}); { svelteHTML.createElement("h3", {});  } { svelteHTML.createElement("p", {});                      } }
             { svelteHTML.createElement("div", { "class":`actions`,}); { svelteHTML.createElement("button", {   "class":`btn`,"onclick":adoptOther,});   } { svelteHTML.createElement("button", {   "class":`btn`,"onclick":ongenerate,});  } } }
          if(keyError){ { svelteHTML.createElement("p", { "class":`note bad`,});keyError; }}
           { svelteHTML.createElement("ul", { "class":`keys`,});
              for(let k of __sveltets_2_ensureArray(keyList)){
               { svelteHTML.createElement("li", {}); { svelteHTML.createElement("div", { "class":`kname`,}); { svelteHTML.createElement("b", { "class":`mono`,});k.name; }if(k.adopted){ { svelteHTML.createElement("span", { "class":`chip ok`,});  }}else{ { svelteHTML.createElement("span", { "class":`chip`,});   }} }
                 { svelteHTML.createElement("div", { "class":`kmeta mono`,});k.path; }
                 { svelteHTML.createElement("div", { "class":`kuse`,});k.usedBy?.length ? "Used by " + k.usedBy.join(", ") : "Not used by any connection"; }
                if(!k.adopted){ { svelteHTML.createElement("button", {   "class":`btn adopt`,"onclick":() => adopt(k.path),});  }} }
            } { svelteHTML.createElement("li", { "class":`empty`,});      }
           }
        }
       }
     }
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const ToolsPanel__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type ToolsPanel__SvelteComponent_ = ReturnType<typeof ToolsPanel__SvelteComponent_>;
/*Ωignore_endΩ*/export default ToolsPanel__SvelteComponent_;