package helm

import (
	"fmt"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/strvals"
)

// RenderOptions controls how a chart is rendered.
type RenderOptions struct {
	// Chart is the loaded Helm chart to render. Obtain it via Fetch.
	Chart *chart.Chart

	// Values are deep-merged over the chart's default values.yaml.
	Values map[string]interface{}

	// SetValues are --set style overrides applied after Values.
	SetValues []string

	// IncludeChartDependencies controls whether subchart templates are rendered.
	// When false, subcharts are removed from the chart before rendering.
	IncludeChartDependencies bool
}

// Render renders the chart, returning non-empty YAML documents.
func Render(opt RenderOptions) ([]string, error) {
	if opt.Chart == nil {
		return nil, fmt.Errorf("Render: Chart must not be nil")
	}
	chrt := opt.Chart

	if !opt.IncludeChartDependencies {
		chrt = stripDependencies(chrt)
	}

	vals, err := buildValues(chrt, opt)
	if err != nil {
		return nil, err
	}

	rendered, err := engine.Render(chrt, vals)
	if err != nil {
		return nil, fmt.Errorf("rendering chart: %w", err)
	}

	var docs []string
	for _, content := range rendered {
		content = strings.TrimSpace(content)
		if content != "" && content != "---" {
			docs = append(docs, content)
		}
	}
	return docs, nil
}

// buildValues merges chart defaults, manifest values, and --set overrides.
func buildValues(chrt *chart.Chart, opt RenderOptions) (chartutil.Values, error) {
	base := chrt.Values
	if base == nil {
		base = map[string]interface{}{}
	}

	// Deep-merge manifest values over chart defaults.
	deepMerge(base, opt.Values)

	// Apply --set overrides.
	for _, s := range opt.SetValues {
		if err := strvals.ParseInto(s, base); err != nil {
			return nil, fmt.Errorf("parsing --set %q: %w", s, err)
		}
	}

	return chartutil.ToRenderValues(chrt, base, chartutil.ReleaseOptions{
		Name:      chrt.Name(),
		Namespace: "default",
	}, nil)
}

// deepMerge merges src into dst recursively. Map values are merged; others are replaced.
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

// stripDependencies returns a shallow copy of chrt with no subcharts.
func stripDependencies(chrt *chart.Chart) *chart.Chart {
	copy := *chrt
	copy.SetDependencies()
	return &copy
}
