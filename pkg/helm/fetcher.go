package helm

import (
	"fmt"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/registry"
)

// Fetch resolves a chart reference and returns the loaded chart.
func Fetch(ref, version string) (*chart.Chart, error) {
	cfg := new(action.Configuration)
	regClient, err := registry.NewClient()
	if err != nil {
		return nil, err
	}
	cfg.RegistryClient = regClient
	client := action.NewInstall(cfg)
	client.Version = version

	chartPath, err := client.LocateChart(ref, cli.New())
	if err != nil {
		return nil, fmt.Errorf("failed to locate chart: %w", err)
	}

	chrt, err := loader.Load(chartPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load chart from %s: %w", chartPath, err)
	}

	return chrt, nil

}

// IsOCIRef reports whether ref is an OCI registry reference.
// Exported so tests can assert detection logic independently.
func IsOCIRef(ref string) bool {
	return registry.IsOCI(ref)
}
