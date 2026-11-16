<script lang="ts">
  import { onMount } from "svelte";
  import { Switch } from "bits-ui";
  import { settings } from "../state/settings.svelte";
  import { updates } from "../state/updates.svelte";

  onMount(() => { updates.load(); });

  const busy = $derived(updates.phase === "checking" || updates.phase === "installing");
</script>

<section id="settings-updates" aria-labelledby="set-updates">
  <div class="head">
    <h3 class="label" id="set-updates">Updates</h3>
    {#if updates.current}<span class="hint mono">Version {updates.current}</span>{/if}
  </div>

  {#if !updates.enabled}
    <p class="hint lead">This is a development build, so it does not check for updates. Released builds do.</p>
  {:else}
    <p class="hint lead">Updates are signed. Relay checks the signature against a key built into the app and refuses anything that does not match.</p>
  {/if}

  {#if updates.phase === "uptodate"}<div class="notice good" role="status">Relay is up to date.</div>{/if}
  {#if updates.phase === "available"}
    <div class="notice" role="status">
      <strong>Relay {updates.latest} is available.</strong>
      {#if updates.notes}<p class="notes">{updates.notes}</p>{/if}
    </div>
  {/if}
  {#if updates.phase === "installing"}<div class="notice" role="status">Downloading and verifying the update…</div>{/if}
  {#if updates.phase === "ready"}<div class="notice good" role="status">Relay {updates.latest} is ready. Restart to finish; open sessions will close.</div>{/if}
  {#if updates.phase === "error"}<div class="notice bad" role="alert">{updates.error}</div>{/if}

  <div class="actions">
    {#if updates.phase === "available" || (updates.phase === "error" && updates.latest)}
      <button type="button" class="btn primary" onclick={() => updates.install()}>Download and install</button>
    {:else if updates.phase === "ready"}
      <button type="button" class="btn primary" onclick={() => updates.restart()}>Restart Relay</button>
    {/if}
    <button type="button" class="btn" disabled={!updates.enabled || busy} onclick={() => updates.check()}>
      {updates.phase === "checking" ? "Checking…" : "Check for updates"}
    </button>
  </div>

  <div class="opt">
    <div class="txt"><span id="set-autoupdate">Check for updates on launch</span><span class="hint">One request to GitHub a few seconds after Relay starts. Nothing about you or your connections is sent.</span></div>
    <Switch.Root class="switch" bind:checked={settings.autoUpdateCheck} aria-labelledby="set-autoupdate" />
  </div>
</section>

<style>
  section { display: grid; gap: var(--s4); margin-top: var(--s5); padding-top: var(--s5); border-top: 1px solid var(--hairline); }
  h3 { margin: 0; }
  .head { display: flex; align-items: center; gap: var(--s2); }
  .head .hint { margin-left: var(--s1); }
  .lead { margin: 0; }
  .actions { display: flex; flex-wrap: wrap; align-items: center; gap: var(--s2); }
  .opt { display: flex; align-items: flex-start; justify-content: space-between; gap: var(--s4); }
  .opt .txt { display: grid; gap: 2px; }
  .opt :global(.switch) { margin-top: 1px; }
  .notes { margin: 6px 0 0; white-space: pre-wrap; font-weight: 400; }
</style>
