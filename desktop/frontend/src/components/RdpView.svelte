<script lang="ts">
  import { onDestroy, onMount, tick } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { RdpService, type Profile, type RdpSessionInfo } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { formatRdpError } from "../lib/rdp_errors";
  import { buildConfig, loadRdpClient, mountClient, type RdpUi } from "../lib/rdp_client";
  import Icon from "./Icon.svelte";
  import Modal from "./Modal.svelte";

  let { target, prof }: { target: string; prof?: Profile } = $props();

  type RdpState = "connecting" | "certificate_prompt" | "password_prompt" | "connected" | "reconnect" | "error";

  interface CertInfo {
    sessionId: string;
    endpoint: string;
    fingerprint: string;
    storedFingerprint: string;
    isMismatch: boolean;
  }

  let viewState = $state<RdpState>("connecting");
  let errorMessage = $state("");
  let errorHint = $state("");
  let sessionId = $state("");
  let host = $state<HTMLElement | undefined>(undefined);

  let certEvent = $state<CertInfo | null>(null);
  let manualPassword = $state("");
  let ui: RdpUi | null = null;
  let resizeObserver: ResizeObserver | null = null;
  let resizeTimer: ReturnType<typeof setTimeout> | undefined;
  let unlistenCert: (() => void) | null = null;
  // The session Open returned while the password was being typed. Its token only lives 60 seconds.
  let pending: { info: RdpSessionInfo; at: number } | null = null;
  const TOKEN_LIFETIME_MS = 45_000;

  function fail(err: unknown) {
    const e = err as { backtrace?: () => string; message?: string };
    const formatted = formatRdpError(e?.backtrace?.() ?? e?.message ?? String(err));
    errorMessage = formatted.title;
    errorHint = formatted.hint;
    viewState = "error";
  }

  function teardown() {
    resizeObserver?.disconnect();
    resizeObserver = null;
    clearTimeout(resizeTimer);
    try { ui?.shutdown(); } catch { /* already closed */ }
    ui = null;
    host?.replaceChildren();
  }

  function closeSession() {
    if (sessionId) RdpService.Close(sessionId).catch(() => {});
    sessionId = "";
  }

  async function connectSession() {
    teardown();
    viewState = "connecting";
    errorMessage = "";
    errorHint = "";
    try {
      const info = await RdpService.Open(target);
      sessionId = info.sessionId;

      let password = manualPassword;
      if (!password && info.hasSavedPassword) {
        try {
          password = await RdpService.Credentials(sessionId);
        } catch {
          // If vault retrieval fails, we prompt the user
        }
      }

      if (!password) {
        pending = { info, at: Date.now() };
        viewState = "password_prompt";
        return;
      }

      await startRdpClient(info, password);
    } catch (err) {
      fail(err);
    }
  }

  async function startRdpClient(info: RdpSessionInfo, password: string) {
    try {
      const client = await loadRdpClient();
      if (!client) {
        // Never draw a "connected" screen without a real client: say so, and free the tunnel and gateway session.
        closeSession();
        errorMessage = "Remote desktop is not built into this version";
        errorHint = "This copy of Relay was built without its RDP client, so it cannot open remote desktops. Build it with `make rdp-client` (see desktop/README.md), or use an SSH connection.";
        viewState = "error";
        return;
      }
      await tick();
      ui = await mountClient(host!, client);
      ui.setEnableClipboard(true);
      const rect = host!.getBoundingClientRect();
      const config = buildConfig(ui, client, {
        username: info.user,
        password,
        domain: info.domain,
        host: info.host,
        port: info.port,
        wsUrl: info.wsUrl,
        token: info.token,
        width: Math.max(640, Math.floor(rect.width)),
        height: Math.max(480, Math.floor(rect.height)),
      });
      // The gateway pauses here while a certificate prompt is answered (see the rdp:cert event).
      const session = await ui.connect(config);
      viewState = "connected";
      ui.setVisibility(true);
      watchSize();
      const end = await session.run();
      errorMessage = "Session ended";
      errorHint = end.reason();
      viewState = "reconnect";
    } catch (err) {
      if (viewState !== "error") fail(err);
    } finally {
      if (viewState === "error" || viewState === "reconnect") {
        teardown();
        closeSession();
      }
    }
  }

  // The remote desktop follows the tab's size (display control), a moment after the resizing stops.
  function watchSize() {
    if (!host) return;
    resizeObserver = new ResizeObserver(() => {
      clearTimeout(resizeTimer);
      resizeTimer = setTimeout(() => {
        const { width, height } = host!.getBoundingClientRect();
        if (ui && width > 0 && height > 0) {
          try { ui.resize(Math.floor(width), Math.floor(height)); } catch { /* the server cannot resize */ }
        }
      }, 250);
    });
    resizeObserver.observe(host);
  }

  function cancel() {
    teardown();
    closeSession();
    errorMessage = "Connection cancelled";
    errorHint = "The connection attempt was cancelled.";
    viewState = "error";
  }

  async function acceptCert(accept: boolean) {
    if (!sessionId) return;
    await RdpService.TrustCert(sessionId, accept);
    certEvent = null;
    if (accept) {
      viewState = "connecting";
    } else {
      viewState = "error";
      errorMessage = "Certificate rejected by user";
      errorHint = "The remote desktop connection was aborted because the TLS certificate was not trusted.";
    }
  }

  function submitPassword(e?: Event) {
    e?.preventDefault();
    if (!manualPassword) return;
    const p = pending;
    pending = null;
    if (p && Date.now() - p.at < TOKEN_LIFETIME_MS) {
      viewState = "connecting";
      startRdpClient(p.info, manualPassword);
    } else {
      // Typing took longer than the gateway token lives: start over with a fresh session.
      closeSession();
      connectSession();
    }
  }

  onMount(() => {
    unlistenCert = Events.On("rdp:cert", (ev: any) => {
      if (ev.data && (!ev.data.sessionId || ev.data.sessionId === sessionId)) {
        certEvent = ev.data;
        viewState = "certificate_prompt";
      }
    });

    connectSession();
  });

  onDestroy(() => {
    if (unlistenCert) unlistenCert();
    teardown();
    closeSession();
  });
</script>

<div class="rdp-view">
  <!-- The client lives here for the whole session. It stays mounted under the prompts and cards below,
       because removing it would end the session (the certificate prompt appears mid-connection). -->
  <div class="stage" class:live={viewState === "connected"} bind:this={host}></div>
  {#if viewState === "connecting"}
    <div class="center-card" role="status">
      <div class="spinner" aria-hidden="true"></div>
      <div class="title">Connecting to {prof?.name ?? target}</div>
      <div class="sub mono">{prof?.user}@{prof?.host}:{prof?.port || 3389}</div>
      <button type="button" class="btn sm" onclick={cancel}>Cancel</button>
    </div>
  {:else if viewState === "password_prompt"}
    <div class="center-card">
      <div class="title">Enter Windows password</div>
      <div class="sub mono">{prof?.domain ? `${prof.domain}\\` : ""}{prof?.user}@{prof?.host}</div>
      <form onsubmit={submitPassword} class="pass-form">
        <label class="field">
          <span>Password</span>
          <input type="password" bind:value={manualPassword} placeholder="Windows password" required aria-label="Password" />
        </label>
        <p class="hint">Stored in memory for this session only; never saved to disk or sent to the backend.</p>
        <div class="acts">
          <button type="submit" class="btn primary">Connect</button>
          <button type="button" class="btn" onclick={() => { viewState = "error"; errorMessage = "Cancelled"; errorHint = "No password entered."; }}>Cancel</button>
        </div>
      </form>
    </div>
  {:else if viewState === "certificate_prompt" && certEvent}
    <Modal title={certEvent.isMismatch ? "Host certificate changed" : "Trust host certificate?"} size="md" dismissOnScrim={false} onclose={() => acceptCert(false)}>
      <div class="cert-modal">
        {#if certEvent.isMismatch}
          <div class="notice bad" role="alert">
            <strong>Warning: Certificate mismatch detected!</strong>
            <p>The TLS certificate presented by <code>{certEvent.endpoint}</code> does not match the stored fingerprint. This could indicate a man-in-the-middle attack or a server reinstallation.</p>
          </div>
          <div class="fp-box">
            <span class="label">Stored fingerprint:</span>
            <code class="fp mono">{certEvent.storedFingerprint}</code>
            <span class="label">New fingerprint:</span>
            <code class="fp mono">{certEvent.fingerprint}</code>
          </div>
          <div class="modal-acts">
            <button type="button" class="btn danger" onclick={() => acceptCert(true)}>Trust new certificate</button>
            <button type="button" class="btn primary" onclick={() => acceptCert(false)}>Disconnect</button>
          </div>
        {:else}
          <p>This is the first time connecting to <code>{certEvent.endpoint}</code>. Trust on first use?</p>
          <div class="fp-box">
            <span class="label">SHA-256 fingerprint:</span>
            <code class="fp mono">{certEvent.fingerprint}</code>
          </div>
          <div class="modal-acts">
            <button type="button" class="btn primary" onclick={() => acceptCert(true)}>Trust &amp; Connect</button>
            <button type="button" class="btn" onclick={() => acceptCert(false)}>Reject</button>
          </div>
        {/if}
      </div>
    </Modal>
  {:else if viewState === "connected"}
    <div class="bar" role="toolbar" aria-label="Remote desktop">
      <span class="dot ok" aria-hidden="true"></span>
      <span class="mono who">{prof?.user}@{prof?.host}</span>
      <button type="button" class="btn ghost sm" title="Send Ctrl+Alt+Delete to the remote desktop" onclick={() => ui?.ctrlAltDel()}>Ctrl+Alt+Del</button>
      <button type="button" class="btn ghost sm" onclick={cancel}>Disconnect</button>
    </div>
  {:else if viewState === "error" || viewState === "reconnect"}
    <div class="center-card" role="alert">
      <div class="err-icon"><Icon name="warning" size={20} /></div>
      <div class="title">{errorMessage || "Session ended"}</div>
      {#if errorHint}<div class="hint-text">{errorHint}</div>{/if}
      <button type="button" class="btn primary sm" onclick={connectSession}>Reconnect</button>
    </div>
  {/if}
</div>

<style>
  .rdp-view { width: 100%; height: 100%; display: flex; flex-direction: column; background: var(--bg); position: relative; overflow: hidden; }
  .center-card { position: relative; z-index: 1; margin: auto; width: min(440px, 90%); padding: var(--s5); background: var(--float); border: 1px solid var(--floatline); border-radius: var(--r-md); box-shadow: var(--e-modal); display: flex; flex-direction: column; align-items: center; gap: var(--s3); text-align: center; }
  .title { font-weight: 600; font-size: var(--t-md); color: var(--strong); }
  .sub { font-size: var(--t-xs); color: var(--muted); }
  .hint-text { font-size: var(--t-sm); color: var(--muted); line-height: 1.5; margin: var(--s1) 0; }
  .spinner { width: 28px; height: 28px; border: 3px solid var(--line); border-top-color: var(--accent); border-radius: 50%; animation: spin 0.8s linear infinite; }
  @keyframes spin { to { transform: rotate(360deg); } }
  @media (prefers-reduced-motion: reduce) { .spinner { animation: none; border-top-color: var(--line); } }
  .pass-form { width: 100%; display: flex; flex-direction: column; gap: var(--s3); text-align: left; }
  .acts { display: flex; justify-content: flex-end; gap: var(--s2); margin-top: var(--s2); }
  .cert-modal { display: flex; flex-direction: column; gap: var(--s4); }
  .fp-box { display: flex; flex-direction: column; gap: var(--s1); padding: var(--s3); background: var(--field); border: 1px solid var(--fieldline); border-radius: var(--r-sm); font-size: var(--t-xs); }
  .fp { word-break: break-all; color: var(--strong); font-size: var(--t-xs); }
  .modal-acts { display: flex; justify-content: flex-end; gap: var(--s2); }
  /* Hidden (not removed) until the session is up; the prompts and cards above it are positioned over it. */
  .stage { position: absolute; inset: 0; visibility: hidden; }
  .stage.live { visibility: visible; }
  .bar { position: absolute; top: var(--s2); right: var(--s2); z-index: 2; display: flex; align-items: center; gap: var(--s1); padding: 2px var(--s2) 2px var(--s3); background: var(--float); border: 1px solid var(--floatline); border-radius: var(--r-sm); font-size: var(--t-xs); color: var(--muted); opacity: 0.35; transition: opacity var(--dur-2) var(--ease); }
  .bar:hover, .bar:focus-within { opacity: 1; }
  .who { margin-right: var(--s1); }
  .err-icon { color: var(--warn); }
</style>
