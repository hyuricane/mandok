import { defineConfig } from 'vite'
import monacoEditorPluginModule from 'vite-plugin-monaco-editor'
import path from 'path'

const monacoEditorPlugin = monacoEditorPluginModule.default || monacoEditorPluginModule;

export default defineConfig({
  base: '/static/',
  build: {
    outDir: 'static',
    emptyOutDir: true,
    rolldownOptions: {
      input: path.resolve(__dirname, 'src/index.js'),
      output: {
        format: 'iife',
        name: 'MandokBundle',
        entryFileNames: 'bundle.js',
        chunkFileNames: 'chunk/[name].[hash].js',
        cleanDir: true,
        assetFileNames: '[name].[ext]',
      }
    },
    minify: true,
  },
  plugins: [
  ],
  optimizeDeps: {
    include: [
      'monaco-editor/esm/vs/editor/editor.api',
      'monaco-editor/esm/vs/language/json/json.worker',
      'monaco-editor/esm/vs/editor/editor.worker',
      'monaco-yaml/yaml.worker'
    ]
  }
})
