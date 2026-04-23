package extractor

import (
	"sort"

	"helmPackageImages/pkg/manifest"

	"k8s.io/apimachinery/pkg/runtime"
)

// Extract discovers all container image references from rendered YAML documents
// using all configured sources, deduplicates them, and returns a sorted slice.
func Extract(docs []runtime.Object, m *manifest.Manifest) ([]string, error) {
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

	// 4. Env var name-pattern extraction (opt-in).
	if len(m.Settings.EnvVarPatterns) > 0 {
		envImgs, err := ExtractEnvVars(docs, m.Settings.EnvVarPatterns)
		if err != nil {
			return nil, err
		}
		for _, img := range envImgs {
			seen[img] = struct{}{}
		}
	}

	// 5. ConfigMap rules extraction (opt-in).
	if len(m.ConfigMapRules) > 0 {
		cmImgs, err := ExtractConfigMaps(docs, m.ConfigMapRules)
		if err != nil {
			return nil, err
		}
		for _, img := range cmImgs {
			seen[img] = struct{}{}
		}
	}

	result := setToSlice(seen)
	sort.Strings(result)
	return result, nil
}
