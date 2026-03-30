import { defineConfig } from 'vite';

export default defineConfig({
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
  },
  optimizeDeps: {
    include: [
      'monaco-editor/esm/vs/editor/editor.api',
      'monaco-editor/esm/vs/language/json/json.worker',
      'monaco-editor/esm/vs/editor/editor.worker',
      'monaco-yaml/yaml.worker',
    ]
  }
});
