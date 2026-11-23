import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig, type Plugin } from "vite";

const excalidrawCdn = /`https:\/\/esm\.sh\/\$\{[^`]*`[^`]*`[^`]*\}\/dist\/(?:prod|dev)\/`/;

function offlineExcalidraw(): Plugin {
  return {
    name: "offline-excalidraw",
    transform(code, id) {
      if (!id.includes("@excalidraw/excalidraw") || !code.includes("ASSETS_FALLBACK_URL")) return null;
      if (!excalidrawCdn.test(code)) this.error("Excalidraw font fallback URL not found");
      return { code: code.replace(excalidrawCdn, 'new URL("/", window.location.href).href'), map: null };
    },
  };
}

export default defineConfig({
  plugins: [react({ compiler: true }), tailwindcss(), offlineExcalidraw()],
  optimizeDeps: {
    rolldownOptions: { plugins: [offlineExcalidraw()] },
  },
  resolve: {
    alias: { "@": "/src" },
  },
  server: {
    port: 5173,
    strictPort: true,
    watch: { ignored: ["**/.mygo/**", "**/build/**"] },
  },
});
