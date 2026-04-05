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
		objType, ok := obj.(*runtime.Unknown)
		if !ok {
			return nil, fmt.Errorf("expected *runtime.Unknown, got %T", obj)
		}
		entry, ok := index[objType.TypeMeta]
		if !ok {
			continue
		}
		u := obj.(*unstructured.Unstructured)
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

// extractPaths resolves a  JSONPath (e.g. "{.spec.image}") against a
// generic map. Returns empty string if any segment is missing or not a string.
func extractPaths(obj map[string]interface{}, path string) ([]string, error) {
	jp := jsonpath.New(path)
	if err := jp.Parse(path); err != nil {
		// todo: validate this when manifest is parsed
		return nil, fmt.Errorf("failed to parse JSON path expression")
	}

	results, err := jp.FindResults(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to find JSON path expression")
	}

	var imgs []string
	for _, result := range results {
		for _, result := range result {
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
