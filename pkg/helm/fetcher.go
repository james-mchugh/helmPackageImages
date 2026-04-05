package helm

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
)

// Fetch resolves a chart reference and returns the loaded chart.
//
// Version is specified inline in the reference:
//   - OCI:       oci://registry/chart:1.2.3  (standard OCI tag, passed through as-is)
//   - HTTP repo: stable/nginx:1.2.3          (version split off before calling action.Pull)
//   - Local:     ./my-chart or /path/chart   (no version concept)
func Fetch(ref string) (*chart.Chart, error) {
	cleanRef, version := SplitVersion(ref)

	cfg := new(action.Configuration)
	client := action.NewPullWithOpts(action.WithConfig(cfg))
	client.Settings = cli.New()
	client.Version = version

	chartPath, err := client.ChartPathOptions.LocateChart(cleanRef, client.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart: %w", err)
	}

	chrt, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	return chrt, nil

}

// SplitVersion separates an inline version from a chart reference.
// For OCI and local refs, the ref is returned unchanged with an empty version.
// For HTTP repo refs, the last ":" segment is treated as the version.
//
// Exported so tests can assert the parsing logic directly.
func SplitVersion(ref string) (cleanRef, version string) {
	// OCI refs own their colon (it's the tag separator) — don't touch them.
	if IsOCIRef(ref) {
		return ref, ""
	}
	// Local path prefixes — no version concept.
	if strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "./") || strings.HasPrefix(ref, "../") {
		return ref, ""
	}
	// HTTP repo: split on last colon.
	if idx := strings.LastIndex(ref, ":"); idx >= 0 {
		return ref[:idx], ref[idx+1:]
	}
	return ref, ""
}

// IsOCIRef reports whether ref is an OCI registry reference.
// Exported so tests can assert detection logic independently.
func IsOCIRef(ref string) bool {
	return registry.IsOCI(ref)
}
