package layouts

import (
	"encoding/json"
	"io/fs"
	"log"
	"os"
	"strings"
	"sync"

	"inovasiriset.co.id/docker/manager/conf"
)

type ManifestEntry struct {
	File           string   `json:"file"`
	Name           string   `json:"name,omitempty"`
	Src            string   `json:"src,omitempty"`
	IsEntry        bool     `json:"isEntry,omitempty"`
	DynamicImports []string `json:"dynamicImports,omitempty"`
	Assets         []string `json:"assets,omitempty"`
	CSS            []string `json:"css,omitempty"`
}

type Manifest map[string]ManifestEntry

var (
	manifestLock sync.RWMutex
	manifestData Manifest
	staticFSList []fs.FS
)

// SetStaticFS sets the static file systems to search for manifest and assets.
func SetStaticFS(fses ...fs.FS) {
	manifestLock.Lock()
	defer manifestLock.Unlock()
	staticFSList = fses
	manifestData = nil
}

func loadManifest() Manifest {
	if !conf.IS_DEBUG {
		manifestLock.RLock()
		if manifestData != nil {
			defer manifestLock.RUnlock()
			return manifestData
		}
		manifestLock.RUnlock()
	}

	manifestLock.Lock()
	defer manifestLock.Unlock()

	if !conf.IS_DEBUG && manifestData != nil {
		return manifestData
	}

	manifest := make(Manifest)
	manifestPaths := []string{
		".vite/manifest.json",
		"static/.vite/manifest.json",
	}

	var data []byte
	var err error

	// Try reading from registered static FS list
	for _, fsys := range staticFSList {
		if fsys == nil {
			continue
		}
		for _, p := range manifestPaths {
			data, err = fs.ReadFile(fsys, p)
			if err == nil && len(data) > 0 {
				break
			}
		}
		if len(data) > 0 {
			break
		}
	}

	// Fallback to reading directly from OS file system
	if len(data) == 0 {
		for _, p := range []string{"static/.vite/manifest.json", ".vite/manifest.json"} {
			data, err = os.ReadFile(p)
			if err == nil && len(data) > 0 {
				break
			}
		}
	}

	if len(data) > 0 {
		if err := json.Unmarshal(data, &manifest); err != nil {
			log.Printf("[WARNING] failed to parse vite manifest: %v", err)
		}
	}

	manifestData = manifest
	return manifestData
}

// AssetPath returns the web path for a given asset (e.g. "src/index.js" or "bundle.js").
func AssetPath(name string) string {
	m := loadManifest()
	if len(m) > 0 {
		// 1. Exact key match
		if entry, ok := m[name]; ok && entry.File != "" {
			return "/static/" + entry.File
		}

		// 2. Lookup by relative path or entry name
		cleanName := strings.TrimPrefix(name, "src/")
		cleanName = strings.TrimPrefix(cleanName, "/")
		for k, entry := range m {
			if k == cleanName || strings.HasSuffix(k, name) || entry.Name == name {
				return "/static/" + entry.File
			}
			// If looking for bundle.js, match entry whose output file is bundle.<hash>.js
			if (name == "bundle.js" || name == "bundle" || name == "src/index.js") && strings.HasPrefix(entry.File, "bundle.") && strings.HasSuffix(entry.File, ".js") {
				return "/static/" + entry.File
			}
		}
	}

	// Fallback when manifest is missing or asset not found
	if strings.HasPrefix(name, "/static/") {
		return name
	}
	return "/static/" + strings.TrimPrefix(name, "/")
}

// AssetCSS returns the CSS file paths associated with a given entry point in the manifest.
func AssetCSS(name string) []string {
	m := loadManifest()
	if len(m) == 0 {
		return nil
	}

	var cssFiles []string
	if entry, ok := m[name]; ok {
		for _, css := range entry.CSS {
			cssFiles = append(cssFiles, "/static/"+css)
		}
		return cssFiles
	}

	// Fallback search across entries
	for k, entry := range m {
		if k == name || strings.HasSuffix(k, name) || (name == "src/index.js" && entry.IsEntry) {
			for _, css := range entry.CSS {
				cssFiles = append(cssFiles, "/static/"+css)
			}
			return cssFiles
		}
	}

	return nil
}
