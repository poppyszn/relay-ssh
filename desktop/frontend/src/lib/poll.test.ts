// @vitest-environment happy-dom
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { startPolling } from "./poll";

let hidden = false;
beforeEach(() => {
  vi.useFakeTimers();
  hidden = false;
  vi.spyOn(document, "hidden", "get").mockImplementation(() => hidden);
});
afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
});

describe("startPolling", () => {
  it("runs on the interval and stops when cancelled", () => {
    const fn = vi.fn();
    const stop = startPolling(fn, 1000);
    vi.advanceTimersByTime(3000);
    expect(fn).toHaveBeenCalledTimes(3);
    stop();
    vi.advanceTimersByTime(3000);
    expect(fn).toHaveBeenCalledTimes(3);
  });

  it("does no work while the window is hidden", () => {
    const fn = vi.fn();
    const stop = startPolling(fn, 1000);
    hidden = true;
    vi.advanceTimersByTime(5000);
    expect(fn).not.toHaveBeenCalled();
    stop();
  });

  it("refreshes once as soon as the window is visible again", () => {
    const fn = vi.fn();
    const stop = startPolling(fn, 60_000);
    hidden = true;
    document.dispatchEvent(new Event("visibilitychange"));
    expect(fn).not.toHaveBeenCalled();
    hidden = false;
    document.dispatchEvent(new Event("visibilitychange"));
    expect(fn).toHaveBeenCalledTimes(1);
    stop();
    document.dispatchEvent(new Event("visibilitychange"));
    expect(fn).toHaveBeenCalledTimes(1);
  });
});
