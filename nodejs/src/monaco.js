import * as monaco from 'monaco-editor'
// import { configureMonacoYaml } from 'monaco-yaml'

// window.MonacoEnvironment = {
//   getWorker(moduleId, label) {
//     switch (label) {
//       case 'editorWorkerService':
//         return new Worker(new URL('monaco-editor/esm/vs/editor/editor.worker', import.meta.url))
//       case 'json':
//         return new Worker(
//           new URL('monaco-editor/esm/vs/language/json/json.worker', import.meta.url)
//         )
//       case 'yaml':
//         return new Worker(new URL('monaco-yaml/yaml.worker', import.meta.url))
//       default:
//         throw new Error(`Unknown label ${label}`)
//     }
//   }
// }

// configureMonacoYaml(monaco, {
//   enableSchemaRequest: false
// })

// const prettierc = monaco.editor.createModel(
//   'singleQuote: true\nproseWrap: always\nsemi: yes\n',
//   undefined,
//   monaco.Uri.parse('file:///.prettierrc.yaml')
// )

// monaco.editor.createModel(
//   'name: John Doe\nage: 42\noccupation: Pirate\n',
//   undefined,
//   monaco.Uri.parse('file:///person.yaml')
// )

self.MonacoEnvironment = {
	getWorkerUrl: function (moduleId, label) {
		if (label === 'json') {
			return '/static/json.worker.js';
		}
		return '/static/editor.worker.js';
	}
};

let _editor = null

self.getMonacoEditor = function () {
  return _editor
}


self.initMonacoEditor = function (elementId, language = 'json', value = "") {
  if (!window?.document) {
    return
  }
  if (_editor) {
    _editor.dispose()
  }
  _editor = monaco.editor.create(document.getElementById(elementId), {
    // model: prettierc,
    value: value,
    language: language,
    theme: 'vs-dark',
    lineNumbers: 'on',
    automaticLayout: true,
    scrollBeyondLastLine: false,
    
    minimap: { enabled: false },
  })
  _editor.onEndUpdate(() => {
    // console.log(_editor.getValue())
  })

  return _editor
}