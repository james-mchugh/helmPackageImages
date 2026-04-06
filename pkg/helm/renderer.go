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

	// Values are user-supplied overrides. Chart default values are merged by
	// ToRenderValues during rendering; these are applied on top of those defaults.
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

	// ProcessDependenciesWithMerge must receive raw user values (not the wrapped
	// render values from ToRenderValues) so that condition paths like "redis.enabled"
	// resolve correctly against the top-level map. ToRenderValues is called after,
	// which merges chart defaults internally.
	rawVals, err := buildValues(opt)
	if err != nil {
		return nil, err
	}

	if err := chartutil.ProcessDependenciesWithMerge(chrt, rawVals); err != nil {
		return nil, err
	}

	vals, err := chartutil.ToRenderValues(chrt, rawVals, chartutil.ReleaseOptions{
		Name:      chrt.Name(),
		Namespace: "default",
	}, nil)
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

// buildValues assembles the raw user-supplied values from opt.Values and --set overrides.
// Chart default values are NOT merged here; ToRenderValues handles that internally.
func buildValues(opt RenderOptions) (map[string]interface{}, error) {
	base := make(map[string]interface{})
	for k, v := range opt.Values {
		base[k] = v
	}

	// Apply --set overrides.
	for _, s := range opt.SetValues {
		if err := strvals.ParseInto(s, base); err != nil {
			return nil, fmt.Errorf("parsing --set %q: %w", s, err)
		}
	}

	return base, nil
}

// decodeObjects splits a multi-document YAML string and decodes each document
// into a typed runtime.Object. Unknown/CRD types fall back to *unstructured.Unstructured.
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

// decodeOne decodes a single raw JSON/YAML document. If the type is not registered
// in the scheme (e.g. a CRD), it falls back to *unstructured.Unstructured.
func decodeOne(decoder runtime.Decoder, raw []byte) (runtime.Object, error) {
	obj, _, err := decoder.Decode(raw, nil, nil)
	if err == nil {
		return obj, nil
	}

	if !runtime.IsNotRegisteredError(err) {
		return nil, err
	}

	// Type not registered in the scheme (e.g. a custom resource) — decode as unstructured.
	u := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(raw, u); err != nil {
		return nil, fmt.Errorf("decoding YAML: %w", err)
	}
	return u, nil
}
