<script lang="ts">
  import { onMount } from "svelte";
  import { openSession, type Session } from "./terminal";
  import { runBench } from "./bench";
  import { TerminalService } from "../bindings/github.com/poppyszn/super-ssh-v2/desktop";

  let host: HTMLElement;
  let target = $state("local");
  let session: Session | undefined;
  let error = $state("");

  async function connect() {
    error = "";
    session?.dispose();
    host.replaceChildren();
    try {
      session = await openSession(host, target);
      session.term.focus();
    } catch (e) {
      error = String(e);
    }
  }

  onMount(async () => {
    const mode = await TerminalService.BenchMode();
    if (mode) await runBench(host, mode);
    else await connect();
  });
</script>

<div class="bar">
  <input bind:value={target} placeholder="local or profile name" onkeydown={(e) => e.key === "Enter" && connect()} />
  <button onclick={connect}>Connect</button>
  <span class="err">{error}</span>
</div>
<div class="term" bind:this={host}></div>

<style>
  .bar { display: flex; gap: 8px; padding: 8px 12px 8px 80px; align-items: center; }
  .term { position: absolute; inset: 44px 0 0 0; padding: 0 8px; }
  input { background: #1d1d24; color: inherit; border: 1px solid #333; border-radius: 6px; padding: 4px 8px; }
  button { background: #2d5bd9; color: #fff; border: 0; border-radius: 6px; padding: 5px 12px; }
  .err { color: #f66; }
</style>
