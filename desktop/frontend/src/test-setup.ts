// The Wails runtime phones home to the backend when it loads. There is no backend in tests,
// so every call hangs quietly instead of failing against localhost:3000.
if (typeof window !== "undefined") {
  window.fetch = (() => new Promise(() => {})) as typeof window.fetch;
}
