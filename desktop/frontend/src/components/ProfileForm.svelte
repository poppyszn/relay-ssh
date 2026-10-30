<script lang="ts">
  import { ProfilesService, ToolsService, VaultService, type KeyInfo, type Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { vault } from "../state/vault.svelte";
  import Icon, { type IconName } from "./Icon.svelte";
  import Modal from "./Modal.svelte";
  import VaultPanel from "./VaultPanel.svelte";
  import RevealPassword from "./RevealPassword.svelte";

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
  // svelte-ignore state_referenced_locally
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
  let revealing = $state(false);
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
    if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) submit();
  }
  const addTunnel = () => p.tunnels.push({ type: "local", listen: "", target: "" });
  const auths: { id: string; label: string; icon: IconName }[] = [
    { id: "agent", label: "Agent", icon: "vault" },
    { id: "key", label: "Key file", icon: "key" },
    { id: "password", label: "Password", icon: "lock" },
  ];
  const formId = "profile-form";
  // Nothing to judge until the user has typed a name or host: no verdict on an empty form.
  const started = $derived(Boolean(p.name.trim() || p.host.trim()));
</script>

<Modal title={isNew ? "New connection" : "Edit connection"} subtitle={isNew ? "" : oldName} size="lg" flush dismissOnScrim={false} onclose={oncancel}>
  <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
  <form id={formId} class="cols" onsubmit={submit} onkeydown={keydown}>
    <div class="left">
      <section>
        <h3 class="label">Connection</h3>
        <div class="g2">
          <label class="field"><span>Name</span><input class="mono" bind:value={p.name} required placeholder="web-1" /></label>
          <label class="field"><span>Folder</span><input class="mono" bind:value={p.folder} list="folders" placeholder="prod/eu" /></label>
        </div>
        <div class="g3">
          <label class="field"><span>Host</span><input class="mono" bind:value={p.host} required placeholder="10.0.0.5 or example.com" /></label>
          <label class="field"><span>Port</span><input class="mono" type="number" bind:value={p.port} min="1" max="65535" required /></label>
          <label class="field"><span>User</span><input class="mono" bind:value={p.user} required /></label>
        </div>
      </section>

      <section>
        <h3 class="label">Authentication</h3>
        <div class="seg" role="group" aria-label="Authentication">
          {#each auths as a (a.id)}
            <button type="button" aria-pressed={p.auth === a.id} onclick={() => (p.auth = a.id)}><Icon name={a.icon} />{a.label}</button>
          {/each}
        </div>
        {#if p.auth === "key"}
          <div class="inline">
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
          {#if keyChoice.startsWith("l:") || keyChoice === "other"}<p class="hint">This key stays where it is and is not in backups. Adopt it in Tools → Keys to copy it into Relay's vault.</p>{/if}
        {:else if p.auth === "password"}
          {#if hasSaved && !changingPassword}
            <div class="saved">
              <span class="chip ok"><Icon name="lock" size={14} />A password is saved</span>
              <span class="acts">
                <button type="button" class="btn sm" onclick={() => (revealing = true)}>Show</button>
                <button type="button" class="btn sm" onclick={() => (changingPassword = true)}>Change</button>
                <button type="button" class="btn sm ghost danger" onclick={forgetSavedPassword}>Forget</button>
              </span>
            </div>
          {:else}
            <div class="inline">
              <input type="password" bind:value={passwordInput} placeholder={changingPassword ? "New password" : "Password (optional)"} aria-label="Password" autocomplete="new-password" />
              {#if changingPassword}
                <button type="button" class="btn" onclick={() => { changingPassword = false; passwordInput = ""; }}>Cancel</button>
              {/if}
            </div>
          {/if}
          <p class="hint">Stored encrypted in Relay's vault, unlocked with your master password. Leave empty to be asked each time.</p>
        {/if}
        {#if p.auth === "agent" || p.auth === "key"}
          <p class="hint">{p.auth === "agent" ? "Uses keys loaded in your ssh-agent." : "Passphrases are asked by ssh and never stored."}</p>
        {/if}
      </section>

      <section>
        <h3 class="label">Routing</h3>
        <div class="g2">
          <label class="field">
            <span>Jump host</span>
            <select bind:value={p.jump}><option value="">None</option>{#each others as o}<option value={o}>{o}</option>{/each}</select>
          </label>
          <div class="field">
            <span class="field-label">Tags</span>
            <div class="tags">
              {#each p.tags as t, i (t)}
                <span class="chip">{t}<button type="button" class="icon-btn dense" aria-label="Remove {t}" onclick={() => p.tags.splice(i, 1)}><Icon name="x" size={14} /></button></span>
              {/each}
              <input bind:value={tagInput} placeholder={p.tags.length ? "" : "Add a tag"} aria-label="Add tag" onkeydown={(e) => { if (e.key === "Enter" && !e.metaKey && !e.ctrlKey || e.key === ",") { e.preventDefault(); e.stopPropagation(); addTag(); } }} onblur={addTag} />
            </div>
          </div>
        </div>
        <div class="route mono" role="group" aria-label="Route">
          <span class="chip">this machine</span>
          {#each chain as h}<Icon name="arrow-right" size={14} /><span class="chip">{h}</span>{/each}
          <Icon name="arrow-right" size={14} /><span class="chip accent">{p.name || "new"}</span>
        </div>
        <div class="opt">
          <div><span class="t">Forward SSH agent</span><span class="hint">Lets this host use your local keys. Only for hosts you trust.</span></div>
          <button type="button" role="switch" class="switch" aria-checked={p.forwardAgent} aria-label="Forward SSH agent" onclick={() => (p.forwardAgent = !p.forwardAgent)}></button>
        </div>
        <div class="opt">
          <div><span class="t">Command blocks</span><span class="hint">Split this session's output into blocks. Needs bash or zsh on the host.</span></div>
          <select bind:value={p.blocks} aria-label="Command blocks">
            <option value="">App default</option>
            <option value="on">Automatic</option>
            <option value="off">Off</option>
          </select>
        </div>
      </section>

      <section>
        <div class="head"><h3 class="label">Port forwarding</h3><button type="button" class="btn link" onclick={addTunnel}><Icon name="plus" size={14} />Add tunnel</button></div>
        {#if p.tunnels.length}
          <div class="tun"><span class="label">Type</span><span class="label">Listen</span><span class="label">Target</span><span></span></div>
          {#each p.tunnels as t, i}
            <div class="tun">
              <select bind:value={t.type} aria-label="Tunnel type"><option value="local">Local</option><option value="remote">Remote</option><option value="dynamic">SOCKS</option></select>
              <input class="mono" bind:value={t.listen} placeholder={t.type === "remote" ? "remote port" : "local port"} aria-label="Listen" required />
              {#if t.type !== "dynamic"}<input class="mono" bind:value={t.target} placeholder="host:port" aria-label="Target" required />{:else}<span class="hint">No target</span>{/if}
              <button type="button" class="icon-btn dense" aria-label="Remove tunnel" onclick={() => p.tunnels.splice(i, 1)}><Icon name="trash" /></button>
            </div>
          {/each}
        {:else}
          <p class="hint">Tunnels start with every login. Add one to forward a local or remote port, or a SOCKS proxy.</p>
        {/if}
      </section>
    </div>

    <aside>
      <div class="phead"><h3 class="label">SSH config preview</h3><span class="hint">Written to ~/.ssh/sssh.d/config on save</span></div>
      <pre class="mono">{#each preview.config.split("\n") as line}{@const m = line.match(/^(\s*)(\S+)(.*)$/)}{#if m}{m[1]}<span class={m[1] ? "k2" : "k1"}>{m[2]}</span>{m[3]}{"\n"}{/if}{/each}</pre>
      {#if started}
        <div class="notice" class:good={!preview.problem} class:bad={!!preview.problem}>{preview.problem || (p.jump ? "Valid. Jump chain resolves." : "Valid.")}</div>
      {/if}
    </aside>
    <datalist id="folders">{#each folders as f}<option value={f}></option>{/each}</datalist>
  </form>
  {#if error}<div class="notice bad fail" role="alert">{error}</div>{/if}

  {#snippet footer()}
    <span class="cli hint">Connect from the CLI <code class="mono">sssh {p.name || "name"}</code></span>
    {#if !isNew}<button type="button" class="btn ghost danger" onclick={ondelete}>Delete</button>{/if}
    <button type="button" class="btn ghost" onclick={oncancel}>Cancel</button>
    <button type="submit" form={formId} class="btn primary" disabled={busy} title="⌘↵">{isNew ? "Add connection" : "Save changes"}</button>
  {/snippet}
</Modal>
{#if revealing}
  <RevealPassword connection={oldName} onclose={() => (revealing = false)} />
{/if}
{#if vaultPromptMode}
  <VaultPanel
    mode={vaultPromptMode}
    onclose={() => { vaultPromptMode = null; pendingPassword = ""; busy = false; }}
    onsuccess={onVaultPromptSuccess}
  />
{/if}

<style>
  .cols { flex: 1; min-height: 0; display: flex; }
  .left { flex: 1; min-width: 0; overflow: auto; }
  section { display: grid; gap: var(--s3); padding: var(--s5); border-bottom: 1px solid var(--line); }
  section:last-child { border-bottom: 0; }
  h3 { margin: 0; }
  .g2 { display: grid; grid-template-columns: 1fr 1fr; gap: var(--s3); }
  .g3 { display: grid; grid-template-columns: 1fr 96px 160px; gap: var(--s3); }
  .seg { width: fit-content; }
  .inline { display: flex; gap: var(--s2); }
  .inline select, .inline input { flex: 1; min-width: 0; }
  .hint { margin: 0; }
  .saved { display: flex; align-items: center; justify-content: space-between; gap: var(--s3); padding: var(--s2) var(--s3); background: var(--field); border: 1px solid var(--fieldline); border-radius: var(--r-sm); }
  .acts { display: flex; gap: var(--s2); }
  .tags { min-height: var(--h-md); display: flex; flex-wrap: wrap; align-items: center; gap: var(--s1); padding: 3px var(--s2); background: var(--field); border: 1px solid var(--fieldline); border-radius: var(--r-sm); }
  .tags:focus-within { border-color: var(--accent); box-shadow: 0 0 0 3px var(--accent-ring); }
  .tags .chip { padding-right: 2px; }
  .tags .chip .icon-btn { width: 16px; height: 16px; border-radius: var(--r-xs); }
  .tags input, .tags input:focus { flex: 1; min-width: 72px; height: 24px; padding: 0 var(--s1); border: 0; background: none; box-shadow: none; }
  .route { display: flex; align-items: center; flex-wrap: wrap; gap: var(--s2); color: var(--muted); }
  .opt { display: flex; align-items: center; justify-content: space-between; gap: var(--s4); }
  .opt > div { display: grid; gap: 2px; }
  .opt .t { color: var(--text); }
  .opt select { flex: none; width: 150px; }
  .head { display: flex; align-items: center; justify-content: space-between; }
  .tun { display: grid; grid-template-columns: 110px 1fr 1.4fr 24px; gap: var(--s2); align-items: center; }
  aside { width: 300px; flex: none; display: flex; flex-direction: column; gap: var(--s3); padding: var(--s5); overflow: auto; background: var(--bar); border-left: 1px solid var(--line); }
  .phead { display: grid; gap: 2px; }
  pre { margin: 0; padding: var(--s3); background: var(--strip); border: 1px solid var(--line); border-radius: var(--r-sm); font-size: var(--t-xs); line-height: 1.7; white-space: pre; overflow-x: auto; }
  .k1 { color: var(--accent-text); } .k2 { color: var(--muted); }
  .fail { flex: none; margin: 0 var(--s5) var(--s3); }
  .cli { margin-right: auto; display: flex; align-items: center; gap: var(--s2); }
  .cli code { padding: 2px var(--s2); background: var(--strip); border: 1px solid var(--line); border-radius: var(--r-xs); color: var(--text2); }
</style>
