// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { describe, expect, it, vi } from "vitest";
import RdpView from "./RdpView.svelte";
import { RdpService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

describe("RdpView component", () => {
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

    expect(target.textContent).toContain("Could not connect");
    expect(target.textContent).toContain("Open bastion first");
    expect(target.textContent).toContain("Reconnect");

    unmount(c);
  });
});
