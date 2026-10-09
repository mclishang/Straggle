import { defineConfig } from 'vite';

// Desktop application: Wails serves the build from the embedded filesystem, so
// assets are requested relatively (base './') and no source map is emitted.
export default defineConfig({
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    target: 'chrome110',
    sourcemap: false,
    assetsInlineLimit: 0,
  },
  server: { port: 5173, strictPort: true },
  preview: { port: 4173, strictPort: true },
});
