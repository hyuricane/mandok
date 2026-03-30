import * as monaco from 'monaco-editor';
import { configureMonacoYaml } from 'monaco-yaml';
import yaml from 'js-yaml';
import composeSpec from './schemas/compose_spec.json';
import serviceSpec from './schemas/service_spec.json';
import traefikSticky from './schemas/traefik-sticky.json';

// Define the supported schemas
/**
 * @typedef {Object} SchemaConfig
 * @property {string} name
 * @property {Object} content
 * @property {string} $id
 */
/**
 * @type {Record<string, SchemaConfig>}
 */
const SCHEMAS = {
  'docker-compose': {
    name: 'Compose',
    $id: composeSpec.$id,
    content: composeSpec,
  },
  'service': {
    name: 'Service',
    $id: serviceSpec.$id,
    content: serviceSpec,
  },
  'traefik-sticky': {
    name: 'Traefik Sticky',
    $id: traefikSticky.$id,
    content: traefikSticky,
  },
};

// Configure Monaco Workers
window.MonacoEnvironment = {
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

// Initial YAML config
const monacoYaml = configureMonacoYaml(monaco, {
  enableSchemaRequest: false,
  hover: true,
  completion: true,
  validate: true,
  format: true,
  schemas: [],
});

// Configure YAML and JSON schemas with fetched content
async function setupSchemas(schemaKey) {
  console.log('[DEBUG] setupSchemas', schemaKey, SCHEMAS[schemaKey])
  const schemaConfig = SCHEMAS[schemaKey];
  if (!schemaConfig) {
    console.warn(`Schema key "${schemaKey}" not found in SCHEMAS`);
    return;
  }
  const schemaContent = schemaConfig.content;

  // Configure YAML schema
  monacoYaml.update({
    schemas: [
      {
        uri: schemaConfig.$id,
        fileMatch: ['*'],
        schema: schemaContent,
      },
    ],
  });

  // Configure JSON schema
  monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
    validate: true,
    allowComments: false,
    schemas: [
      {
        fileMatch: ['*'],
        schema: schemaContent,
      },
    ],
  });

  console.log(`Schema "${schemaConfig.name}" loaded successfully`)
}
/**
 * @type {monaco.editor.IStandaloneCodeEditor}
 */
let editor = null;
/**
 * @type {'json'|'yaml'}
 */
let currentLanguage = 'json';

/**
 * @type {keyof typeof SCHEMAS}
 */
let currentSchemaKey = 'docker-compose';
/**
 * 
 * @param {string} elementId
 * @param {'json'|'yaml'} language 
 * @param {keyof typeof SCHEMAS} schemaKey 
 * @param {string} value 
 */
window.initMonacoEditor = async function (elementId, language, schemaKey = 'docker-compose', value = "") {
  console.log('[DEBUG] initMonacoEditor', elementId, language, schemaKey, value)

  // If the 3rd argument is a value (e.g. from projects.templ which called it incorrectly), 
  // try to detect it. If schemaKey is NOT in SCHEMAS and contains '{' or 'services:', it's likely a value.
  if (typeof schemaKey === 'string' && !SCHEMAS[schemaKey] && (schemaKey.includes('{') || schemaKey.includes('services:'))) {
    value = schemaKey;
    schemaKey = 'docker-compose';
  }

  if (currentSchemaKey !== schemaKey) {
    await setupSchemas(schemaKey);
    currentSchemaKey = schemaKey;
  }
  if (!editor) {
    console.log('[DEBUG] create editor')
    const container = document.getElementById(elementId);
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
      if (window.onMonacoEditorChange) {
        window.onMonacoEditorChange(editor.getValue());
      }
    });
  }
  console.log('[DEBUG] getValue')
  value = value || editor.getValue();
  console.log('[DEBUG] language', language)

  const model = editor.getModel();
  if (language !== currentLanguage) {
    let objValue;
    try {
      if (language === 'yaml') { // from json to yaml
        console.log('[DEBUG] from json to yaml')
        objValue = JSON.parse(value);
        value = yaml.dump(objValue, {
          indent: 2,
          lineWidth: -1,
          sortKeys: false,
        });
        monaco.editor.setModelLanguage(model, 'yaml');
      } else { // from yaml to json
        console.log('[DEBUG] from yaml to json')
        objValue = yaml.load(value);
        value = JSON.stringify(objValue, null, 2);
        monaco.editor.setModelLanguage(model, 'json');
      }
    } catch (e) {
      console.error('Conversion failed, keeping current value', e);
    }
    currentLanguage = language;
  }
  console.log('[DEBUG] existing models', monaco.editor.getModels())
  console.log('[DEBUG] setValue', value)
  editor.setValue(value);
  console.log('[DEBUG] return editor')
  return editor;
}

window.getMonacoEditor = function () {
  return editor;
}
