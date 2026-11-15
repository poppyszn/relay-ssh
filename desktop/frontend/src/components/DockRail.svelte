<script lang="ts" module>
  export type DrawerKind = "none" | "hosts" | "files" | "forwards" | "snippets" | "tools";
</script>

<script lang="ts">
  import Logo from "./Logo.svelte";
  import Icon, { type IconName } from "./Icon.svelte";

  const mac = /Mac|iPhone|iPad/.test(navigator.platform);
  const mod = mac ? "⌘" : "Ctrl+";

  let {
    drawer,
    onDeck,
    activeForwardCount = 0,
    vaultExists = false,
    vaultUnlocked = false,
    ondeck,
    ontoggledrawer,
    onvault,
    onsettings,
  }: {
    drawer: DrawerKind;
    onDeck: boolean;
    activeForwardCount?: number;
    vaultExists?: boolean;
    vaultUnlocked?: boolean;
    ondeck: () => void;
    ontoggledrawer: (kind: Exclude<DrawerKind, "none">) => void;
    onvault: () => void;
    onsettings: () => void;
  } = $props();

  const items: { id: Exclude<DrawerKind, "none">; icon: IconName; label: string; hint: string }[] = [
    { id: "hosts", icon: "server", label: "Saved Hosts", hint: `Saved Hosts (${mod}B)` },
    { id: "files", icon: "folder", label: "Remote Files (SFTP)", hint: `Remote Files · SFTP (${mac ? "⇧⌘O" : "Ctrl+Shift+O"})` },
    { id: "forwards", icon: "forwards", label: "Port Forwards", hint: `Port Forwards (${mac ? "⇧⌘P" : "Ctrl+Shift+P"})` },
    { id: "snippets", icon: "snippets", label: "Command Snippets", hint: `Command Snippets (${mac ? "⇧⌘S" : "Ctrl+Shift+S"})` },
    { id: "tools", icon: "tools", label: "Tools, Import & SSH Keys", hint: "Tools, Health, Import & Keys" },
  ];

  const vaultLabel = $derived(
    vaultExists
      ? vaultUnlocked
        ? "Vault unlocked · Manage in Settings"
        : "Vault locked · Unlock in Settings"
      : "Set up encrypted Vault in Settings",
  );
  const vaultIcon = $derived<IconName>(vaultExists ? (vaultUnlocked ? "unlock" : "lock") : "vault");
</script>

<nav class="dock-rail" aria-label="Command dock">
  <div class="dock-top">
    <button
      type="button"
      class="dock-brand"
      class:active={onDeck && drawer === "none"}
      aria-label="Relay Deck home"
      aria-pressed={onDeck && drawer === "none"}
      title="Relay Deck — Spotlight Home"
      onclick={ondeck}
    >
      <Logo size={20} />
    </button>

    <div class="dock-rule" aria-hidden="true"></div>

    {#each items as item (item.id)}
      <button
        type="button"
        class="dock-btn"
        class:active={drawer === item.id}
        data-dock={item.id}
        aria-label={item.label}
        aria-pressed={drawer === item.id}
        title={item.hint}
        onclick={() => ontoggledrawer(item.id)}
      >
        <Icon name={item.icon} size={17} />
        {#if item.id === "forwards" && activeForwardCount > 0}
          <span class="pulse-dot" aria-label="{activeForwardCount} active forward"></span>
        {/if}
      </button>
    {/each}
  </div>

  <div class="dock-bottom">
    <button
      type="button"
      class="dock-btn vault-btn"
      class:vault-ok={vaultExists && vaultUnlocked}
      class:vault-warn={vaultExists && !vaultUnlocked}
      aria-label={vaultLabel}
      title={vaultLabel}
      onclick={onvault}
    >
      <Icon name={vaultIcon} size={16} />
    </button>

    <button
      type="button"
      class="dock-btn"
      data-dock="settings"
      aria-label="Settings"
      title={`Settings (${mod},)`}
      onclick={onsettings}
    >
      <Icon name="settings" size={16} />
    </button>
  </div>
</nav>

<style>
  .dock-rail {
    width: 56px;
    flex: none;
    background: var(--rail);
    border-right: 1px solid var(--hairline);
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: space-between;
    padding: 12px 0;
    user-select: none;
    z-index: 20;
  }

  .dock-top,
  .dock-bottom {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    width: 100%;
  }

  .dock-rule {
    width: 20px;
    height: 1px;
    background: var(--hairline);
    margin: 2px 0;
  }

  .dock-brand,
  .dock-btn {
    position: relative;
    width: 36px;
    height: 36px;
    border-radius: 10px;
    display: grid;
    place-items: center;
    color: var(--muted);
    background: transparent;
    border: 1px solid transparent;
    transition:
      background var(--dur-1) var(--ease),
      color var(--dur-1) var(--ease),
      border-color var(--dur-1) var(--ease),
      transform var(--dur-1) var(--ease);
  }
  .dock-brand:active,
  .dock-btn:active {
    transform: scale(0.94);
  }

  .dock-brand {
    background: var(--hover);
  }

  .dock-brand:hover,
  .dock-btn:hover {
    background: var(--active);
    color: var(--strong);
  }

  .dock-brand.active,
  .dock-btn.active {
    background: var(--accent-soft);
    color: var(--accent-text);
    border-color: var(--accent-edge);
  }

  .dock-btn.vault-ok {
    color: var(--ok);
  }

  .dock-btn.vault-ok:hover {
    background: color-mix(in srgb, var(--ok) 14%, transparent);
    color: var(--ok);
  }

  .dock-btn.vault-warn {
    color: var(--warn);
  }

  .dock-btn.vault-warn:hover {
    background: color-mix(in srgb, var(--warn) 14%, transparent);
    color: var(--warn);
  }

  .pulse-dot {
    position: absolute;
    top: 6px;
    right: 6px;
    width: 6px;
    height: 6px;
    border-radius: 999px;
    background: var(--ok);
    animation: dockPulse 2.2s ease-in-out infinite;
  }

  @keyframes dockPulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.45; }
  }

  @media (prefers-reduced-motion: reduce) {
    .pulse-dot { animation: none; }
  }
</style>
