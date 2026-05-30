import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  server: {
    port: 3000,
    host: true,
    proxy: {
      '/api': 'http://gateway:8080',
      '/health': 'http://gateway:8080',
    },
  },
})
