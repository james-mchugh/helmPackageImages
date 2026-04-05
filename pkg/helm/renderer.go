package helm

import (
	"fmt"
	"io"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
	"helm.sh/helm/v3/pkg/strvals"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/scheme"
)

// RenderOptions controls how a chart is rendered.
type RenderOptions struct {
	// Chart is the loaded Helm chart to render. Obtain it via Fetch.
	Chart *chart.Chart

	// Values are deep-merged over the chart's default values.yaml.
	Values map[string]interface{}

	// SetValues are --set style overrides applied after Values.
	SetValues []string
}

// Render renders the chart, returning Kubernetes objects.
func Render(opt RenderOptions) ([]runtime.Object, error) {
	if opt.Chart == nil {
		return nil, fmt.Errorf("Render: Chart must not be nil")
	}
	chrt := opt.Chart

	vals, err := buildValues(chrt, opt)
	if err != nil {
		return nil, err
	}

	err = chartutil.ProcessDependenciesWithMerge(chrt, vals)
	if err != nil {
		return nil, err
	}

	rendered, err := engine.Render(chrt, vals)
	if err != nil {
		return nil, fmt.Errorf("rendering chart: %w", err)
	}

	var objs []runtime.Object
	for filename, content := range rendered {
		if !(strings.HasSuffix(filename, ".yml") || strings.HasSuffix(filename, ".yaml")) {
			continue
		}
		renderedObjs, err := decodeObjects(content)
		if err != nil {
			return nil, err
		}
		objs = append(objs, renderedObjs...)
	}
	return objs, nil
}

// buildValues merges chart defaults, manifest values, and --set overrides.
func buildValues(chrt *chart.Chart, opt RenderOptions) (chartutil.Values, error) {

	base := opt.Values

	// Apply --set overrides.
	for _, s := range opt.SetValues {
		if err := strvals.ParseInto(s, base); err != nil {
			return nil, fmt.Errorf("parsing --set %q: %w", s, err)
		}
	}

	return chartutil.ToRenderValues(
		chrt, base, chartutil.ReleaseOptions{
			Name:      chrt.Name(),
			Namespace: "default",
		}, nil,
	)
}

func decodeObjects(data string) ([]runtime.Object, error) {
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()

	var objs []runtime.Object

	splitter := yaml.NewYAMLOrJSONDecoder(strings.NewReader(data), 4096)

	for {
		raw := runtime.RawExtension{}
		if err := splitter.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decoding YAML: %w", err)
		}

		if raw.Raw == nil {
			continue
		}

		obj, err := decodeOne(decoder, raw.Raw)
		if err != nil {
			return nil, err
		}

		objs = append(objs, obj)
	}

	return objs, nil
}

func decodeOne(decoder runtime.Decoder, raw []byte) (runtime.Object, error) {
	obj, _, err := decoder.Decode(raw, nil, nil)
	if err == nil {
		return obj, nil
	}

	if !runtime.IsNotRegisteredError(err) {
		return nil, err
	}

	// Fallback if CRD is not registered
	u := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(raw, u); err != nil {
		return nil, fmt.Errorf("decoding YAML: %w", err)
	}
	return u, nil
}
