<script lang="ts">
  import { ProfilesService, ToolsService, VaultService, type KeyInfo, type Profile } from "../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { vault } from "./vault.svelte";
  import VaultPanel from "./VaultPanel.svelte";

  let { initial, oldName, profiles, folders, onsave, ondelete, oncancel }: {
    initial: Profile;
    oldName: string;
    profiles: Profile[];
    folders: string[];
    onsave: (p: Profile, password: string) => Promise<void>;
    ondelete: () => void;
    oncancel: () => void;
  } = $props();

  // The form always edits real arrays, even though the generated type allows null.
  type Editable = Profile & { tags: string[]; tunnels: NonNullable<Profile["tunnels"]> };
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
  // The key list for the picker: keys adopted into the vault, and keys found in ~/.ssh.
  let keyInfos = $state<KeyInfo[]>([]);
  let otherKey = $state(false);
  ToolsService.Keys().then((k) => (keyInfos = k ?? [])).catch(() => {});
  const adoptedKeys = $derived(keyInfos.filter((k) => k.adopted));
  const linkedKeys = $derived(keyInfos.filter((k) => !k.adopted));
  const keyChoice = $derived.by(() => {
    if (adoptedKeys.some((k) => k.name === p.key)) return `a:${p.key}`;
    const linked = linkedKeys.find((k) => k.path === p.key || `~/.ssh/${k.name}` === p.key);
    if (linked) return `l:${linked.path}`;
    return otherKey || p.key ? "other" : "";
  });
  function pickKey(e: Event) {
    const v = (e.currentTarget as HTMLSelectElement).value;
    otherKey = v === "other";
    if (v.startsWith("a:")) p.key = v.slice(2);
    else if (v.startsWith("l:")) p.key = v.slice(2);
    else if (v === "") p.key = "";
  }
  async function browse() {
    const f = await ProfilesService.PickKeyFile().catch(() => "");
    if (f) p.key = f;
  }

  // svelte-ignore state_referenced_locally
  let hasSaved = $state(Boolean(oldName && vault.hasRemembered(oldName)));
  let changingPassword = $state(false);
  let passwordInput = $state("");
  let pendingPassword = "";
  let vaultPromptMode = $state<"create" | "unlock" | null>(null);

  // svelte-ignore state_referenced_locally
  if (oldName) {
    VaultService.HasSecret(oldName).then((has) => {
      if (has) {
        hasSaved = true;
        vault.remember(oldName);
      }
    }).catch(() => {});
  }

  async function forgetSavedPassword() {
    if (oldName) {
      try {
        await VaultService.ForgetSecret(oldName);
      } catch {}
      vault.forget(oldName);
    }
    hasSaved = false;
    changingPassword = false;
    passwordInput = "";
  }

  async function submit(e?: Event) {
    e?.preventDefault();
    if (tagInput.trim()) addTag();
    error = "";

    // A typed password, or a saved one that has to follow a rename, needs the vault open.
    const typed = p.auth === "password" ? passwordInput : "";
    const carrying = p.auth === "password" && hasSaved && oldName !== "" && oldName !== p.name;
    if (typed || carrying) {
      const st = await vault.refresh();
      if (!st.unlocked && (st.exists || typed)) {
        pendingPassword = typed;
        passwordInput = "";
        vaultPromptMode = st.exists ? "unlock" : "create";
        return;
      }
    }
    await commit(typed);
  }

  /** Saves the connection; the password (if any) goes with it, so the backend can bind it to the saved details. */
  async function commit(password: string) {
    busy = true;
    error = "";
    try {
      await onsave({ ...$state.snapshot(p), port: Number(p.port), key: p.auth === "key" ? p.key : "" } as Profile, password);
      // The form is closed by now; keep the "has a saved password" hints in step.
      if (p.auth !== "password" && oldName) vault.forget(oldName);
      else if (oldName && oldName !== p.name && hasSaved) vault.rename(oldName, p.name);
      if (password) vault.remember(p.name);
    } catch (err) {
      error = (err as Error).message ?? String(err);
    } finally {
      passwordInput = "";
      busy = false;
    }
  }

  async function onVaultPromptSuccess() {
    vaultPromptMode = null;
    const pass = pendingPassword;
    pendingPassword = "";
    await commit(pass);
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
</script>

<div class="scrim" role="presentation" onkeydown={keydown}>
  <form role="dialog" aria-label={isNew ? "New connection" : "Edit connection"} onsubmit={submit}>
    <header>
      <div class="tile"><svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M4 17l6-6-6-6M12 19h8" /></svg></div>
      <div class="titles"><h2>{isNew ? "New connection" : "Edit connection"}</h2>{#if !isNew}<span>{oldName}{initial.folder ? " · " + initial.folder : ""}</span>{/if}</div>
      <button type="button" class="close" aria-label="Close" onclick={oncancel}>&times;</button>
    </header>

    <div class="body">
      <div class="left">
        <section>
          <div class="label">Connection</div>
          <div class="g2">
            <label>Name<input class="mono" bind:value={p.name} required placeholder="web-1" /></label>
            <label>Folder<input class="mono" bind:value={p.folder} list="folders" placeholder="prod/eu" /></label>
          </div>
          <div class="g3">
            <label>Host<input class="mono" bind:value={p.host} required placeholder="10.0.0.5 or example.com" /></label>
            <label>Port<input class="mono" type="number" bind:value={p.port} min="1" max="65535" required /></label>
            <label>User<input class="mono" bind:value={p.user} required /></label>
          </div>
        </section>

        <section>
          <div class="label">Authentication</div>
          <div class="seg" role="radiogroup" aria-label="Authentication">
            {#each auths as a}
              <button type="button" role="radio" aria-checked={p.auth === a.id} class:on={p.auth === a.id} onclick={() => (p.auth = a.id)}>
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d={a.icon} /></svg>{a.label}
              </button>
            {/each}
          </div>
          {#if p.auth === "key"}
            <div class="keyrow">
              <select aria-label="Key" value={keyChoice} onchange={pickKey}>
                <option value="" disabled>Choose a key…</option>
                {#if adoptedKeys.length}
                  <optgroup label="Adopted (in backups)">
                    {#each adoptedKeys as k}<option value={`a:${k.name}`}>{k.name}{k.encrypted ? " (passphrase)" : ""}</option>{/each}
                  </optgroup>
                {/if}
                {#if linkedKeys.length}
                  <optgroup label="In ~/.ssh (not in backups)">
                    {#each linkedKeys as k}<option value={`l:${k.path}`}>{k.name}{k.encrypted ? " (passphrase)" : ""}</option>{/each}
                  </optgroup>
                {/if}
                <option value="other">Another file…</option>
              </select>
              {#if keyChoice === "other"}
                <input class="mono" bind:value={p.key} required placeholder="~/.ssh/id_ed25519 or a full path" aria-label="Key file path" />
                <button type="button" class="btn" onclick={browse}>Browse</button>
              {/if}
            </div>
            {#if keyChoice.startsWith("l:") || keyChoice === "other"}<p class="note">This key stays where it is and is not in backups. Adopt it in Tools → Keys to copy it into Relay's vault.</p>{/if}
          {:else if p.auth === "password"}
            {#if hasSaved && !changingPassword}
              <div class="saved-pass-row">
                <span class="badge-saved">
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><rect x="3" y="11" width="18" height="11" rx="2" ry="2" /><path d="M7 11V7a5 5 0 0 1 10 0v4" /></svg>
                  <span>A password is saved</span>
                </span>
                <div class="saved-pass-actions">
                  <button type="button" class="btn small" onclick={() => (changingPassword = true)}>Change</button>
                  <button type="button" class="btn small danger-text" onclick={forgetSavedPassword}>Forget</button>
                </div>
              </div>
            {:else}
              <div class="passrow">
                <input type="password" bind:value={passwordInput} placeholder={changingPassword ? "New password" : "Password (optional)"} aria-label="Password" autocomplete="new-password" />
                {#if changingPassword}
                  <button type="button" class="btn small" onclick={() => { changingPassword = false; passwordInput = ""; }}>Cancel</button>
                {/if}
              </div>
            {/if}
            <p class="note">Stored encrypted in Relay's vault, unlocked with your master password. Leave empty to be asked each time.</p>
          {/if}
          {#if p.auth === "agent" || p.auth === "key"}
            <p class="note">{p.auth === "agent" ? "Uses keys loaded in your ssh-agent." : "Passphrases are asked by ssh and never stored."}</p>
          {/if}
        </section>

        <section>
          <div class="label">Routing</div>
          <div class="g2">
            <label>Jump host
              <select bind:value={p.jump}><option value="">None</option>{#each others as o}<option value={o}>{o}</option>{/each}</select>
            </label>
            <div class="field">Tags
              <div class="tags">
                {#each p.tags as t, i}<span class="tag">{t}<button type="button" aria-label="Remove {t}" onclick={() => p.tags.splice(i, 1)}>&times;</button></span>{/each}
                <input bind:value={tagInput} placeholder={p.tags.length ? "" : "add tag"} aria-label="Add tag" onkeydown={(e) => { if (e.key === "Enter" || e.key === ",") { e.preventDefault(); e.stopPropagation(); addTag(); } }} onblur={addTag} />
              </div>
            </div>
          </div>
          <div class="route mono" aria-label="Route">
            <span class="rl">Route</span><span class="hop">this machine</span>
            {#each chain as h}<span>→</span><span class="hop">{h}</span>{/each}
            <span>→</span><span class="hop last">{p.name || "new"}</span>
          </div>
          <div class="switch">
            <div><b>Forward SSH agent</b><span>Lets this host use your local keys. Only for hosts you trust.</span></div>
            <button type="button" role="switch" aria-checked={p.forwardAgent} aria-label="Forward SSH agent" class:on={p.forwardAgent} onclick={() => (p.forwardAgent = !p.forwardAgent)}><i></i></button>
          </div>
          <div class="switch">
            <div><b>Command blocks</b><span>Split this session's output into blocks. Needs bash or zsh on the host.</span></div>
            <select bind:value={p.blocks} aria-label="Command blocks">
              <option value="">App default</option>
              <option value="on">Automatic</option>
              <option value="off">Off</option>
            </select>
          </div>
        </section>

        <section class="last">
          <div class="row"><span class="label">Port forwarding</span><button type="button" class="link" onclick={addTunnel}>+ Add tunnel</button></div>
          {#if p.tunnels.length}
            <div class="tun head"><span>Type</span><span>Listen</span><span>Target</span><span></span></div>
            {#each p.tunnels as t, i}
              <div class="tun">
                <select bind:value={t.type} aria-label="Tunnel type"><option value="local">Local</option><option value="remote">Remote</option><option value="dynamic">SOCKS</option></select>
                <input class="mono" bind:value={t.listen} placeholder={t.type === "remote" ? "remote port" : "local port"} aria-label="Listen" required />
                {#if t.type !== "dynamic"}<input class="mono" bind:value={t.target} placeholder="host:port" aria-label="Target" required />{:else}<span class="na">no target</span>{/if}
                <button type="button" aria-label="Remove tunnel" onclick={() => p.tunnels.splice(i, 1)}>&times;</button>
              </div>
            {/each}
          {/if}
        </section>
      </div>

      <aside>
        <div class="phead"><span class="label">SSH config preview</span><span>Written to ~/.ssh/sssh.d/config on save</span></div>
        <pre class="mono">{#each preview.config.split("\n") as line}{@const m = line.match(/^(\s*)(\S+)(.*)$/)}{#if m}{m[1]}<span class={m[1] ? "k2" : "k1"}>{m[2]}</span>{m[3]}{"\n"}{/if}{/each}</pre>
        <div class="state"><span class="dot" class:ok={!preview.problem} class:err={!!preview.problem}></span>{preview.problem || "Valid. Jump chain resolves."}</div>
        <div class="cli"><span>Connect from the CLI</span><div class="mono"><i>$</i> sssh {p.name || "name"}</div></div>
      </aside>
    </div>

    {#if error}<p class="error" role="alert">{error}</p>{/if}
    <footer>
      {#if !isNew}<button type="button" class="btn danger" onclick={ondelete}>Delete connection</button>{/if}
      <span class="esc">Esc to cancel</span>
      <button type="button" class="btn" onclick={oncancel}>Cancel</button>
      <button class="btn primary" disabled={busy}>{isNew ? "Add connection" : "Save changes"}<span class="mono">⌘↵</span></button>
    </footer>
    <datalist id="folders">{#each folders as f}<option value={f}></option>{/each}</datalist>
  </form>
  {#if vaultPromptMode}
    <VaultPanel
      mode={vaultPromptMode}
      onclose={() => { vaultPromptMode = null; pendingPassword = ""; busy = false; }}
      onsuccess={onVaultPromptSuccess}
    />
  {/if}
</div>

<style>
  .scrim { position: fixed; inset: 0; background: var(--scrim); display: flex; align-items: center; justify-content: center; z-index: 20; padding: 16px; }
  form { width: min(900px, 100%); max-height: 100%; display: flex; flex-direction: column; background: var(--float); border: 1px solid var(--floatline); border-radius: 8px; box-shadow: 0 30px 80px var(--shadow); overflow: hidden; }
  header { height: 64px; flex: none; display: flex; align-items: center; gap: 14px; padding: 0 20px 0 24px; border-bottom: 1px solid var(--line2); }
  .tile { width: 34px; height: 34px; display: grid; place-items: center; background: var(--accent-soft); border: 1px solid var(--accent-edge); border-radius: 6px; color: var(--accent-text); }
  .titles { display: grid; gap: 1px; }
  h2 { margin: 0; font-size: 16px; font-weight: 600; color: var(--strong); }
  .titles span { font-size: 12px; color: var(--muted); }
  .close { margin-left: auto; width: 32px; height: 32px; border: 0; background: none; border-radius: 4px; color: var(--muted); font-size: 20px; }
  .close:hover { background: var(--line); color: var(--strong); }
  .body { flex: 1; min-height: 0; display: flex; overflow: hidden; }
  .left { flex: 1; min-width: 0; overflow: auto; }
  section { padding: 13px 24px 15px; border-bottom: 1px solid var(--line2); display: grid; gap: 10px; }
  section.last { border-bottom: 0; gap: 8px; }
  .g2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
  .g3 { display: grid; grid-template-columns: 1fr 92px 150px; gap: 12px; }
  label, .field { display: grid; gap: 5px; font-size: 12px; color: var(--text2); }
  select { appearance: none; background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 24 24' fill='none' stroke='%239aa8bd' stroke-width='2.4' stroke-linecap='round' stroke-linejoin='round'%3E%3Cpath d='M6 9l6 6 6-6'/%3E%3C/svg%3E"); background-repeat: no-repeat; background-position: right 11px center; padding-right: 30px; font-family: var(--mono); }
  .seg { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; }
  .seg button { display: flex; align-items: center; gap: 10px; padding: 9px 12px; background: var(--field); border: 1px solid var(--fieldline); border-radius: 4px; color: var(--text2); }
  .seg button svg { color: var(--dim); }
  .seg button.on { background: var(--accent-soft); border-color: var(--accent); color: var(--strong); font-weight: 500; }
  .seg button.on svg { color: var(--accent-text); }
  .keyrow { display: flex; gap: 8px; }
  .keyrow select { flex: 1; min-width: 0; }
  .keyrow input { flex: 1; min-width: 0; }
  .passrow { display: flex; gap: 8px; align-items: center; }
  .passrow input { flex: 1; min-width: 0; }
  .saved-pass-row { display: flex; align-items: center; justify-content: space-between; padding: 6px 12px; background: var(--field); border: 1px solid var(--fieldline); border-radius: 4px; }
  .badge-saved { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--ok); font-weight: 500; }
  .saved-pass-actions { display: flex; gap: 6px; }
  .btn.small { height: 26px; padding: 0 10px; font-size: 11.5px; }
  .danger-text { color: var(--err-text, var(--warn)); }
  .note { margin: 0; font-size: 12px; color: var(--muted); }
  .tags { height: 36px; display: flex; align-items: center; gap: 6px; padding: 0 8px; background: var(--field); border: 1px solid var(--fieldline); border-radius: 4px; overflow: hidden; }
  .tags:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .tags input { flex: 1; min-width: 40px; height: 100%; border: 0; background: none; padding: 0 3px; box-shadow: none; }
  .tag { display: flex; align-items: center; gap: 4px; font-size: 12px; background: var(--line); padding: 2px 4px 2px 8px; border-radius: 3px; color: var(--text); white-space: nowrap; }
  .tag button { all: unset; cursor: pointer; color: var(--muted); font-size: 14px; line-height: 1; padding: 0 3px; }
  .route { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; font-size: 12px; color: var(--muted); }
  .rl { font: 11.5px var(--sans); color: var(--dim); margin-right: 4px; }
  .hop { padding: 2px 8px; background: var(--rail); border: 1px solid var(--fieldline); border-radius: 3px; }
  .hop.last { background: var(--accent-soft); border-color: var(--accent); color: var(--strong); }
  .switch { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 10px 12px; background: var(--rail); border: 1px solid var(--line2); border-radius: 4px; }
  .switch div { display: grid; gap: 1px; }
  .switch b { font-weight: 400; color: var(--strong); }
  .switch span { font-size: 11.5px; color: var(--muted); }
  .switch select { flex: none; width: 128px; }
  .switch button { flex: none; width: 36px; height: 20px; border: 0; border-radius: 10px; background: var(--floatline); position: relative; padding: 0; }
  .switch button.on { background: var(--accent); }
  .switch i { position: absolute; top: 3px; left: 3px; width: 14px; height: 14px; border-radius: 7px; background: var(--switch-thumb); transition: left 0.12s; }
  .switch button.on i { left: 19px; background: var(--on-accent); }
  .row { display: flex; align-items: center; justify-content: space-between; }
  .link { all: unset; font-size: 12px; color: var(--accent-text); cursor: pointer; }
  .tun { display: grid; grid-template-columns: 100px 1fr 1.4fr 28px; gap: 8px; align-items: center; }
  .tun.head { font-size: 11px; color: var(--dim); }
  .tun select, .tun input { height: 32px; font-size: 12.5px; }
  .tun button { all: unset; width: 28px; height: 28px; display: grid; place-items: center; color: var(--muted); font-size: 16px; cursor: pointer; border-radius: 4px; }
  .tun button:hover { background: var(--line); }
  .na { font-size: 12px; color: var(--dim); }
  aside { width: 290px; flex: none; background: var(--bar); border-left: 1px solid var(--line2); display: flex; flex-direction: column; overflow: auto; }
  .phead { padding: 16px 20px 12px; display: grid; gap: 3px; }
  .phead span:last-child { font-size: 11.5px; color: var(--dim); }
  pre { margin: 0 20px; padding: 14px; background: var(--strip); border: 1px solid var(--line); border-radius: 4px; font-size: 11px; line-height: 1.7; white-space: pre; overflow-x: auto; }
  .k1 { color: var(--accent-text); } .k2 { color: var(--muted); }
  .state { margin: 14px 20px 0; display: flex; align-items: flex-start; gap: 8px; font-size: 12px; color: var(--text2); }
  .state .dot { margin-top: 4px; width: 8px; height: 8px; }
  .cli { margin: auto 20px 20px; padding-top: 16px; display: grid; gap: 6px; }
  .cli span { font-size: 11.5px; color: var(--dim); }
  .cli div { display: flex; align-items: center; gap: 6px; height: 32px; padding: 0 10px; background: var(--strip); border: 1px solid var(--line); border-radius: 4px; font-size: 12px; }
  .cli i { color: var(--dim); font-style: normal; }
  .error { margin: 0; padding: 10px 24px; color: var(--err-text); background: var(--err-bg); border-top: 1px solid var(--err); }
  footer { height: 60px; flex: none; display: flex; align-items: center; gap: 10px; padding: 0 20px 0 24px; border-top: 1px solid var(--line2); background: var(--foot); }
  .esc { margin-left: auto; font-size: 11.5px; color: var(--dim); }
  .btn.primary { display: flex; align-items: center; gap: 10px; padding: 9px 16px; }
  .btn.primary span { font: 500 11px var(--sans); opacity: 0.75; }
  .btn.danger { margin-right: auto; padding-left: 4px; }
  footer .esc { margin-left: auto; }
</style>
