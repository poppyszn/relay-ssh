import { openSession, type Session } from "./terminal";
import { TerminalService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));

// Measures how late timers and animation frames fire while a flood is running:
// the direct signal of whether the UI thread stays responsive.
function lagMonitor() {
  let maxTimer = 0, maxFrame = 0, last = performance.now(), lastFrame = last, run = true;
  const tick = () => {
    const now = performance.now();
    maxTimer = Math.max(maxTimer, now - last - 5);
    last = now;
    if (run) setTimeout(tick, 5);
  };
  const frame = () => {
    const now = performance.now();
    maxFrame = Math.max(maxFrame, now - lastFrame);
    lastFrame = now;
    if (run) requestAnimationFrame(frame);
  };
  setTimeout(tick, 5);
  requestAnimationFrame(frame);
  return () => { run = false; return { maxTimerLagMs: Math.round(maxTimer), maxFrameGapMs: Math.round(maxFrame) }; };
}

// Run a command and wait for its sentinel in the byte stream, then for xterm
// to finish parsing the chunk that carried it.
async function timed(s: Session, cmd: string) {
  const dec = new TextDecoder();
  let tail = "", bytes = 0, tReceived = 0, tDone = 0;
  const done = new Promise<void>((res) => {
    s.onChunk = (d) => {
      bytes += d.length;
      // Search before trimming: a coalesced frame can carry the marker plus
      // more than 64 chars of prompt after it.
      const text = tail + dec.decode(d, { stream: true });
      tail = text.slice(-64);
      if (!tReceived && text.includes("__DONE_42")) {
        tReceived = performance.now();
        s.term.write("", () => { tDone = performance.now(); res(); });
      }
    };
  });
  const stop = lagMonitor();
  const t0 = performance.now();
  s.ws.send(new TextEncoder().encode(`${cmd}; echo __DONE_$((6*7))\n`));
  await done;
  return {
    cmd,
    bytes,
    receivedMs: Math.round(tReceived - t0),
    renderedMs: Math.round(tDone - t0),
    ...stop(),
  };
}

async function echoRtt(s: Session, n = 20) {
  const times: number[] = [];
  for (let i = 0; i < n; i++) {
    const t0 = performance.now();
    await new Promise<void>((res) => {
      s.onChunk = () => { s.onChunk = undefined; res(); };
      s.ws.send(new TextEncoder().encode("x"));
    });
    times.push(performance.now() - t0);
    await sleep(30);
  }
  s.ws.send(new TextEncoder().encode("\x15"));
  times.sort((a, b) => a - b);
  return { medianMs: +times[n >> 1].toFixed(1), maxMs: +times[n - 1].toFixed(1) };
}

export async function runBench(el: HTMLElement, mode: string) {
  const out: Record<string, unknown> = { mode, userAgent: navigator.userAgent, origin: location.origin };
  try {
    const s = await openSession(el, "local", 100000);
    out.webgl = s.webgl;
    await sleep(1500);
    if (mode === "idle") {
      const hidden = [];
      for (let i = 0; i < 4; i++) {
        const d = document.createElement("div");
        d.style.cssText = "display:none;width:800px;height:400px";
        document.body.appendChild(d);
        hidden.push(await openSession(d, "local"));
      }
      out.sessions = 5;
      await sleep(25000); // external sampler measures CPU in this window
    } else {
      out.idleEcho = await echoRtt(s);
      out.seq5M = await timed(s, "seq 1 5000000");
      const t0 = performance.now();
      s.term.selectAll();
      const sel = s.term.getSelection();
      out.select100k = { lines: sel.split("\n").length, ms: Math.round(performance.now() - t0) };
      await sleep(500);
      await timed(s, `[ -f /tmp/relay-200m.txt ] || yes "the quick brown fox jumps over the lazy dog 0123456789" | head -c 200000000 > /tmp/relay-200m.txt`);
      out.cat200M = await timed(s, "cat /tmp/relay-200m.txt");
      await sleep(500);
      out.echoAfterFlood = await echoRtt(s);
    }
  } catch (e) {
    out.error = String(e);
  }
  await TerminalService.ReportBench(JSON.stringify(out));
}
