import path from "path"
import tailwindcss from "@tailwindcss/vite"
import react from "@vitejs/plugin-react"
import { defineConfig } from "vite"

// https://vite.dev/config/
export default defineConfig({
  base: "./",
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      "/api": {
        target: process.env.VITE_OPS_PROXY_TARGET || "http://127.0.0.1:18080",
        changeOrigin: true,
      },
      "/healthz": {
        target: process.env.VITE_OPS_PROXY_TARGET || "http://127.0.0.1:18080",
        changeOrigin: true,
      },
    },
  },
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src"),
    },
  },
})
