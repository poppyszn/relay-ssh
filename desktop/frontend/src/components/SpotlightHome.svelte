<script lang="ts">
  import { tick } from "svelte";
  import type { Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { matches, parseTarget } from "../lib/lib";
  import Logo from "./Logo.svelte";
  import Icon from "./Icon.svelte";
  import { panelSwap } from "../lib/motion";

  const mac = /Mac|iPhone|iPad/.test(navigator.platform);
  const mod = mac ? "⌘" : "Ctrl+";

  let {
    profiles,
    recentIds = [],
    activeTargets,
    vaultExists = false,
    vaultUnlocked = false,
    onconnect,
    onnew,
    onlocal,
    onhosts,
    onimport,
    onsettings,
    onpalette,
  }: {
    profiles: Profile[];
    recentIds?: string[];
    activeTargets: Set<string>;
    vaultExists?: boolean;
    vaultUnlocked?: boolean;
    onconnect: (p: Profile, split?: boolean) => void;
    onnew: (prefill?: { user: string; host: string; port: number }) => void;
    onlocal: () => void;
    onhosts: () => void;
    onimport: () => void;
    onsettings: () => void;
    onpalette: () => void;
  } = $props();

  let query = $state("");
  let sel = $state(0);
  let inputEl: HTMLInputElement | undefined = $state();

  const parsed = $derived(parseTarget(query));

  const ranked = $derived.by(() => {
    const rank = new Map(recentIds.map((id, i) => [id, i]));
    const copy = profiles.filter((p) => matches(p, query));
    copy.sort((a, b) => {
      const ra = rank.get(a.id) ?? Number.MAX_SAFE_INTEGER;
      const rb = rank.get(b.id) ?? Number.MAX_SAFE_INTEGER;
      if (ra !== rb) return ra - rb;
      return a.name.localeCompare(b.name);
    });
    return copy;
  });

  const shown = $derived(ranked.slice(0, 5));
  const extraCount = $derived(Math.max(0, ranked.length - shown.length));

  $effect(() => {
    query;
    sel = 0;
  });

  $effect(() => {
    tick().then(() => {
      // Only autofocus when no modal/dialog is currently mounted above the deck
      if (!document.querySelector("[role=dialog], [role=alertdialog]")) {
        inputEl?.focus({ preventScroll: true });
      }
    });
  });

  function subtitle(p: Profile): string {
    const portSuffix = p.port && ((p.protocol === "rdp" && p.port !== 3389) || (p.protocol !== "rdp" && p.port !== 22)) ? `:${p.port}` : "";
    const addr = `${p.user ? p.user + "@" : ""}${p.host}${portSuffix}`;
    if (p.protocol === "rdp") return `${addr} · RDP`;
    if (p.tunnels?.length) {
      const first = p.tunnels[0];
      return `${addr} · Forward :${first.listen}`;
    }
    if (p.jump) {
      const jumpName = profiles.find((x) => x.id === p.jump)?.name ?? p.jump;
      return `${addr} · via ${jumpName}`;
    }
    if (p.tags?.length) return `${addr} · ${p.tags.length} ${p.tags.length === 1 ? "tag" : "tags"}`;
    return addr;
  }

  function handleKey(e: KeyboardEvent) {
    if (e.key === "ArrowDown") {
      if (shown.length > 0) {
        e.preventDefault();
        sel = Math.min(sel + 1, shown.length - 1);
      }
    } else if (e.key === "ArrowUp") {
      if (shown.length > 0) {
        e.preventDefault();
        sel = Math.max(sel - 1, 0);
      }
    } else if (e.key === "Enter") {
      e.preventDefault();
      if (parsed) {
        onnew(parsed);
      } else if (shown[sel]) {
        onconnect(shown[sel], e.metaKey || e.ctrlKey);
      } else if (!query.trim() && profiles.length === 0) {
        onlocal();
      }
    } else if (e.key === "Escape" && query) {
      e.preventDefault();
      query = "";
    }
  }
</script>

<section class="spotlight-stage" aria-label="Relay Deck launcher">
  <div class="ambient-glow" aria-hidden="true"></div>

  <div class="spotlight-shell">
    <!-- Hero Header -->
    <header class="hero-row">
      <div class="brand-lockup">
        <Logo size={40} />
        <div>
          <h1 class="hero-title">Relay Deck</h1>
          <p class="hero-sub">Direct SSH, jump chains, and remote desktops with zero sidebar noise.</p>
        </div>
      </div>

      <div class="hero-actions">
        <button type="button" class="deck-btn-primary" onclick={onlocal}>
          Open Local Shell
        </button>
        <button type="button" class="deck-btn-ghost" onclick={() => onnew()}>
          <Icon name="plus" size={14} />
          <span>New Connection</span>
        </button>
      </div>
    </header>

    <!-- Flagship Spotlight Card -->
    <div class="spotlight-card">
      <div class="search-band">
        <span class="search-icon"><Icon name="search" size={18} /></span>
        <input
          bind:this={inputEl}
          bind:value={query}
          class="spotlight-input"
          placeholder="Connect user@host:port, filter hosts, or trigger command…"
          aria-label="Connect or filter hosts"
          spellcheck="false"
          autocomplete="off"
          onkeydown={handleKey}
        />
        {#if query}
          <button
            type="button"
            class="clear-btn"
            aria-label="Clear search"
            onclick={() => { query = ""; inputEl?.focus(); }}
          >
            <Icon name="x" size={14} />
          </button>
        {/if}
        <button
          type="button"
          class="kbd-trigger mono"
          title="Open Command Palette"
          onclick={onpalette}
        >
          {mod}K
        </button>
      </div>

      {#if parsed}
        <div class="parsed-wrap" in:panelSwap|global={{ duration: 150, y: 4 }}>
          <button
            type="button"
            class="parsed-row"
            onclick={() => onnew(parsed)}
          >
            <div class="parsed-left">
              <span class="parsed-badge"><Icon name="zap" size={14} /></span>
              <div>
                <div class="parsed-title">
                  Configure &amp; Connect <span class="mono">{parsed.user ? `${parsed.user}@` : ""}{parsed.host}{parsed.port !== 22 ? `:${parsed.port}` : ""}</span>
                </div>
                <div class="parsed-sub">Opens prefilled host profile for key, jump, and vault setup</div>
              </div>
            </div>
            <span class="tile-enter mono">↵</span>
          </button>
        </div>
      {/if}

      {#if shown.length > 0}
        <div class="tile-grid" role="group" aria-label="Recent and matching hosts">
          {#each shown as p, idx (p.id)}
            {@const isLive = activeTargets.has(p.id)}
            {@const isRdp = p.protocol === "rdp"}
            <button
              type="button"
              class="host-tile"
              class:selected={!parsed && idx === sel}
              title="Click to open · {mod}Click to split"
              onclick={(e) => onconnect(p, e.metaKey || e.ctrlKey)}
              onmouseenter={() => (sel = idx)}
            >
              <span
                class="status-dot"
                class:live={isLive && !isRdp}
                class:rdp={isRdp}
                aria-hidden="true"
              ></span>
              <span class="tile-name">{p.name}</span>
              <span class="tile-sub mono">{subtitle(p)}</span>
              <span class="tile-badge mono" class:rdp-tag={isRdp}>
                {isRdp ? "RDP" : "↵"}
              </span>
            </button>
          {/each}
        </div>
      {:else if !parsed}
        <div class="deck-empty">
          {#if profiles.length === 0}
            <p class="deck-empty-title">No saved hosts yet</p>
            <p class="deck-empty-sub">Import your existing <code class="mono">~/.ssh/config</code> or add your first server to start.</p>
            <div class="deck-empty-acts">
              <button type="button" class="deck-btn-ghost" onclick={onimport}>Import ~/.ssh/config</button>
              <button type="button" class="deck-btn-primary" onclick={() => onnew()}>Add First Host</button>
            </div>
          {:else}
            <p class="deck-empty-title">No hosts match "{query}"</p>
            <p class="deck-empty-sub">Type <span class="mono">user@host</span> to configure a new target, or browse all hosts.</p>
          {/if}
        </div>
      {/if}

      {#if extraCount > 0}
        <div class="card-foot">
          <button type="button" class="foot-more" onclick={onhosts}>
            +{extraCount} more matching {extraCount === 1 ? "host" : "hosts"} · Browse All (<span class="mono">{mod}B</span>)
          </button>
        </div>
      {/if}
    </div>

    <!-- Subtle Context Triggers Below Card -->
    <footer class="context-bar">
      <div class="context-links">
        <button type="button" onclick={onhosts}>
          <span class="mono key-hint">{mod}B</span> Browse All Hosts
        </button>
        <button type="button" onclick={onimport}>
          Import ~/.ssh/config
        </button>
        <button type="button" onclick={onsettings}>
          <span class="mono key-hint">{mod},</span> Themes &amp; Vault
        </button>
      </div>

      <button
        type="button"
        class="vault-pill"
        class:ok={vaultExists && vaultUnlocked}
        class:warn={vaultExists && !vaultUnlocked}
        onclick={onsettings}
      >
        <span class="v-dot" aria-hidden="true"></span>
        <span>
          {#if !vaultExists}
            Vault Not Configured
          {:else if vaultUnlocked}
            Vault Unlocked
          {:else}
            Vault Locked
          {/if}
        </span>
      </button>
    </footer>
  </div>
</section>

<style>
  .spotlight-stage {
    flex: 1;
    min-width: 0;
    min-height: 0;
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 36px;
    overflow-y: auto;
    background: var(--bg);
  }

  .ambient-glow {
    pointer-events: none;
    position: absolute;
    top: 48px;
    width: 560px;
    height: 220px;
    border-radius: 999px;
    background: var(--accent);
    opacity: 0.085;
    filter: blur(110px);
  }

  .spotlight-shell {
    width: 100%;
    max-width: 780px;
    display: flex;
    flex-direction: column;
    gap: 22px;
    position: relative;
    z-index: 1;
    animation: page-in 260ms var(--ease) both;
  }

  .hero-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
  }

  .brand-lockup {
    display: flex;
    align-items: center;
    gap: 14px;
  }

  .hero-title {
    margin: 0;
    font-size: 23px;
    font-weight: 600;
    letter-spacing: -0.025em;
    color: var(--strong);
    line-height: 1.2;
  }

  .hero-sub {
    margin: 3px 0 0;
    font-size: 13px;
    color: var(--muted);
  }

  .hero-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .deck-btn-primary {
    height: 36px;
    padding: 0 16px;
    border-radius: 10px;
    background: var(--strong);
    color: var(--bg);
    font-size: 12px;
    font-weight: 600;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    transition: opacity var(--dur-1) var(--ease), transform var(--dur-1) var(--ease);
  }

  .deck-btn-primary:hover {
    opacity: 0.92;
  }

  .deck-btn-primary:active {
    transform: translateY(1px);
  }

  .deck-btn-ghost {
    height: 36px;
    padding: 0 14px;
    border-radius: 10px;
    background: var(--hover);
    border: 1px solid var(--hairline);
    color: var(--text);
    font-size: 12px;
    font-weight: 500;
    display: inline-flex;
    align-items: center;
    gap: 6px;
    transition: background var(--dur-1) var(--ease), color var(--dur-1) var(--ease);
  }

  .deck-btn-ghost:hover {
    background: var(--active);
    color: var(--strong);
  }

  .spotlight-card {
    border-radius: 16px;
    background: var(--float);
    border: 1px solid var(--floatline);
    box-shadow: var(--e-modal);
    overflow: hidden;
  }

  .search-band {
    height: 56px;
    padding: 0 18px;
    display: flex;
    align-items: center;
    gap: 12px;
    border-bottom: 1px solid var(--hairline);
  }

  .search-icon {
    color: var(--accent-text);
    display: grid;
    place-items: center;
    flex: none;
  }

  .spotlight-input {
    flex: 1;
    min-width: 0;
    height: 100%;
    padding: 0;
    border: 0;
    background: transparent;
    color: var(--strong);
    font-size: 15px;
    box-shadow: none;
  }

  .spotlight-input:focus {
    outline: none;
    box-shadow: none;
  }

  .spotlight-input::placeholder {
    color: var(--dim);
  }

  .clear-btn {
    width: 24px;
    height: 24px;
    border-radius: 6px;
    display: grid;
    place-items: center;
    color: var(--muted);
  }

  .clear-btn:hover {
    background: var(--active);
    color: var(--strong);
  }

  .kbd-trigger {
    padding: 4px 8px;
    border-radius: 6px;
    background: var(--hover);
    border: 1px solid var(--hairline);
    color: var(--muted);
    font-size: 12px;
    flex: none;
  }

  .kbd-trigger:hover {
    background: var(--active);
    color: var(--strong);
  }

  .parsed-wrap {
    padding: 12px 16px 0;
  }

  .parsed-row {
    width: 100%;
    padding: 12px 14px;
    border-radius: 12px;
    background: var(--accent-soft);
    border: 1px solid var(--accent-edge);
    display: flex;
    align-items: center;
    justify-content: space-between;
    text-align: left;
  }

  .parsed-left {
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;
  }

  .parsed-badge {
    width: 28px;
    height: 28px;
    border-radius: 8px;
    background: var(--accent);
    color: var(--on-accent);
    display: grid;
    place-items: center;
    flex: none;
  }

  .parsed-title {
    font-size: 13.5px;
    font-weight: 500;
    color: var(--strong);
    margin-bottom: 2px;
  }

  .parsed-sub {
    font-size: 12px;
    color: var(--muted);
  }

  .tile-grid {
    padding: 8px 10px 10px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .host-tile {
    width: 100%;
    height: 38px;
    padding: 0 12px;
    border-radius: 8px;
    background: transparent;
    border: 0;
    text-align: left;
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
    color: var(--muted);
    transition: background var(--dur-1) var(--ease), color var(--dur-1) var(--ease);
  }

  .host-tile:hover,
  .host-tile.selected {
    background: var(--hover);
    color: var(--text);
  }

  .status-dot {
    width: 7px;
    height: 7px;
    border-radius: 999px;
    background: var(--dot-off);
    flex: none;
    opacity: 0.75;
    transition: opacity var(--dur-1) var(--ease);
  }

  .status-dot.live {
    background: var(--ok);
    opacity: 0.9;
  }

  .status-dot.rdp {
    background: var(--accent-text);
    opacity: 0.9;
  }

  .host-tile:hover .status-dot,
  .host-tile.selected .status-dot {
    opacity: 1;
  }

  .tile-name {
    font-size: 13.5px;
    font-weight: 500;
    color: var(--muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    flex: none;
    max-width: 44%;
    transition: color var(--dur-1) var(--ease);
  }

  .host-tile:hover .tile-name,
  .host-tile.selected .tile-name {
    color: var(--text);
  }

  .tile-sub {
    font-size: 12px;
    color: var(--dim);
    flex: 1;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    transition: color var(--dur-1) var(--ease);
  }

  .host-tile:hover .tile-sub,
  .host-tile.selected .tile-sub {
    color: var(--muted);
  }

  .tile-badge,
  .tile-enter {
    font-size: 12px;
    color: var(--dim);
    opacity: 0.55;
    flex: none;
    transition: opacity var(--dur-1) var(--ease), color var(--dur-1) var(--ease);
  }

  .host-tile:hover .tile-badge,
  .host-tile.selected .tile-badge {
    color: var(--muted);
    opacity: 1;
  }

  .tile-badge.rdp-tag {
    color: var(--accent-text);
    opacity: 0.75;
  }

  .deck-empty {
    padding: 28px 20px;
    text-align: center;
    display: grid;
    gap: 8px;
    justify-items: center;
  }

  .deck-empty-title {
    margin: 0;
    font-size: 14px;
    font-weight: 500;
    color: var(--strong);
  }

  .deck-empty-sub {
    margin: 0;
    font-size: 12.5px;
    color: var(--muted);
  }

  .deck-empty-acts {
    margin-top: 8px;
    display: flex;
    gap: 8px;
  }

  .card-foot {
    padding: 0 16px 12px;
    display: flex;
    justify-content: flex-end;
  }

  .foot-more {
    font-size: 12px;
    color: var(--accent-text);
    padding: 2px 6px;
    border-radius: 6px;
  }

  .foot-more:hover {
    background: var(--accent-soft);
  }

  .context-bar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 0 4px;
    font-size: 12px;
    color: var(--muted);
  }

  .context-links {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 18px;
  }

  .context-links button {
    color: var(--muted);
  }

  .context-links button:hover {
    color: var(--strong);
  }

  .key-hint {
    color: var(--text2);
    margin-right: 4px;
  }

  .vault-pill {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--dim);
    font-size: 12px;
  }

  .vault-pill.ok {
    color: var(--ok);
  }

  .vault-pill.warn {
    color: var(--warn);
  }

  .v-dot {
    width: 6px;
    height: 6px;
    border-radius: 999px;
    background: currentColor;
  }
</style>
