import { defineConfig } from 'vite'
import path from 'path'
import glob from 'glob'

export default defineConfig({
  cacheDir: ".vite_cache",
  build: {
    target: 'esnext',
    minify: 'esbuild',
    lib: {
      entry: path.resolve(__dirname, 'static/scripts/main.js'),
      name: 'highlight',
      fileName: 'lib',
      formats: ['iife'],
    },
    outDir: 'static/dist',
    cssCodeSplit: false,
    reportCompressedSize: false,
    emptyOutDir: true, // Clean output directory only once
  },
  optimizeDeps: {
    include: ['monaco-editor'],
    esbuildOptions: {
      target: 'esnext' // Consistent with build target
    }
  },
  esbuild: {
    legalComments: 'none', // Remove license comments
    target: 'esnext'
  }
})