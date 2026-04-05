package extractor

import (
	"sort"

	"helmPackageImages/pkg/manifest"
)

// Extract discovers all container image references from rendered YAML documents
// using all configured sources, deduplicates them, and returns a sorted slice.
func Extract(docs []string, m *manifest.Manifest) ([]string, error) {
	seen := map[string]struct{}{}

	// 1. Built-in workload types.
	builtin, err := ExtractBuiltin(docs)
	if err != nil {
		return nil, err
	}
	for _, img := range builtin {
		seen[img] = struct{}{}
	}

	// 2. Custom resources (only when CRD entries are configured).
	if len(m.CRDs) > 0 {
		custom, err := ExtractCustom(docs, m.CRDs)
		if err != nil {
			return nil, err
		}
		for _, img := range custom {
			seen[img] = struct{}{}
		}
	}

	// 3. Naive values.yaml scan (opt-in).
	for _, img := range ScrapeValues(m.Values, m.Settings.ScrapeValues) {
		seen[img] = struct{}{}
	}

	result := setToSlice(seen)
	sort.Strings(result)
	return result, nil
}
