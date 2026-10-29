///<reference types="svelte" />
;
import { ProfilesService, type Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
;

  // The form always edits real arrays, even though the generated type allows null.
  type Editable = Profile & { tags: string[]; tunnels: NonNullable<Profile["tunnels"]> };;
;type $$ComponentProps =  {
    initial: Profile;
    oldName: string;
    profiles: Profile[];
    folders: string[];
    onsave: (p: Profile) => Promise<void>;
    ondelete: () => void;
    oncancel: () => void;
  };function $$render() {

  

  let { initial, oldName, profiles, folders, onsave, ondelete, oncancel }:/*Ωignore_startΩ*/$$ComponentProps/*Ωignore_endΩ*/ = $props();

  const isNew = oldName === "";
  // The form edits its own copy; `initial` is only read once.
  // svelte-ignore state_referenced_locally
  let p = $state<Editable>({ ...structuredClone($state.snapshot(initial)), tags: [...(initial.tags ?? [])], tunnels: [...(initial.tunnels ?? [])] });
  let tagInput = $state("");
  let error = $state("");
  let busy = $state(false);
  let preview = $state({ config: "", problem: "" });

  const others = $derived(profiles.map((x) => x.name).filter((n) => n !== oldName && n !== p.name));
  const chain = $derived.by(() => {
    const out: string[] = [];
    let cur = p.jump;
    while (cur && !out.includes(cur) && out.length < 8) {
      out.push(cur);
      cur = profiles.find((x) => x.name === cur)?.jump ?? "";
    }
    return out;
  });

  // Live preview straight from the real generator, so it is exactly what ssh will read.
  $effect(() => {
    const snap = $state.snapshot(p) as Profile;
    snap.port = Number(snap.port);
    const t = setTimeout(async () => { preview = await ProfilesService.Preview(oldName, snap); }, 120);
    return () => clearTimeout(t);
  });

  function addTag() {
    for (const t of tagInput.split(",").map((s) => s.trim()).filter(Boolean)) if (!p.tags.includes(t)) p.tags.push(t);
    tagInput = "";
  }
  async function browse() {
    const f = await ProfilesService.PickKeyFile().catch(() => "");
    if (f) p.key = f;
  }
  async function submit(e?: Event) {
    e?.preventDefault();
    if (tagInput.trim()) addTag();
    busy = true;
    error = "";
    try {
      await onsave({ ...$state.snapshot(p), port: Number(p.port), key: p.auth === "key" ? p.key : "" } as Profile);
    } catch (err) {
      error = (err as Error).message ?? String(err);
    } finally {
      busy = false;
    }
  }
  function keydown(e: KeyboardEvent) {
    if (e.key === "Escape") oncancel();
    else if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) submit();
  }
  const addTunnel = () => p.tunnels.push({ type: "local", listen: "", target: "" });
  const auths = [
    { id: "agent", label: "SSH agent", icon: "M12 3l8 3v6c0 5-3.5 8-8 9-4.5-1-8-4-8-9V6z" },
    { id: "key", label: "Key file", icon: "M11 12l9-9M16 7l3 3M8 11a4 4 0 1 0 0 8 4 4 0 0 0 0-8z" },
    { id: "password", label: "Password", icon: "M5 11h14v9H5zM8 11V8a4 4 0 0 1 8 0v3" },
  ];
;
async () => {

 { svelteHTML.createElement("div", {     "class":`scrim`,"role":`presentation`,"onkeydown":keydown,});
   { svelteHTML.createElement("form", {     "role":`dialog`,"aria-label":isNew ? "New connection" : "Edit connection","onsubmit":submit,});
     { svelteHTML.createElement("header", {});
       { svelteHTML.createElement("div", { "class":`tile`,}); { svelteHTML.createElement("svg", {                 "width":`18`,"height":`18`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":`M4 17l6-6-6-6M12 19h8`,});} } }
       { svelteHTML.createElement("div", { "class":`titles`,}); { svelteHTML.createElement("h2", {});isNew ? "New connection" : "Edit connection"; }if(!isNew){ { svelteHTML.createElement("span", {});oldName;initial.folder ? " · " + initial.folder : ""; }} }
       { svelteHTML.createElement("button", {       "type":`button`,"class":`close`,"aria-label":`Close`,"onclick":oncancel,});  }
     }

     { svelteHTML.createElement("div", { "class":`body`,});
       { svelteHTML.createElement("div", { "class":`left`,});
         { svelteHTML.createElement("section", {});
           { svelteHTML.createElement("div", { "class":`label`,});  }
           { svelteHTML.createElement("div", { "class":`g2`,});
             { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {      "class":`mono`,"bind:value":p.name,"required":true,"placeholder":`web-1`,});/*Ωignore_startΩ*/() => p.name = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
             { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {        "class":`mono`,"bind:value":p.folder,"list":`folders`,"placeholder":`prod/eu`,});/*Ωignore_startΩ*/() => p.folder = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
           }
           { svelteHTML.createElement("div", { "class":`g3`,});
             { svelteHTML.createElement("label", {});  { svelteHTML.createElement("input", {      "class":`mono`,"bind:value":p.host,"required":true,"placeholder":`10.0.0.5 or example.com`,});/*Ωignore_startΩ*/() => p.host = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
             { svelteHTML.createElement("label", {});   { svelteHTML.createElement("input", {          "class":`mono`,"type":`number`,"bind:value":p.port,"min":`1`,"max":`65535`,"required":true,});/*Ωignore_startΩ*/() => p.port = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
             { svelteHTML.createElement("label", {});   { svelteHTML.createElement("input", {    "class":`mono`,"bind:value":p.user,"required":true,});/*Ωignore_startΩ*/() => p.user = __sveltets_2_any(null);/*Ωignore_endΩ*/} }
           }
         }

         { svelteHTML.createElement("section", {});
           { svelteHTML.createElement("div", { "class":`label`,});  }
           { svelteHTML.createElement("div", {     "class":`seg`,"role":`radiogroup`,"aria-label":`Authentication`,});
              for(let a of __sveltets_2_ensureArray(auths)){
               { svelteHTML.createElement("button", {        "type":`button`,"role":`radio`,"aria-checked":p.auth === a.id,"onclick":() => (p.auth = a.id),});p.auth === a.id;
                 { svelteHTML.createElement("svg", {                 "width":`16`,"height":`16`,"viewBox":`0 0 24 24`,"fill":`none`,"stroke":`currentColor`,"stroke-width":`2`,"stroke-linecap":`round`,"stroke-linejoin":`round`,"aria-hidden":`true`,}); { svelteHTML.createElement("path", {  "d":a.icon,});} }a.label;
               }
            }
           }
          if(p.auth === "key"){
             { svelteHTML.createElement("div", { "class":`keyrow`,}); { svelteHTML.createElement("input", {        "class":`mono`,"bind:value":p.key,"required":true,"placeholder":`id_ed25519 or ~/.ssh/id_ed25519`,"aria-label":`Key file`,});/*Ωignore_startΩ*/() => p.key = __sveltets_2_any(null);/*Ωignore_endΩ*/} { svelteHTML.createElement("button", {     "type":`button`,"class":`btn`,"onclick":browse,});  } }
          }
           { svelteHTML.createElement("p", { "class":`note`,});p.auth === "agent" ? "Uses keys loaded in your ssh-agent." : p.auth === "key" ? "Passphrases are asked by ssh and never stored." : "ssh asks for the password in the terminal. It is never stored."; }
         }

         { svelteHTML.createElement("section", {});
           { svelteHTML.createElement("div", { "class":`label`,});  }
           { svelteHTML.createElement("div", { "class":`g2`,});
             { svelteHTML.createElement("label", {}); 
               { svelteHTML.createElement("select", { "bind:value":p.jump,});/*Ωignore_startΩ*/() => p.jump = __sveltets_2_any(null);/*Ωignore_endΩ*/ { svelteHTML.createElement("option", {"value":"",});  }  for(let o of __sveltets_2_ensureArray(others)){ { svelteHTML.createElement("option", { "value":o,});o; }} }
             }
             { svelteHTML.createElement("div", { "class":`field`,});
               { svelteHTML.createElement("div", { "class":`tags`,});
                   for(let t of __sveltets_2_ensureArray(p.tags)){let i = 1; { svelteHTML.createElement("span", { "class":`tag`,});t; { svelteHTML.createElement("button", {     "type":`button`,"aria-label":`Remove ${t}`,"onclick":() => p.tags.splice(i, 1),});  } }}
                 { svelteHTML.createElement("input", {          "bind:value":tagInput,"placeholder":p.tags.length ? "" : "add tag","aria-label":`Add tag`,"onkeydown":(e) => { if (e.key === "Enter" || e.key === ",") { e.preventDefault(); e.stopPropagation(); addTag(); } },"onblur":addTag,});/*Ωignore_startΩ*/() => tagInput = __sveltets_2_any(null);/*Ωignore_endΩ*/}
               }
             }
           }
           { svelteHTML.createElement("div", {   "class":`route mono`,"aria-label":`Route`,});
             { svelteHTML.createElement("span", { "class":`rl`,});  } { svelteHTML.createElement("span", { "class":`hop`,});  }
              for(let h of __sveltets_2_ensureArray(chain)){ { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", { "class":`hop`,});h; }}
             { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", { "class":`hop last`,});p.name || "new"; }
           }
           { svelteHTML.createElement("div", { "class":`switch`,});
             { svelteHTML.createElement("div", {}); { svelteHTML.createElement("b", {});   } { svelteHTML.createElement("span", {});            } }
             { svelteHTML.createElement("button", {          "type":`button`,"role":`switch`,"aria-checked":p.forwardAgent,"aria-label":`Forward SSH agent`,"onclick":() => (p.forwardAgent = !p.forwardAgent),});p.forwardAgent; { svelteHTML.createElement("i", {}); } }
           }
         }

         { svelteHTML.createElement("section", { "class":`last`,});
           { svelteHTML.createElement("div", { "class":`row`,}); { svelteHTML.createElement("span", { "class":`label`,});  } { svelteHTML.createElement("button", {     "type":`button`,"class":`link`,"onclick":addTunnel,});   } }
          if(p.tunnels.length){
             { svelteHTML.createElement("div", { "class":`tun head`,}); { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {});  } { svelteHTML.createElement("span", {}); } }
               for(let t of __sveltets_2_ensureArray(p.tunnels)){let i = 1;
               { svelteHTML.createElement("div", { "class":`tun`,});
                 { svelteHTML.createElement("select", {   "bind:value":t.type,"aria-label":`Tunnel type`,});/*Ωignore_startΩ*/() => t.type = __sveltets_2_any(null);/*Ωignore_endΩ*/ { svelteHTML.createElement("option", { "value":`local`,});  } { svelteHTML.createElement("option", { "value":`remote`,});  } { svelteHTML.createElement("option", { "value":`dynamic`,});  } }
                  { svelteHTML.createElement("input", {        "class":`mono`,"bind:value":t.listen,"placeholder":t.type === "remote" ? "remote port" : "local port","aria-label":`Listen`,"required":true,});/*Ωignore_startΩ*/() => t.listen = __sveltets_2_any(null);/*Ωignore_endΩ*/}
                if(t.type !== "dynamic"){  { svelteHTML.createElement("input", {        "class":`mono`,"bind:value":t.target,"placeholder":`host:port`,"aria-label":`Target`,"required":true,});/*Ωignore_startΩ*/() => t.target = __sveltets_2_any(null);/*Ωignore_endΩ*/}}else{ { svelteHTML.createElement("span", { "class":`na`,});  }}
                 { svelteHTML.createElement("button", {     "type":`button`,"aria-label":`Remove tunnel`,"onclick":() => p.tunnels.splice(i, 1),});  }
               }
            }
          }
         }
       }

       { svelteHTML.createElement("aside", {});
         { svelteHTML.createElement("div", { "class":`phead`,}); { svelteHTML.createElement("span", { "class":`label`,});   } { svelteHTML.createElement("span", {});     } }
         { svelteHTML.createElement("pre", { "class":`mono`,});  for(let line of __sveltets_2_ensureArray(preview.config.split("\n"))){const m = line.match(/^(\s*)(\S+)(.*)$/);if(m){m[1]; { svelteHTML.createElement("span", { "class":m[1] ? "k2" : "k1",});m[2]; }m[3];"\n";}} }
         { svelteHTML.createElement("div", { "class":`state`,}); { svelteHTML.createElement("span", {   "class":`dot`,});!preview.problem;!!preview.problem; }preview.problem || "Valid. Jump chain resolves."; }
         { svelteHTML.createElement("div", { "class":`cli`,}); { svelteHTML.createElement("span", {});    } { svelteHTML.createElement("div", { "class":`mono`,}); { svelteHTML.createElement("i", {});  }  p.name || "name"; } }
       }
     }

    if(error){ { svelteHTML.createElement("p", {   "class":`error`,"role":`alert`,});error; }}
     { svelteHTML.createElement("footer", {});
      if(!isNew){ { svelteHTML.createElement("button", {     "type":`button`,"class":`btn danger`,"onclick":ondelete,});  }}
       { svelteHTML.createElement("span", { "class":`esc`,});   }
       { svelteHTML.createElement("button", {     "type":`button`,"class":`btn`,"onclick":oncancel,});  }
       { svelteHTML.createElement("button", {   "class":`btn primary`,"disabled":busy,});isNew ? "Add connection" : "Save changes"; { svelteHTML.createElement("span", { "class":`mono`,});  } }
     }
     { svelteHTML.createElement("datalist", { "id":`folders`,});  for(let f of __sveltets_2_ensureArray(folders)){ { svelteHTML.createElement("option", { "value":f,}); }} }
   }
 }


};
return { props: {} as any as $$ComponentProps, exports: {}, bindings: __sveltets_$$bindings(''), slots: {}, events: {} }}
const ProfileForm__SvelteComponent_ = __sveltets_2_fn_component($$render());
/*Ωignore_startΩ*/type ProfileForm__SvelteComponent_ = ReturnType<typeof ProfileForm__SvelteComponent_>;
/*Ωignore_endΩ*/export default ProfileForm__SvelteComponent_;