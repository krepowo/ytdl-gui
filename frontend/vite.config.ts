import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath } from 'node:url'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      // In tests, the Wails runtime is replaced with a vi.fn() double (there is
      // no webview in jsdom). In a real build the plugin resolves the real one.
      '@wailsjs': fileURLToPath(new URL('./wailsjs/runtime', import.meta.url)),
    },
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    css: true,
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    alias: {
      '../wailsjs/runtime/runtime': fileURLToPath(
        new URL('./src/test/wailsRuntimeMock.ts', import.meta.url),
      ),
      '../../wailsjs/runtime/runtime': fileURLToPath(
        new URL('./src/test/wailsRuntimeMock.ts', import.meta.url),
      ),
    },
  },
})
