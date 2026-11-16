// @vitest-environment happy-dom
import { flushSync, mount, unmount } from "svelte";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Toaster from "./Toaster.svelte";
import { toast } from "../state/toasts.svelte";

let c: ReturnType<typeof mount> | null = null;

beforeEach(() => {
  vi.useFakeTimers();
  toast.clear();
});

afterEach(() => {
  toast.clear();
  if (c) unmount(c);
  c = null;
  document.body.innerHTML = "";
  vi.useRealTimers();
});

function show(theme: "dark" | "light" = "dark") {
  const target = document.createElement("div");
  document.body.append(target);
  c = mount(Toaster, { target, props: { theme } });
  flushSync();
  return target;
}

describe("Toaster & toast store", () => {
  it("renders toasts with status role, polite live region, message, and description", () => {
    const t = show();
    toast("Relay 0.1.1 is available", { description: "Install it from Settings." });
    flushSync();

    const item = t.querySelector<HTMLElement>(".toast[role='status']");
    expect(item).not.toBeNull();
    expect(item?.getAttribute("aria-live")).toBe("polite");
    expect(item?.textContent).toContain("Relay 0.1.1 is available");
    expect(item?.textContent).toContain("Install it from Settings.");
  });

  it("auto-dismisses after default timeout and stays open longer when an action exists", () => {
    const t = show();
    toast.success("Copied hostname");
    toast("Update ready", {
      action: { label: "Open Settings", onClick: () => {} },
    });
    flushSync();
    expect(t.querySelectorAll(".toast")).toHaveLength(2);

    vi.advanceTimersByTime(4500);
    flushSync();
    const remaining = [...t.querySelectorAll(".toast")].map((el) => el.textContent ?? "");
    expect(remaining).toHaveLength(1);
    expect(remaining[0]).toContain("Update ready");

    vi.advanceTimersByTime(4500);
    flushSync();
    expect(t.querySelectorAll(".toast")).toHaveLength(0);
  });

  it("invokes action onClick and dismisses the toast", () => {
    const t = show();
    const onClick = vi.fn();
    toast("Relay 0.1.1 is available", { action: { label: "Open Settings", onClick } });
    flushSync();

    const actionBtn = [...t.querySelectorAll("button")].find((b) => b.textContent?.trim() === "Open Settings")!;
    expect(actionBtn).toBeTruthy();
    actionBtn.click();
    flushSync();

    expect(onClick).toHaveBeenCalledTimes(1);
    expect(t.querySelectorAll(".toast")).toHaveLength(0);
  });

  it("dismisses a specific toast via its close button and preserves stack order", () => {
    const t = show();
    toast("First notice");
    toast.error("Second notice");
    toast.success("Third notice");
    flushSync();

    const titles = () => [...t.querySelectorAll(".toast-title")].map((el) => el.textContent);
    expect(titles()).toEqual(["First notice", "Second notice", "Third notice"]);

    const closeBtns = t.querySelectorAll<HTMLButtonElement>('button[aria-label="Dismiss notification"]');
    closeBtns[1].click();
    flushSync();

    expect(titles()).toEqual(["First notice", "Third notice"]);
  });
});
