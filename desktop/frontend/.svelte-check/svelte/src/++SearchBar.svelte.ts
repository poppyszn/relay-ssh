///<reference types="svelte" />
;
import { tick } from "svelte";
import type { Session } from "./terminal";

;type $$ComponentProps =  { session: Session; onclose: () => void };function $$render() {

  
  

  let { session, onclose }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  let query = $state("");
  let caseSensitive = $state(false);
  let count = $state({ index: -1, total: 0 });
  let input: HTMLInputElement;

  // Match highlights: yellow like the design, brighter for the current match.
  const decorations = {
    matchBackground: "#5d4e2b", matchBorder: "#8a7440", matchOverviewRuler: "#e6c07b",
    activeMatchBackground: "#e6c07b", activeMatchBorder: "#f3d398", activeMatchColorOverviewRuler: "#f3d398",
  };
  const opts = () => ({ caseSensitive, decorations });

  $effect(() => {
    const sub = session.search.onDidChangeResults((r) => (count = { index: r.resultIndex, total: r.resultCount }));
    tick().then(() => { input?.focus(); input?.select(); });
    return () => { sub.dispose(); session.search.clearDecorations(); };
  });
  $effect(() => {
    // incremental: re-run whenever the query or case option changes
    caseSensitive;
    if (query) session.search.findNext(query, { ...opts(), incremental: true });
    else { session.search.clearDecorations(); count = { index: -1, total: 0 }; }
  });

  const next = () => query && session.search.findNext(query, opts());
  const prev = () => query && session.search.findPrevious(query, opts());
  function key(e: KeyboardEvent) {
    if (e.key === "Escape") { e.stopPropagation(); onclose(); }
    else if (e.key === "Enter") { e.preventDefault(); e.shiftKey ? prev() : next(); }
  }
  const label = $derived(!query ? "" : count.total === 0 ? "No results" : count.index < 0 ? `${count.total}+` : `${count.index + 1} of ${count.total}`);
;
async () => {

 { svelteHTML.createElement("div", {     "class":`bar`,"role":`search`,"onkeydown":key,});
   { const $$_input1 = svelteHTML.createElement("input", {          "bind:value":query,"class":`mono`,"placeholder":`Find`,"aria-label":`Find in scrollback`,});input = $$_input1;/*Ωignore_startΩ*/() => query = __sveltets_2_any(null);/*Ωignore_endΩ*/}
   { svelteHTML.createElement("span", {  "class":`count`,});query && count.total === 0;label; }
   { svelteHTML.createElement("button", {     "aria-label":`Previous match`,"title":`Previous (⇧↵)`,"onclick":prev,});
     { svelteHTML.createElement("svg", {                 "width":`14`,"height":`14`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M6 15l6-6 6 6`,});} }
   }
   { svelteHTML.createElement("button", {     "aria-label":`Next match`,"title":`Next (↵)`,"onclick":next,});
     { svelteHTML.createElement("svg", {                 "width":`14`,"height":`14`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M6 9l6 6 6-6`,});} }
   }
   { svelteHTML.createElement("button", {          "class":`aa`,"aria-label":`Match case`,"aria-pressed":caseSensitive,"title":`Match case`,"onclick":() => (caseSensitive = !caseSensitive),});caseSensitive;  }
   { svelteHTML.createElement("button", {     "aria-label":`Close search`,"title":`Close (Esc)`,"onclick":onclose,});  }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const SearchBar__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type SearchBar__SvelteComponent_ = ReturnType<typeof SearchBar__SvelteComponent_>;
/*Ωignore_endΩ*/export default SearchBar__SvelteComponent_;