package manifest

import "k8s.io/apimachinery/pkg/runtime"

// CRDEntry describes a custom resource kind whose image fields should be extracted.
type CRDEntry struct {
	runtime.TypeMeta
	ImagePaths []string `yaml:"imagePaths"`
}

// Settings holds plugin-level behavioral configuration.
type Settings struct {
	// Platform is a comma-separated list of os/arch targets (e.g. "linux/amd64,linux/arm64").
	// Empty means the current system platform.
	Platform string `yaml:"platform"`

	// ScrapeValues enables naive heuristic scanning of values.yaml for image-like strings.
	ScrapeValues bool `yaml:"scrapeValues"`
}

// Profile defines overrides applied on top of the base manifest configuration.
type Profile struct {
	// CRDs replaces the base CRD list when non-nil (even if empty).
	CRDs *[]CRDEntry `yaml:"crds"`

	// Values is deep-merged over the base values when non-nil.
	Values map[string]interface{} `yaml:"values"`

	// Settings fields are merged individually over base settings when non-zero.
	Settings *Settings `yaml:"settings"`
}

// Manifest is the parsed and fully-resolved airgap.yaml.
type Manifest struct {
	CRDs     []CRDEntry             `yaml:"crds"`
	Values   map[string]interface{} `yaml:"values"`
	Settings Settings               `yaml:"settings"`
}

// rawManifest mirrors Manifest but uses pointers so we can distinguish
// "not set" from "set to zero value" when merging profiles.
type rawManifest struct {
	CRDs     []CRDEntry             `yaml:"crds"`
	Values   map[string]interface{} `yaml:"values"`
	Settings Settings               `yaml:"settings"`
	Profiles map[string]rawProfile  `yaml:"profiles"`
}

type rawProfile struct {
	CRDs     *[]CRDEntry            `yaml:"crds"`
	Values   map[string]interface{} `yaml:"values"`
	Settings *rawSettings           `yaml:"settings"`
}

type rawSettings struct {
	Platform                 *string `yaml:"platform"`
	IncludeChartDependencies *bool   `yaml:"includeChartDependencies"`
	ScrapeValues             *bool   `yaml:"scrapeValues"`
}
