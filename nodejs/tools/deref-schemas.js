// import $RefParser from '@apidevtools/json-schema-ref-parser';
// import fs from 'fs';

const $RefParser = require('@apidevtools/json-schema-ref-parser');
const fs = require('fs');

const workDir = process.cwd();
const schemas = fs.readdirSync(`${workDir}/src/schemas`);

schemas.forEach((schema) => {
  if (!schema.endsWith('.json')) {
    return;
  }
  if (schema.endsWith('.deref.json')) {
    return;
  }

  const schemaPath = `${workDir}/src/schemas/${schema}`;
  const schemaContent = JSON.parse(fs.readFileSync(schemaPath, 'utf8'));

  $RefParser.dereference(schemaContent).then((dereferencedSchema) => {
    fs.writeFileSync(schemaPath.replace('.json', '.deref.json'), JSON.stringify(dereferencedSchema, null, 2));
  });
});