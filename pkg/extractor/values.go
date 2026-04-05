package extractor

import (
	"fmt"
	"strings"
)

// ScrapeValues naively walks a values map and collects image-like strings.
// When enabled is false, it returns nil immediately.
//
// Detection strategy:
//  1. If a map contains a "repository" key (and optionally "registry" and "tag"),
//     reconstruct the full image reference: [registry/]repository[:tag].
//  2. For scalar string values, flag those that look like image references
//     (contain a ":" or "/" but are not URLs).
func ScrapeValues(values map[string]interface{}, enabled bool) []string {
	if !enabled {
		return nil
	}
	seen := map[string]struct{}{}
	walkValues(values, seen)
	return setToSlice(seen)
}

func walkValues(v map[string]interface{}, seen map[string]struct{}) {
	// Check if this map looks like an image block with repository/tag keys.
	if img := imageFromMap(v); img != "" {
		seen[img] = struct{}{}
		return
	}
	for _, val := range v {
		switch tv := val.(type) {
		case string:
			if looksLikeImage(tv) {
				seen[tv] = struct{}{}
			}
		case map[string]interface{}:
			walkValues(tv, seen)
		}
	}
}

// imageFromMap attempts to reconstruct an image reference from a map containing
// "repository" (required), "registry" (optional), and "tag" (optional) keys.
func imageFromMap(m map[string]interface{}) string {
	repo, ok := m["repository"].(string)
	if !ok || repo == "" {
		return ""
	}
	registry, _ := m["registry"].(string)
	tag, _ := m["tag"].(string)

	var ref string
	if registry != "" {
		ref = fmt.Sprintf("%s/%s", registry, repo)
	} else {
		ref = repo
	}
	if tag != "" {
		ref = fmt.Sprintf("%s:%s", ref, tag)
	}
	return ref
}

// looksLikeImage returns true if s resembles a container image reference.
// Heuristic: contains ":" (for tag) or "/" (for registry/repo path),
// but is not an HTTP/HTTPS URL.
func looksLikeImage(s string) bool {
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
		return false
	}
	return strings.Contains(s, ":") || strings.Contains(s, "/")
}
