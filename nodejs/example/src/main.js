import * as monaco from 'monaco-editor';
import { configureMonacoYaml } from 'monaco-yaml';
import yaml from 'js-yaml';

import composeSpec from './schemas/compose_spec.json';

// Define the supported schemas
const SCHEMAS = {
  'docker-compose': {
    name: 'Docker Compose',
    url: 'https://raw.githubusercontent.com/compose-spec/compose-spec/main/schema/compose-spec.json',
    schema: composeSpec,
    defaultContent: `services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: example`
  },
  'traefik-static': {
    name: 'Traefik Static',
    url: 'https://json.schemastore.org/traefik-v2.json',
    defaultContent: `api:
  dashboard: true
  insecure: true

entryPoints:
  web:
    address: ":80"

providers:
  docker:
    endpoint: "unix:///var/run/docker.sock"
    exposedByDefault: false`
  },
  'traefik-dynamic': {
    name: 'Traefik Dynamic',
    url: 'https://json.schemastore.org/traefik-v2-file-provider.json',
    defaultContent: `http:
  routers:
    my-router:
      rule: "Host(\`example.com\`)"
      service: my-service
      entryPoints:
        - web

  services:
    my-service:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:8080"`
  }
};

let currentSchemaKey = 'docker-compose';

// Configure Monaco Workers
window.MonacoEnvironment = {
  getWorkerUrl(workerId, label) {
    switch (label) {
      case 'yaml':
        return './yaml.worker.js';
      case 'json':
        return './json.worker.js';
      default:
        return './editor.worker.js';
    }
  },
};

// Initial YAML config
const monacoYaml = configureMonacoYaml(monaco, {
  enableSchemaRequest: true,
  hover: true,
  completion: true,
  validate: true,
  format: true,
  schemas: [],
});

// Configure YAML and JSON schemas with fetched content
async function setupSchemas(schemaKey) {
  const schemaConfig = SCHEMAS[schemaKey];
  const schemaUrl = schemaConfig.url;

  try {
    if (!schemaConfig.schema) {
      const response = await fetch(schemaUrl);
      schemaConfig.schema = await response.json();
    }

    // Configure YAML schema
    monacoYaml.update({
      schemas: [
        {
          uri: schemaUrl,
          fileMatch: ['*'],
          schema: schemaConfig.schema,
        },
      ],
    });

    // Configure JSON schema
    monaco.languages.json.jsonDefaults.setDiagnosticsOptions({
      validate: true,
      allowComments: false,
      schemas: [
        {
          uri: schemaUrl,
          fileMatch: ['*'],
          schema: schemaConfig.schema,
        },
      ],
    });

    console.log(`Schema "${schemaConfig.name}" loaded successfully`);
  } catch (e) {
    console.error(`Failed to load schema "${schemaConfig.name}":`, e);
  } finally {
    console.log('[DEBUG] existing models', monaco.editor.getModels());
  }
}

setupSchemas(currentSchemaKey);

const schemaSelect = document.getElementById('schema-select');
const formatToggle = document.getElementById('format-toggle');
const fileTypeDisplay = document.getElementById('file-type');
const saveBtn = document.getElementById('save-btn');
const editorContainer = document.getElementById('editor-container');

const searchParams = new URLSearchParams(document.location.search);
let currentLanguage = searchParams.get('lang') || 'json';

// Create the editor
const editor = monaco.editor.create(editorContainer, {
  language: currentLanguage,
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
});
if (currentLanguage === 'yaml') {
  editor.setValue(SCHEMAS[currentSchemaKey].defaultContent);
} else {
  editor.setValue(JSON.stringify(yaml.load(SCHEMAS[currentSchemaKey].defaultContent), null, 2));
}

// Handle schema change
schemaSelect.addEventListener('change', async (e) => {
  const newSchemaKey = e.target.value;
  if (newSchemaKey === currentSchemaKey) return;

  const oldContent = editor.getValue();
  const oldSchema = SCHEMAS[currentSchemaKey];
  const newSchema = SCHEMAS[newSchemaKey];

  // If the content is exactly the default of the old schema, update it to the default of the new schema
  let shouldUpdateContent = false;
  try {
    const oldDefault = currentLanguage === 'yaml' ? oldSchema.defaultContent : JSON.stringify(yaml.load(oldSchema.defaultContent), null, 2);
    if (oldContent.trim() === oldDefault.trim()) {
      shouldUpdateContent = true;
    }
  } catch (err) {
    // Ignore conversion errors
  }

  currentSchemaKey = newSchemaKey;
  await setupSchemas(currentSchemaKey);

  if (shouldUpdateContent) {
    const newDefault = currentLanguage === 'yaml' ? newSchema.defaultContent : JSON.stringify(yaml.load(newSchema.defaultContent), null, 2);
    editor.setValue(newDefault);
  }
});

// Switch between YAML and JSON
formatToggle.addEventListener('click', () => {
  const currentValue = editor.getValue();

  try {
    if (currentLanguage === 'yaml') {
      // Convert YAML to JSON
      const obj = yaml.load(currentValue);
      const jsonValue = JSON.stringify(obj, null, 2);

      monaco.editor.setModelLanguage(editor.getModel(), 'json');
      editor.setValue(jsonValue);

      currentLanguage = 'json';
      formatToggle.textContent = 'Switch to YAML';
      fileTypeDisplay.textContent = 'JSON';
    } else {
      // Convert JSON to YAML
      const obj = JSON.parse(currentValue);
      const yamlValue = yaml.dump(obj);

      monaco.editor.setModelLanguage(editor.getModel(), 'yaml');
      editor.setValue(yamlValue);

      currentLanguage = 'yaml';
      formatToggle.textContent = 'Switch to JSON';
      fileTypeDisplay.textContent = 'YAML';
    }
  } catch (e) {
    console.error('Conversion failed:', e);
    alert('Conversion failed: ' + e.message);
  }
});

// Save button handling (just a simulation)
saveBtn.addEventListener('click', () => {
  const value = editor.getValue();
  console.log('Saving content:', value);
  saveBtn.textContent = 'Saved!';
  setTimeout(() => {
    saveBtn.textContent = 'Save';
  }, 2000);
});

// Status bar validation updates
editor.onDidChangeModelDecorations(() => {
  const model = editor.getModel();
  if (!model) return;

  const owner = currentLanguage === 'yaml' ? 'yaml' : 'json';
  const markers = monaco.editor.getModelMarkers({ owner, resource: model.uri });

  const statusEl = document.getElementById('validation-status');
  if (markers.length > 0) {
    statusEl.textContent = `${markers.length} Issue(s) found`;
    statusEl.style.color = '#f85149';
  } else {
    statusEl.textContent = 'No issues';
    statusEl.style.color = '#3fb950';
  }
});
