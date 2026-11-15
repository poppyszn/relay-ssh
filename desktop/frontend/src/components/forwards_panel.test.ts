// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, describe, expect, it, vi } from "vitest";
import ForwardsPanel from "./ForwardsPanel.svelte";
import { ForwardsService } from "../../bindings/github.com/poppyszn/relay-ssh/desktop";

let c: ReturnType<typeof mount> | null = null;
afterEach(() => {
  if (c) unmount(c);
  c = null;
  document.body.innerHTML = "";
  vi.restoreAllMocks();
});

async function showEmbedded() {
  vi.spyOn(ForwardsService, "Status").mockResolvedValue({
    connected: true,
    message: "",
    forwards: [
      { type: "local", listen: "127.0.0.1:5432", target: "db.internal:5432", source: "profile", listening: true },
      { type: "dynamic", listen: "127.0.0.1:1080", target: "", source: "runtime", listening: false },
    ],
  } as never);
  const target = document.createElement("div");
  document.body.append(target);
  c = mount(ForwardsPanel, {
    target,
    props: { target: "prod-eu", name: "prod-eu", embedded: true, onclose: () => {}, onopen: () => {}, onchanged: () => {} },
  });
  flushSync();
  await Promise.resolve();
  flushSync();
  return target;
}

describe("ForwardsPanel embedded drawer", () => {
  it("renders a compact diagram without double card chrome and updates tunnel type state", async () => {
    const t = await showEmbedded();
    expect(t.querySelector("figure.diagram.compact")).not.toBeNull();
    expect(t.querySelector(".diagram-card")).toBeNull();

    const seg = t.querySelector<HTMLElement>('[role="group"][aria-label="Tunnel type"]')!;
    const buttons = [...seg.querySelectorAll("button")];
    expect(buttons.map((b) => b.getAttribute("aria-pressed"))).toEqual(["true", "false", "false"]);

    buttons[2].click();
    flushSync();
    expect(buttons.map((b) => b.getAttribute("aria-pressed"))).toEqual(["false", "false", "true"]);
  });
});
