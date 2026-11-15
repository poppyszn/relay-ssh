// Runs fn every `ms` while the window is visible, and once more when it becomes visible
// again. A hidden window does no work, so background polling never keeps ssh busy.
export function startPolling(fn: () => void, ms: number): () => void {
  const tick = () => { if (!document.hidden) fn(); };
  const timer = setInterval(tick, ms);
  document.addEventListener("visibilitychange", tick);
  return () => {
    clearInterval(timer);
    document.removeEventListener("visibilitychange", tick);
  };
}
