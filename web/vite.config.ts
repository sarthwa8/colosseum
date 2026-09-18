import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// The spectator UI builds into the Go binary: Vite emits to internal/web/dist,
// which server.go go:embeds. `npm run dev` proxies /api to a running
// `colosseum serve` so the replay UI can be developed live against real data.
export default defineConfig({
  plugins: [react(), tailwindcss()],
  build: {
    outDir: '../internal/web/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/api': 'http://localhost:8080',
    },
  },
})
