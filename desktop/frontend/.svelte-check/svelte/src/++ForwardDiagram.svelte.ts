///<reference types="svelte" />
;
;type $$ComponentProps =  { kind: "local" | "remote" | "dynamic"; listen: string; dest: string; server: string };function $$render() {

  // A picture of one forward: who listens where, and which way connections travel.
  let { kind, listen, dest, server }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  const port = $derived(listen.trim() || "port");
  const target = $derived(dest.trim() || "host:port");
  const onServer = $derived(/^(localhost|127\.0\.0\.1|\[?::1\]?):/.test(dest.trim()));

  // Left = your Mac, middle = the server, right = the far end. For a remote
  // forward connections travel right to left, so the arrows flip.
  const flip = $derived(kind === "remote");
  const nodes = $derived(
    kind === "local"
      ? [
          { icon: "mac", title: "This Mac", sub: `localhost:${port}`, role: "listens here" },
          { icon: "server", title: server, sub: "SSH server", role: "" },
          { icon: "target", title: onServer ? `${server} itself` : "Destination", sub: onServer ? target : target, role: "connects to" },
        ]
      : kind === "remote"
        ? [
            { icon: "mac", title: "This Mac", sub: target, role: "receives the connection" },
            { icon: "server", title: server, sub: `listens on :${port}`, role: "" },
            { icon: "people", title: "Anyone on the server", sub: `connects to :${port}`, role: "" },
          ]
        : [
            { icon: "mac", title: "This Mac", sub: `SOCKS proxy on localhost:${port}`, role: "apps use this proxy" },
            { icon: "server", title: server, sub: "SSH server", role: "" },
            { icon: "globe", title: "Any website or host", sub: "chosen by each app", role: "reached from the server" },
          ],
  );
  const caption = $derived(
    kind === "local" ? `Connecting to localhost:${port} on this Mac reaches ${target}${onServer ? ` on ${server}` : ` from ${server}`}.`
    : kind === "remote" ? `Connecting to port ${port} on ${server} comes back to ${target} on this Mac.`
    : `Point a browser or app at the SOCKS proxy on localhost:${port} and its traffic leaves from ${server}.`,
  );
;
async () => {

 { svelteHTML.createElement("figure", {   "class":`diagram`,"aria-label":`How this forward works`,});
   { svelteHTML.createElement("div", { "class":`row`,});
       for(let n of __sveltets_2_ensureArray(nodes)){let i = 1;
       { svelteHTML.createElement("div", { "class":`node`,});
         { svelteHTML.createElement("svg", {                   "class":`icon`,"width":`34`,"height":`34`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`1.6`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,});
          if(n.icon === "mac"){ { svelteHTML.createElement("rect", {          "x":`4`,"y":`5`,"width":`16`,"height":`11`,"rx":`1.5`,});} { svelteHTML.createElement("path", {  "d":`M2 19h20`,});}
          } else if (n.icon === "server"){ { svelteHTML.createElement("rect", {          "x":`4`,"y":`4`,"width":`16`,"height":`6`,"rx":`1.2`,});} { svelteHTML.createElement("rect", {          "x":`4`,"y":`14`,"width":`16`,"height":`6`,"rx":`1.2`,});} { svelteHTML.createElement("path", {  "d":`M8 7h.01M8 17h.01`,});}
          } else if (n.icon === "globe"){ { svelteHTML.createElement("circle", {      "cx":`12`,"cy":`12`,"r":`9`,});} { svelteHTML.createElement("path", {  "d":`M3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18`,});}
          } else if (n.icon === "people"){ { svelteHTML.createElement("circle", {      "cx":`9`,"cy":`8`,"r":`3`,});} { svelteHTML.createElement("path", {  "d":`M3 20c0-3.3 2.7-5 6-5s6 1.7 6 5M17 7a3 3 0 0 1 0 6M18 15c2 .5 3 2 3 5`,});}
          }else{ { svelteHTML.createElement("circle", {      "cx":`12`,"cy":`12`,"r":`8`,});} { svelteHTML.createElement("circle", {      "cx":`12`,"cy":`12`,"r":`3`,});}}
         }
         { svelteHTML.createElement("div", { "class":`t`,});n.title; }
         { svelteHTML.createElement("div", { "class":`s mono`,});n.sub; }
        if(n.role){ { svelteHTML.createElement("div", { "class":`r`,});n.role; }}
       }
      if(i < nodes.length - 1){
         { svelteHTML.createElement("div", {   "class":`link`,});flip;i === 0;
           { svelteHTML.createElement("svg", {     "viewBox":`0 0 90 14`,"preserveAspectRatio":`none`,"aria-hidden":`true`,});
             { svelteHTML.createElement("line", {          "x1":`2`,"y1":`7`,"x2":`78`,"y2":`7`,"class":`wire`,});}
             { svelteHTML.createElement("path", {    "d":flip ? "M12 2L2 7l10 5" : "M78 2l10 5-10 5","class":`head`,});}
           }
           { svelteHTML.createElement("span", {});i === 0 ? "encrypted SSH tunnel" : kind === "dynamic" ? "opens connections" : flip ? "" : "connects"; }
         }
      }
    }
   }
   { svelteHTML.createElement("figcaption", {});caption; }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const ForwardDiagram__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type ForwardDiagram__SvelteComponent_ = ReturnType<typeof ForwardDiagram__SvelteComponent_>;
/*Ωignore_endΩ*/export default ForwardDiagram__SvelteComponent_;