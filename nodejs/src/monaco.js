import yaml from 'js-yaml';
import composeSpec from './schemas/compose_spec.deref.json';
import traefikSticky from './schemas/traefik-sticky.deref.json';

const composeSpecBlob = new Blob([
  JSON.stringify(composeSpec, null, 2)
], { type: 'application/json' })
const composeSpecUri = URL.createObjectURL(composeSpecBlob);

const serviceSpec = {
  $schema: composeSpec.$schema,
  $id: composeSpecUri + '#/definitions/service',
  title: 'Service',
  definitions: composeSpec.definitions,
  ...composeSpec.definitions.service,
}

const serviceSpecBlob = new Blob([
  JSON.stringify(serviceSpec, null, 2)
], { type: 'application/json' })
const serviceSpecUri = URL.createObjectURL(serviceSpecBlob);

const traefikStickyBlob = new Blob([
  JSON.stringify(traefikSticky, null, 2)
], { type: 'application/json' })
const traefikStickyUri = URL.createObjectURL(traefikStickyBlob);

// Define the supported schemas
/**
 * @typedef {Object} SchemaConfig
 * @property {string} name
 * @property {string} uri
 */
/**
 * @type {Record<string, SchemaConfig>}
 */
const SCHEMAS = {
  'docker-compose': {
    name: 'Compose',
    uri: composeSpecUri,
  },
  'service': {
    name: 'Service',
    uri: serviceSpecUri,
  },
  'traefik-sticky': {
    name: 'Traefik Sticky',
    uri: traefikStickyUri,
  },
};

let monacoModule = null;
let monacoYamlModule = null;
let configured = false

/**
 * 
 * @returns {Promise<{monaco: typeof import('monaco-editor'), monacoYaml: typeof import('monaco-yaml').configureMonacoYaml}>}
 */
async function loadMonaco() {
  if (monacoModule) return { monaco: monacoModule, monacoYaml: monacoYamlModule };
  // Configure Monaco Workers
  globalThis.MonacoEnvironment = {
    getWorkerUrl(workerId, label) {
      switch (label) {
        case 'yaml':
          return '/static/yaml.worker.js';
        case 'json':
          return '/static/json.worker.js';
        default:
          return '/static/editor.worker.js';
      }
    },
  };
  const [{ default: monaco }, { configureMonacoYaml }] = await Promise.all([import('monaco-editor'), import('monaco-yaml')])
  monacoModule = monaco;
  monacoYamlModule = configureMonacoYaml(monaco, {
    enableSchemaRequest: true,
    hover: true,
    completion: true,
    validate: true,
    format: true,
    schemas: [],
  });
  configured = true;

  // expose to global
  globalThis.monaco = monacoModule
  globalThis.monacoYaml = monacoYamlModule
  return { monaco: monacoModule, monacoYaml: monacoYamlModule };
}


// Shared editor state
/** @type {monaco.editor.IStandaloneCodeEditor | null} */
let editor = null;
/** @type {'json'|'yaml'} */
let currentLanguage = 'json';
/** @type {keyof typeof SCHEMAS | null} */
let currentSchemaKey = null;
/** @type {HTMLElement | null} */
let currentContainer = null;


// Configure YAML and JSON schemas with fetched content
async function setupSchemas(schemaKey) {
  const schemaConfig = SCHEMAS[schemaKey];
  if (!schemaConfig) {
    console.warn(`Schema key "${schemaKey}" not found in SCHEMAS`);
    return;
  }

  const { monaco, monacoYaml } = await loadMonaco();

  // Configure YAML schema
  await monacoYaml.update({
    schemas: [
      {
        uri: schemaConfig.uri,
        fileMatch: ['*']
      },
    ],
  });

  // Configure JSON schema
  monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
    enableSchemaRequest: false,
    validate: true,
    allowComments: false,
    schemas: [
      {
        uri: schemaConfig.uri,
        fileMatch: ['*'],
        schema: await fetch(schemaConfig.uri).then(res => res.json()),
      },
    ]
  });
}

/**
 * 
 * @param {string} elementId
 * @param {'json'|'yaml'} language 
 * @param {keyof typeof SCHEMAS} schemaKey 
 * @param {string} value 
 */
globalThis.initMonacoEditor = async function (elementId, language, schemaKey = 'docker-compose', value = "") {
  const { monaco } = await loadMonaco();
  if (currentSchemaKey !== schemaKey) {
    await setupSchemas(schemaKey);
    currentSchemaKey = schemaKey;
  }

  const container = document.getElementById(elementId);
  if (currentContainer !== container) {
    editor?.dispose();
    editor = null;
    currentContainer = container;
  }
  if (!editor) {
    if (!container) {
      console.error(`Editor container #${elementId} not found`);
      return null;
    }
    editor = monaco.editor.create(container, {
      language: language,
      theme: 'vs-dark',
      automaticLayout: true,
      fontSize: 14,
      fontFamily: 'JetBrains Mono, monospace',
      minimap: { enabled: true },
      scrollBeyondLastLine: false,
      lineNumbers: 'on',
      roundedSelection: false,
      cursorStyle: 'line',
      glyphMargin: true,
      quickSuggestions: true,
      quickSuggestionsDelay: 400,
    });

    // Provide a way for external code to get notified of changes (since projects.templ expects it)
    editor.onDidChangeModelContent(() => {
      if (globalThis.onMonacoEditorChange) {
        globalThis.onMonacoEditorChange(editor.getValue());
      }
    });
  }
  value = value || editor.getValue();
  const model = editor.getModel();
  if (language !== currentLanguage) {
    let objValue;
    try {
      if (language === 'yaml') { // from json to yaml
        objValue = JSON.parse(value);
        value = yaml.dump(objValue, {
          indent: 2,
          lineWidth: -1,
          sortKeys: false,
        });
        monaco.editor.setModelLanguage(model, 'yaml');
      } else { // from yaml to json
        objValue = yaml.load(value);
        value = JSON.stringify(objValue, null, 2);
        monaco.editor.setModelLanguage(model, 'json');
      }
    } catch (e) {
      console.error('Conversion failed, keeping current value', e);
    }
    currentLanguage = language;
  }
  editor.setValue(value);
  return editor;
}

globalThis.getMonacoEditor = function () {
  return editor;
}

globalThis.disposeMonacoEditor = function () {
  editor?.dispose();
  editor = null;
  currentContainer = null;
}

globalThis.listenToMarkers = function (listener) {
  globalThis.monaco.editor.onDidChangeMarkers((e) => {
    listener(monacoErrorMarkers())
  })
}

// error markers
globalThis.monacoErrorMarkers = function () {
  if (!editor) {
    return [];
  }

  const model = editor.getModel();
  if (!model) {
    return [];
  }
  const owner = model.getLanguageId();
  const markers = globalThis.monaco.editor.getModelMarkers({ owner: owner, resource: model.uri });
  return markers;
}

