<script lang="ts">
  import { onDestroy, onMount, tick } from "svelte";
  import { Events } from "@wailsio/runtime";
  import { RdpService, type Profile } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";
  import { formatRdpError } from "../lib/rdp_errors";
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
  let canvasContainer = $state<HTMLElement | undefined>(undefined);
  let canvas = $state<HTMLCanvasElement | undefined>(undefined);
  let canvasFocused = $state(false);

  let certEvent = $state<CertInfo | null>(null);
  let manualPassword = $state("");
  let wasmClient: any = null;
  let resizeObserver: ResizeObserver | null = null;
  let unlistenCert: (() => void) | null = null;

  const MAX_CLIPBOARD = 1024 * 1024; // 1 MiB limit

  async function connectSession() {
    viewState = "connecting";
    errorMessage = "";
    errorHint = "";
    try {
      const info = await RdpService.Open(target);
      sessionId = info.sessionId;

      let password = "";
      if (info.hasSavedPassword) {
        try {
          password = await RdpService.Credentials(sessionId);
        } catch {
          // If vault retrieval fails, we prompt the user
        }
      }

      if (!password && !manualPassword) {
        viewState = "password_prompt";
        return;
      }

      await startRdpClient(info, password || manualPassword);
    } catch (err: any) {
      const formatted = formatRdpError(err?.message ?? String(err));
      errorMessage = formatted.title;
      errorHint = formatted.hint;
      viewState = "error";
    }
  }

  async function startRdpClient(info: any, pass: string) {
    try {
      // Dynamic import of IronRDP WASM client wrapper
      const wasmPath = "../wasm/ironrdp.js";
      const ironrdp = await import(/* @vite-ignore */ wasmPath).catch(() => null);

      if (!ironrdp) {
        // Never draw a "connected" screen without a real client: say so, and free the tunnel and gateway session.
        RdpService.Close(info.sessionId).catch(() => {});
        errorMessage = "Remote desktop is not built into this version";
        errorHint = "This copy of Relay was built without its RDP client, so it cannot open remote desktops. Build it with the IronRDP client (see desktop/README.md), or use an SSH connection.";
        viewState = "error";
        return;
      }

      // Initialize IronRDP client
      viewState = "connected";
      await tick();

      if (canvas && canvasContainer) {
        const rect = canvasContainer.getBoundingClientRect();
        canvas.width = Math.max(800, Math.floor(rect.width));
        canvas.height = Math.max(600, Math.floor(rect.height));

        setupResizeObserver();
      }
    } catch (err: any) {
      const formatted = formatRdpError(err?.message ?? String(err));
      errorMessage = formatted.title;
      errorHint = formatted.hint;
      viewState = "error";
    }
  }

  function setupResizeObserver() {
    if (!canvasContainer || !canvas) return;
    resizeObserver = new ResizeObserver((entries) => {
      for (const entry of entries) {
        const { width, height } = entry.contentRect;
        if (width > 0 && height > 0 && canvas) {
          canvas.width = Math.floor(width);
          canvas.height = Math.floor(height);
          if (wasmClient?.resize) {
            wasmClient.resize(canvas.width, canvas.height);
          }
        }
      }
    });
    resizeObserver.observe(canvasContainer);
  }

  function handleKey(e: KeyboardEvent) {
    if (!canvasFocused) return;
    // Cmd combinations on macOS stay with Relay
    if (e.metaKey) {
      return;
    }
    // Prevent default browser shortcuts when canvas has focus
    e.preventDefault();
    e.stopPropagation();

    if (wasmClient?.sendKey) {
      wasmClient.sendKey(e.code, e.type === "keydown");
    }
  }

  function handlePaste(e: ClipboardEvent) {
    if (!canvasFocused) return;
    const text = e.clipboardData?.getData("text/plain") ?? "";
    if (text && text.length <= MAX_CLIPBOARD) {
      if (wasmClient?.sendClipboardText) {
        wasmClient.sendClipboardText(text);
      }
    }
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
    if (manualPassword) {
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
    if (resizeObserver) resizeObserver.disconnect();
    if (sessionId) {
      RdpService.Close(sessionId).catch(() => {});
    }
  });
</script>

<svelte:window onkeydown={handleKey} onkeyup={handleKey} onpaste={handlePaste} />

<div class="rdp-view" bind:this={canvasContainer}>
  {#if viewState === "connecting"}
    <div class="center-card" role="status">
      <div class="spinner" aria-hidden="true"></div>
      <div class="title">Connecting to {prof?.name ?? target}</div>
      <div class="sub mono">{prof?.user}@{prof?.host}:{prof?.port || 3389}</div>
      <button type="button" class="btn sm" onclick={() => { if (sessionId) RdpService.Close(sessionId); viewState = "error"; errorMessage = "Connection cancelled"; errorHint = "The connection attempt was cancelled."; }}>Cancel</button>
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
    <div class="canvas-wrap">
      <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
      <canvas
        bind:this={canvas}
        tabindex="0"
        class="rdp-canvas"
        onfocus={() => (canvasFocused = true)}
        onblur={() => (canvasFocused = false)}
        aria-label="Remote Desktop canvas"
      ></canvas>
      <div class="hud" class:focused={canvasFocused}>
        <span class="dot ok"></span>
        <span class="mono">{prof?.user}@{prof?.host}:{prof?.port || 3389}</span>
        <span class="kb-hint">Keyboard active (⌘ shortcuts stay with Relay)</span>
      </div>
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
  .center-card { margin: auto; width: min(440px, 90%); padding: var(--s5); background: var(--float); border: 1px solid var(--floatline); border-radius: var(--r-md); box-shadow: var(--e-modal); display: flex; flex-direction: column; align-items: center; gap: var(--s3); text-align: center; }
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
  .canvas-wrap { width: 100%; height: 100%; position: relative; display: flex; }
  .rdp-canvas { width: 100%; height: 100%; display: block; outline: none; }
  .rdp-canvas:focus-visible { box-shadow: inset 0 0 0 2px var(--accent); }
  .hud { position: absolute; bottom: var(--s2); left: var(--s2); display: flex; align-items: center; gap: var(--s2); padding: 4px var(--s3); background: var(--float); border: 1px solid var(--line); border-radius: var(--r-sm); font-size: var(--t-xs); color: var(--muted); pointer-events: none; opacity: 0.7; transition: opacity 0.15s ease; }
  .hud.focused { opacity: 1; }
  .kb-hint { border-left: 1px solid var(--line); padding-left: var(--s2); }
  .err-icon { color: var(--warn); }
</style>
