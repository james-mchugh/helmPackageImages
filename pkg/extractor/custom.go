package extractor

import (
	"fmt"
	"helmPackageImages/pkg/manifest"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/jsonpath"
)

// ExtractCustom scans rendered Kubernetes objects for images in custom resources
// described by the given CRD entries. Each entry specifies a kind+apiVersion
// to match and JSONPath-style dot-notation paths to image fields.
func ExtractCustom(objs []runtime.Object, crds []manifest.CRDEntry) ([]string, error) {
	if len(crds) == 0 {
		return nil, nil
	}

	// Build a lookup keyed by "kind/apiVersion" for O(1) matching.
	index := make(map[runtime.TypeMeta]*manifest.CRDEntry, len(crds))
	for i := range crds {
		k := crds[i].TypeMeta
		index[k] = &crds[i]
	}

	seen := map[string]struct{}{}
	for _, obj := range objs {
		u, ok := obj.(*unstructured.Unstructured)
		if !ok {
			continue // registered K8s types are handled by ExtractBuiltin
		}
		key := runtime.TypeMeta{Kind: u.GetKind(), APIVersion: u.GetAPIVersion()}
		entry, ok := index[key]
		if !ok {
			continue
		}
		for _, imgPath := range entry.ImagePaths {
			imgs, err := extractPaths(u.Object, imgPath)
			if err != nil {
				return nil, err
			}
			for _, img := range imgs {
				if img != "" {
					seen[img] = struct{}{}
				}
			}

		}
	}
	return setToSlice(seen), nil
}

// extractPaths resolves a JSONPath expression (e.g. "{.spec.image}") against an
// unstructured object map. Returns nil if the path is not found. Returns an error
// if the path expression is syntactically invalid or a matched value is not a string.
func extractPaths(obj map[string]interface{}, path string) ([]string, error) {
	jp := jsonpath.New(path)
	if err := jp.Parse(path); err != nil {
		return nil, fmt.Errorf("invalid JSONPath expression %q: %w", path, err)
	}

	results, err := jp.FindResults(obj)
	if err != nil {
		return nil, nil // path not found — skip gracefully
	}

	var imgs []string
	for _, result := range results {
		for _, result := range result {
			// Unstructured maps store values as interface{}; unwrap one level.
			if result.Kind() == reflect.Interface {
				result = result.Elem()
			}
			if result.Kind() != reflect.String {
				return nil, fmt.Errorf(
					"expected JSONPath expression to return String, received %s",
					result.Kind().String(),
				)
			}
			img := result.String()
			if img != "" {
				imgs = append(imgs, img)
			}
		}
	}

	return imgs, nil
}
