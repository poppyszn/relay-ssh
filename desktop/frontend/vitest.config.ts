import { defineConfig } from "vitest/config";
import { svelte } from "@sveltejs/vite-plugin-svelte";

// Separate from vite.config.ts: tests need Svelte's browser build (so components can be mounted)
// and have no use for the Wails bindings plugin.
export default defineConfig({
  plugins: [svelte()],
  resolve: { conditions: ["browser"] },
});
