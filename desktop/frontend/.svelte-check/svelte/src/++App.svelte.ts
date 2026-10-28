///<reference types="svelte" />
;
import { onMount } from "svelte";
import { openSession, type Session } from "./terminal";
import { runBench } from "./bench";
import { TerminalService } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";
function $$render() {

  
  
  
  

  let host: HTMLElement;
  let target = $state("local");
  let session: Session | undefined;
  let error = $state("");

  async function connect() {
    error = "";
    session?.dispose();
    host.replaceChildren();
    try {
      session = await openSession(host, target);
      session.term.focus();
    } catch (e) {
      error = String(e);
    }
  }

  onMount(async () => {
    const mode = await TerminalService.BenchMode();
    if (mode) await runBench(host, mode);
    else await connect();
  });
;
async () => {

 { svelteHTML.createElement("div", { "class":`bar`,});
   { svelteHTML.createElement("input", {      "bind:value":target,"placeholder":`local or profile name`,"onkeydown":(e) => e.key === "Enter" && connect(),});/*Ωignore_startΩ*/() => target = __sveltets_2_any(null);/*Ωignore_endΩ*/}
   { svelteHTML.createElement("button", { "onclick":connect,});  }
   { svelteHTML.createElement("span", { "class":`err`,});error; }
 }
 { const $$_div0 = svelteHTML.createElement("div", {  "class":`term`,});host = $$_div0; }


};
return { props: {} as Record<string, never>, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const App__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type App__SvelteComponent_ = ReturnType<typeof App__SvelteComponent_>;
/*Ωignore_endΩ*/export default App__SvelteComponent_;