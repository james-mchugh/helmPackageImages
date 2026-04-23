package manifest

import (
	"fmt"
	"os"
	"slices"

	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/yaml"
)

// Options controls how a manifest is located and resolved.
type Options struct {
	// ManifestPath is an explicit path to the airgap.yaml. Takes precedence over Chart.
	ManifestPath string

	Chart *chart.Chart

	// Profile is the name of the profile to activate. Error if it doesn't exist.
	Profile string

	// CLI overrides — applied last, after profile merge. Nil means "not set by CLI".
	OverridePlatform     string
	OverrideScrapeValues *bool
	OverrideIncludeDeps  *bool
	OverrideStrict       *bool
}

// Load locates, parses, and resolves the manifest according to opt.
// If no airgap.yaml is found, built-in defaults are returned with no error.
func Load(opt Options) (*Manifest, error) {

	raw := &rawManifest{}
	var err error
	if opt.ManifestPath != "" {
		raw, err = readFromPath(opt.ManifestPath)
	} else if index := slices.IndexFunc(
		opt.Chart.Files,
		func(file *chart.File) bool { return file.Name == "airgap.yaml" },
	); index != -1 {
		raw, err = readFromChartFile(opt.Chart.Files[index])
	}

	if err != nil {
		return nil, fmt.Errorf("loading manifest: %w", err)
	}

	// Apply profile if requested.
	if opt.Profile != "" {
		p, ok := raw.Profiles[opt.Profile]
		if !ok {
			return nil, fmt.Errorf("profile %q not found in manifest", opt.Profile)
		}
		mergeProfile(raw, p)
	}

	// Build resolved manifest.
	m := &Manifest{
		CRDs:           raw.CRDs,
		ConfigMapRules: raw.ConfigMapRules,
		Values:         raw.Values,
		Settings:       raw.Settings,
	}
	if m.Values == nil {
		m.Values = map[string]interface{}{}
	}

	// Apply CLI overrides last.
	if opt.OverridePlatform != "" {
		m.Settings.Platform = opt.OverridePlatform
	}
	if opt.OverrideScrapeValues != nil {
		m.Settings.ScrapeValues = *opt.OverrideScrapeValues
	}
	if opt.OverrideStrict != nil {
		m.Settings.StrictImageValidation = *opt.OverrideStrict
	}

	return m, nil
}

// readFromPath parses the airgap.yaml file (if any) into a rawManifest.
func readFromPath(path string) (*rawManifest, error) {

	if path == "" {
		return &rawManifest{}, nil
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &rawManifest{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading manifest: %w", err)
	}

	var raw rawManifest
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &raw, nil
}

// readFromPath parses the airgap.yaml file (if any) into a rawManifest.
func readFromChartFile(file *chart.File) (*rawManifest, error) {

	var raw rawManifest
	if err := yaml.Unmarshal(file.Data, &raw); err != nil {
		return nil, fmt.Errorf("parsing manifest: %w", err)
	}
	return &raw, nil
}

// mergeProfile deep-merges p over the base rawManifest.
func mergeProfile(base *rawManifest, p rawProfile) {
	// CRDs: replace when profile explicitly sets the field (even to empty).
	if p.CRDs != nil {
		base.CRDs = *p.CRDs
	}

	// ConfigMapRules: replace when profile explicitly sets the field (even to empty).
	if p.ConfigMapRules != nil {
		base.ConfigMapRules = *p.ConfigMapRules
	}

	// Values: deep-merge.
	if p.Values != nil {
		if base.Values == nil {
			base.Values = map[string]interface{}{}
		}
		deepMerge(base.Values, p.Values)
	}

	// Settings: merge individual fields when non-nil.
	if p.Settings != nil {
		if p.Settings.Platform != nil {
			base.Settings.Platform = *p.Settings.Platform
		}
		if p.Settings.ScrapeValues != nil {
			base.Settings.ScrapeValues = *p.Settings.ScrapeValues
		}
		if p.Settings.EnvVarPatterns != nil {
			base.Settings.EnvVarPatterns = *p.Settings.EnvVarPatterns
		}
		if p.Settings.StrictImageValidation != nil {
			base.Settings.StrictImageValidation = *p.Settings.StrictImageValidation
		}
	}
}

// deepMerge merges src into dst recursively. Scalar values in src overwrite dst.
// Map values are merged recursively. Other types (slices, etc.) are replaced.
func deepMerge(dst, src map[string]interface{}) {
	for k, sv := range src {
		dv, exists := dst[k]
		if !exists {
			dst[k] = sv
			continue
		}
		dsm, dstIsMap := dv.(map[string]interface{})
		ssm, srcIsMap := sv.(map[string]interface{})
		if dstIsMap && srcIsMap {
			deepMerge(dsm, ssm)
		} else {
			dst[k] = sv
		}
	}
}
