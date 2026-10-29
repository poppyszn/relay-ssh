<script lang="ts">
  // A picture of one forward: who listens where, and which way connections travel.
  let { kind, listen, dest, server }: { kind: "local" | "remote" | "dynamic"; listen: string; dest: string; server: string } = $props();

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
</script>

<figure class="diagram" aria-label="How this forward works">
  <div class="row">
    {#each nodes as n, i}
      <div class="node">
        <span class="icon"><svg class="glyph" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
          {#if n.icon === "mac"}<rect x="4" y="5" width="16" height="11" rx="1.5" /><path d="M2 19h20" />
          {:else if n.icon === "server"}<rect x="4" y="4" width="16" height="6" rx="1.2" /><rect x="4" y="14" width="16" height="6" rx="1.2" /><path d="M8 7h.01M8 17h.01" />
          {:else if n.icon === "globe"}<circle cx="12" cy="12" r="9" /><path d="M3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18" />
          {:else if n.icon === "people"}<circle cx="9" cy="8" r="3" /><path d="M3 20c0-3.3 2.7-5 6-5s6 1.7 6 5M17 7a3 3 0 0 1 0 6M18 15c2 .5 3 2 3 5" />
          {:else}<circle cx="12" cy="12" r="8" /><circle cx="12" cy="12" r="3" />{/if}
        </svg></span>
        <div class="t">{n.title}</div>
        <div class="s mono">{n.sub}</div>
        {#if n.role}<div class="r">{n.role}</div>{/if}
      </div>
      {#if i < nodes.length - 1}
        <div class="link" class:flip class:tunnel={i === 0}>
          <svg viewBox="0 0 90 14" preserveAspectRatio="none" aria-hidden="true">
            <line x1="2" y1="7" x2="78" y2="7" class="wire" />
            <path d={flip ? "M12 2L2 7l10 5" : "M78 2l10 5-10 5"} class="head" />
          </svg>
          <span>{i === 0 ? "encrypted SSH tunnel" : kind === "dynamic" ? "opens connections" : flip ? "" : "connects"}</span>
        </div>
      {/if}
    {/each}
  </div>
  <figcaption>{caption}</figcaption>
</figure>

<style>
  .diagram { --stroke: 1.75px; margin: 0; padding: var(--s4) var(--s5) var(--s3); background: var(--rail); border: 1px solid var(--line); border-radius: var(--r-md); }
  .row { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--s2); }
  .node { width: 168px; flex: none; display: grid; justify-items: center; gap: var(--s1); text-align: center; }
  .icon { display: grid; place-items: center; width: 40px; height: 40px; color: var(--accent-text); background: var(--accent-soft); border: 1px solid var(--accent-edge); border-radius: var(--r-md); }
  .t { font-weight: 600; color: var(--strong); max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; margin-top: var(--s1); }
  .s { font-size: var(--t-xs); color: var(--text2); max-width: 100%; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .r { font-size: var(--t-xs); color: var(--muted); }
  .link { flex: 1; min-width: 40px; display: grid; justify-items: center; gap: var(--s1); padding-top: var(--s5); font-size: var(--t-xs); color: var(--muted); text-align: center; }
  .link svg { width: 100%; height: 14px; overflow: visible; }
  .wire, .head { stroke: currentColor; stroke-width: var(--stroke); vector-effect: non-scaling-stroke; color: var(--text2); }
  .head { fill: none; stroke-linecap: round; stroke-linejoin: round; }
  .tunnel .wire, .tunnel .head { color: var(--accent); }
  .tunnel .wire { stroke-dasharray: 5 4; animation: flow 1.2s linear infinite; }
  .flip.tunnel .wire { animation-direction: reverse; }
  figcaption { margin-top: var(--s3); padding-top: var(--s3); border-top: 1px solid var(--line); font-size: var(--t-sm); color: var(--text2); line-height: 1.5; }
  @keyframes flow { to { stroke-dashoffset: -18; } }
  @media (prefers-reduced-motion: reduce) { .tunnel .wire { animation: none; } }
</style>
