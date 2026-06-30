import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  base: '/',
  server: {
    host: '0.0.0.0', // Escucha en todas las interfaces de red
    port: 5173,      // Puerto por defecto
    strictPort: true, // Si el puerto está ocupado, falla en lugar de usar el siguiente
  },
  build: {
    outDir: 'dist',
    assetsDir: 'assets',
  },
})
