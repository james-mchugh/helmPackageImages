package manifest

import "k8s.io/apimachinery/pkg/runtime"

// CRDEntry describes a custom resource kind whose image fields should be extracted.
type CRDEntry struct {
	runtime.TypeMeta
	ImagePaths []string `yaml:"imagePaths"`
}

// ConfigMapRule describes how to extract image references from ConfigMap data values.
type ConfigMapRule struct {
	// NamePattern is a regex matched against the ConfigMap metadata.name. Empty matches all.
	NamePattern string `yaml:"namePattern"`
	// KeyPattern is a regex matched against each data key. Empty matches all.
	KeyPattern string `yaml:"keyPattern"`
	// JSONPath parses the value as YAML/JSON and extracts via this expression.
	// Takes precedence over Regex when both are set.
	JSONPath string `yaml:"jsonPath"`
	// Regex scans the raw string value for all matches. When empty and JSONPath is also
	// empty, the looksLikeImage heuristic is applied per whitespace-separated token.
	Regex string `yaml:"regex"`
}

// Settings holds plugin-level behavioral configuration.
type Settings struct {
	// Platform is a comma-separated list of os/arch targets (e.g. "linux/amd64,linux/arm64").
	// Empty means the current system platform.
	Platform string `yaml:"platform"`

	// ScrapeValues enables naive heuristic scanning of values.yaml for image-like strings.
	ScrapeValues bool `yaml:"scrapeValues"`

	// EnvVarPatterns is a list of regexes matched against container env var names.
	// Env vars whose names match have their values treated as image references. Disabled if empty.
	EnvVarPatterns []string `yaml:"envVarPatterns"`

	// StrictImageValidation causes the command to fail if any extracted string is not a
	// valid image reference. Without this, invalid refs are always warned about and filtered out.
	StrictImageValidation bool `yaml:"strictImageValidation"`
}

// Profile defines overrides applied on top of the base manifest configuration.
type Profile struct {
	// CRDs replaces the base CRD list when non-nil (even if empty).
	CRDs *[]CRDEntry `yaml:"crds"`

	// ConfigMapRules replaces the base rules when non-nil (even if empty).
	ConfigMapRules *[]ConfigMapRule `yaml:"configMapRules"`

	// Values is deep-merged over the base values when non-nil.
	Values map[string]interface{} `yaml:"values"`

	// Settings fields are merged individually over base settings when non-zero.
	Settings *Settings `yaml:"settings"`
}

// Manifest is the parsed and fully-resolved airgap.yaml.
type Manifest struct {
	CRDs           []CRDEntry             `yaml:"crds"`
	ConfigMapRules []ConfigMapRule        `yaml:"configMapRules"`
	Values         map[string]interface{} `yaml:"values"`
	Settings       Settings               `yaml:"settings"`
}

// rawManifest mirrors Manifest but uses pointers so we can distinguish
// "not set" from "set to zero value" when merging profiles.
type rawManifest struct {
	CRDs           []CRDEntry             `yaml:"crds"`
	ConfigMapRules []ConfigMapRule        `yaml:"configMapRules"`
	Values         map[string]interface{} `yaml:"values"`
	Settings       Settings               `yaml:"settings"`
	Profiles       map[string]rawProfile  `yaml:"profiles"`
}

type rawProfile struct {
	CRDs           *[]CRDEntry            `yaml:"crds"`
	ConfigMapRules *[]ConfigMapRule       `yaml:"configMapRules"`
	Values         map[string]interface{} `yaml:"values"`
	Settings       *rawSettings           `yaml:"settings"`
}

type rawSettings struct {
	Platform                 *string   `yaml:"platform"`
	IncludeChartDependencies *bool     `yaml:"includeChartDependencies"`
	ScrapeValues             *bool     `yaml:"scrapeValues"`
	EnvVarPatterns           *[]string `yaml:"envVarPatterns"`
	StrictImageValidation    *bool     `yaml:"strictImageValidation"`
}
