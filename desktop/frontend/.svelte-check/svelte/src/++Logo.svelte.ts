///<reference types="svelte" />
;
;type $$ComponentProps =  { size?: number };function $$render() {

  // Relay's mark: a terminal chevron cut out of a block, with a lighter block behind it.
  // The geometry is the icon's 2000-unit grid (see desktop/build/icon/make-icon.swift).
  let { size = 64 }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();
;
async () => {

 { svelteHTML.createElement("svg", {         "width":size,"height":size,"viewBox":`500 500 1000 1000`,"role":`img`,"aria-label":`Relay`,});
   { svelteHTML.createElement("path", {      "fill":`#318EDC`,"fill-rule":`evenodd`,"d":`M726 500H1250V1500H750V1250H500V726Z M750 750H904L1156 1001L904 1250H750L1001 1001Z`,});}
   { svelteHTML.createElement("polygon", {    "fill":`#7AB2EE`,"points":`1250,750 1500,750 1500,1250 1250,1500`,});}
 }
};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const Logo__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type Logo__SvelteComponent_ = ReturnType<typeof Logo__SvelteComponent_>;
/*Ωignore_endΩ*/export default Logo__SvelteComponent_;