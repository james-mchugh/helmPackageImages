package extractor

import (
	"strings"

	"helmPackageImages/pkg/manifest"
	"sigs.k8s.io/yaml"
)

// ExtractCustom scans rendered YAML documents for images in custom resources
// described by the given CRD entries. Each entry specifies a kind+apiVersion
// to match and JSONPath-style dot-notation paths to image fields.
func ExtractCustom(docs []string, crds []manifest.CRDEntry) ([]string, error) {
	if len(crds) == 0 {
		return nil, nil
	}

	// Build a lookup keyed by "kind/apiVersion" for O(1) matching.
	type crdKey struct{ kind, apiVersion string }
	index := make(map[crdKey]*manifest.CRDEntry, len(crds))
	for i := range crds {
		k := crdKey{crds[i].Kind, crds[i].APIVersion}
		index[k] = &crds[i]
	}

	seen := map[string]struct{}{}
	for _, doc := range docs {
		for _, part := range splitYAMLDocs(doc) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			// Parse into a generic map to preserve all fields.
			var raw map[string]interface{}
			if err := yaml.Unmarshal([]byte(part), &raw); err != nil {
				continue
			}
			kind, _ := raw["kind"].(string)
			apiVersion, _ := raw["apiVersion"].(string)
			entry, ok := index[crdKey{kind, apiVersion}]
			if !ok {
				continue
			}
			for _, imgPath := range entry.ImagePaths {
				if img := extractPath(raw, imgPath); img != "" {
					seen[img] = struct{}{}
				}
			}
		}
	}
	return setToSlice(seen), nil
}

// extractPath resolves a dot-notation path (e.g. ".spec.image") against a
// generic map. Returns empty string if any segment is missing or not a string.
func extractPath(obj map[string]interface{}, path string) string {
	// Strip leading dot.
	path = strings.TrimPrefix(path, ".")
	segments := strings.Split(path, ".")

	var current interface{} = obj
	for _, seg := range segments {
		if seg == "" {
			continue
		}
		m, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = m[seg]
		if !ok {
			return ""
		}
	}
	s, _ := current.(string)
	return s
}
