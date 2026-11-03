<script lang="ts">
  import Modal from "./Modal.svelte";
  import { ProfilesService, type Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

  /** Confirms deleting a connection. One that others reach through (their jump host) cannot be
   * deleted, so the dialog says which ones and offers no Delete, instead of failing on save. */
  let { profile, profiles, ondeleted, onclose }: { profile: Profile; profiles: Profile[]; ondeleted: () => void; onclose: () => void } = $props();

  const users = $derived(profiles.filter((p) => p.jump === profile.name).map((p) => p.name));
  let error = $state("");
  let busy = $state(false);

  async function remove() {
    busy = true;
    error = "";
    try {
      await ProfilesService.Delete(profile.name);
      ondeleted();
    } catch (e) {
      error = (e as Error)?.message ?? String(e);
    } finally {
      busy = false;
    }
  }
</script>

<Modal title="Delete connection" subtitle={profile.name} size="sm" role="alertdialog" {onclose}>
  {#if users.length}
    <p class="text"><strong>{profile.name}</strong> is the jump host for {users.length === 1 ? "another connection" : `${users.length} connections`}, so it can't be deleted yet:</p>
    <ul class="mono">{#each users as u (u)}<li>{u}</li>{/each}</ul>
    <p class="text hint">Change their jump host or delete them first.</p>
  {:else}
    <p class="text">Delete <strong>{profile.name}</strong>? This removes it from connections.toml and your SSH config.</p>
  {/if}
  {#if error}<div class="notice bad" role="alert">{error}</div>{/if}
  {#snippet footer()}
    {#if users.length}
      <button type="button" class="btn primary" data-autofocus onclick={onclose}>OK</button>
    {:else}
      <button type="button" class="btn" data-autofocus onclick={onclose}>Cancel</button>
      <button type="button" class="btn danger" disabled={busy} onclick={remove}>Delete</button>
    {/if}
  {/snippet}
</Modal>

<style>
  .text { margin: 0; line-height: 1.5; }
  .hint { margin-top: var(--s3); }
  ul { margin: var(--s2) 0 0; padding-left: var(--s5); font-size: var(--t-sm); color: var(--text2); }
  .notice { margin-top: var(--s3); }
</style>
