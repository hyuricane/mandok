package layouts

import (
	"testing"
	"testing/fstest"
)

func TestAssetPathWithManifest(t *testing.T) {
	manifestJSON := `{
		"node_modules/monaco-editor/esm/vs/base/browser/ui/codicons/codicon/codicon.ttf": {
			"file": "codicon.DCmgc-ay.ttf",
			"src": "node_modules/monaco-editor/esm/vs/base/browser/ui/codicons/codicon/codicon.ttf"
		},
		"src/index.js": {
			"file": "bundle.BTBtY_9f.js",
			"name": "index",
			"src": "src/index.js",
			"isEntry": true,
			"css": ["bundle.BTBtY_9f.css"]
		}
	}`

	mockFS := fstest.MapFS{
		".vite/manifest.json": &fstest.MapFile{
			Data: []byte(manifestJSON),
		},
	}

	SetStaticFS(mockFS)

	tests := []struct {
		input    string
		expected string
	}{
		{"src/index.js", "/static/bundle.BTBtY_9f.js"},
		{"bundle.js", "/static/bundle.BTBtY_9f.js"},
		{"node_modules/monaco-editor/esm/vs/base/browser/ui/codicons/codicon/codicon.ttf", "/static/codicon.DCmgc-ay.ttf"},
		{"alpine-data.js", "/static/alpine-data.js"},
		{"/static/alpine-data.js", "/static/alpine-data.js"},
	}

	for _, tt := range tests {
		got := AssetPath(tt.input)
		if got != tt.expected {
			t.Errorf("AssetPath(%q) = %q, expected %q", tt.input, got, tt.expected)
		}
	}

	css := AssetCSS("src/index.js")
	if len(css) != 1 || css[0] != "/static/bundle.BTBtY_9f.css" {
		t.Errorf("AssetCSS(\"src/index.js\") = %v, expected [\"/static/bundle.BTBtY_9f.css\"]", css)
	}
}
