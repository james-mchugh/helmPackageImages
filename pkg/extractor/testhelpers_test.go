package extractor_test

import (
	"io"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/kubectl/pkg/scheme"
)

// parseObjects decodes a YAML string (potentially multi-document) into runtime.Objects.
// Registered K8s types are decoded to their typed structs; unregistered CRD types
// fall back to *unstructured.Unstructured.
func parseObjects(t *testing.T, yamlStr string) []runtime.Object {
	t.Helper()
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()
	splitter := k8syaml.NewYAMLOrJSONDecoder(strings.NewReader(yamlStr), 4096)
	var objs []runtime.Object
	for {
		raw := runtime.RawExtension{}
		if err := splitter.Decode(&raw); err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("parseObjects decode: %v", err)
		}
		if raw.Raw == nil {
			continue
		}
		obj, _, err := decoder.Decode(raw.Raw, nil, nil)
		if err != nil {
			if runtime.IsNotRegisteredError(err) {
				u := &unstructured.Unstructured{}
				if err := k8syaml.Unmarshal(raw.Raw, u); err != nil {
					t.Fatalf("parseObjects unmarshal unstructured: %v", err)
				}
				objs = append(objs, u)
			} else {
				t.Fatalf("parseObjects decode: %v", err)
			}
		} else {
			objs = append(objs, obj)
		}
	}
	return objs
}
