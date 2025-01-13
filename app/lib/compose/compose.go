package compose

import (
	"os"
)

var PROJECT_DIRS = "projects"

func init() {
	if os.Getenv("PROJECT_DIRS") != "" {
		PROJECT_DIRS = os.Getenv("PROJECT_DIRS")
	}
	if os.Getenv("NETWORK") != "" {
		NETWORK = os.Getenv("NETWORK")
	}
}

func mergeMap(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMap(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}
