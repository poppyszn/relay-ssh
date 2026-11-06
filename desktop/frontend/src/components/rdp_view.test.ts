// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { describe, expect, it, vi } from "vitest";

// These tests are about the view, not the 5 MB client, which exists only after `make rdp-client`.
vi.mock("../lib/rdp_client", async (orig) => ({ ...(await orig<typeof import("../lib/rdp_client")>()), loadRdpClient: async () => null }));
import RdpView from "./RdpView.svelte";
import { formatRdpError } from "../lib/rdp_errors";
import { RdpService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

describe("RdpView component", () => {
  it("formats plain language errors correctly for all 8 failure causes", () => {
    // 1. Jump host requires authentication
    const jumpErr = formatRdpError("Open bastion first");
    expect(jumpErr.title).toBe("Jump host requires authentication");
    expect(jumpErr.hint).toContain("bastion");

    // 2. Jump host unreachable
    const jumpUnreach = formatRdpError("jump host unreachable");
    expect(jumpUnreach.title).toBe("Jump host unreachable");
    expect(jumpUnreach.hint).toContain("online");

    // 3. Port closed or unreachable
    const refErr = formatRdpError("dial tcp 127.0.0.1:3389: connection refused");
    expect(refErr.title).toBe("Port closed or unreachable");
    expect(refErr.hint).toContain("3389");

    // 4. Host unreachable
    const unreachErr = formatRdpError("no route to host");
    expect(unreachErr.title).toBe("Host unreachable");
    expect(unreachErr.hint).toContain("network");

    // 5. Wrong password or NLA refused
    const nlaErr = formatRdpError("NLA logon failed: access denied");
    expect(nlaErr.title).toBe("Login failed (wrong password or NLA refused)");
    expect(nlaErr.hint).toContain("credentials");

    // 6. Certificate changed
    const certErr = formatRdpError("certificate mismatch");
    expect(certErr.title).toBe("Host certificate changed or rejected");
    expect(certErr.hint).toContain("certificate");

    // 7. Timed out
    const timeoutErr = formatRdpError("connection timed out");
    expect(timeoutErr.title).toBe("Connection timed out");
    expect(timeoutErr.hint).toContain("latency");

    // 8. Server closed the session
    const closedErr = formatRdpError("connection reset by peer");
    expect(closedErr.title).toBe("Session closed by server");
    expect(closedErr.hint).toContain("terminated");
  });

  it("mounts in connecting state and prompts for password if none saved", async () => {
    vi.spyOn(RdpService, "Open").mockResolvedValue({
      sessionId: "test-sess-1",
      wsUrl: "ws://127.0.0.1:1234/rdp",
      token: "tok-1",
      host: "win-test",
      user: "Administrator",
      domain: "CORP",
      port: 3389,
      hasSavedPassword: false,
    });

    const target = document.createElement("div");
    const c = mount(RdpView, {
      target,
      props: {
        target: "win-box",
        prof: {
          name: "win-box",
          protocol: "rdp",
          host: "win-test",
          user: "Administrator",
          domain: "CORP",
          port: 3389,
          auth: "",
          key: "",
          folder: "",
          tags: [],
          jump: "",
          forwardAgent: false,
          blocks: "",
          tunnels: [],
        },
      },
    });
    flushSync();
    await new Promise((r) => setTimeout(r, 20));
    flushSync();

    expect(target.textContent).toContain("Enter Windows password");
    expect(target.textContent).toContain("CORP\\Administrator@win-test");
    expect(target.querySelector('input[type="password"]')).not.toBeNull();

    unmount(c);
  });

  it("shows friendly error on connection failure", async () => {
    vi.spyOn(RdpService, "Open").mockRejectedValue(new Error("Open bastion first"));

    const target = document.createElement("div");
    const c = mount(RdpView, {
      target,
      props: {
        target: "win-box-err",
        prof: {
          name: "win-box-err",
          protocol: "rdp",
          host: "10.0.0.5",
          user: "admin",
          port: 3389,
          auth: "",
          key: "",
          folder: "",
          tags: [],
          jump: "bastion",
          forwardAgent: false,
          blocks: "",
          tunnels: [],
        },
      },
    });
    flushSync();
    await new Promise((r) => setTimeout(r, 20));
    flushSync();

    expect(target.textContent).toContain("Jump host requires authentication");
    expect(target.textContent).toContain("Open a connection to your jump host");
    expect(target.textContent).toContain("Reconnect");

    unmount(c);
  });

  it("never pretends to be connected when the RDP client is not built into the app", async () => {
    vi.spyOn(RdpService, "Open").mockResolvedValue({ sessionId: "s-nowasm", wsUrl: "ws://127.0.0.1:1/rdp", token: "t", host: "win-test", user: "u", domain: "", port: 3389, hasSavedPassword: true });
    vi.spyOn(RdpService, "Credentials").mockResolvedValue("pw");
    const close = vi.spyOn(RdpService, "Close").mockResolvedValue(undefined as never);
    const target = document.createElement("div");
    const c = mount(RdpView, { target, props: { target: "win-box", prof: { name: "win-box", protocol: "rdp", host: "win-test", user: "u", port: 3389, auth: "", key: "", folder: "", tags: [], jump: "", forwardAgent: false, blocks: "", tunnels: [] } } });
    flushSync();
    await new Promise((r) => setTimeout(r, 50));
    flushSync();
    expect(target.querySelector("canvas")).toBeNull();
    expect(target.textContent).not.toContain("RDP Connected");
    expect(target.textContent).toContain("Remote desktop is not built into this version");
    expect(close).toHaveBeenCalledWith("s-nowasm");
    unmount(c);
  });
});
