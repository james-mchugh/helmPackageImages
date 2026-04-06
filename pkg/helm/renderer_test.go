package helm_test

import (
	"testing"

	"helmPackageImages/pkg/extractor"
	helmrender "helmPackageImages/pkg/helm"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// extractImages returns all images from built-in workload objects.
func extractImages(t *testing.T, objs []runtime.Object) []string {
	t.Helper()
	imgs, err := extractor.ExtractBuiltin(objs)
	if err != nil {
		t.Fatalf("ExtractBuiltin: %v", err)
	}
	return imgs
}

func containsImage(imgs []string, img string) bool {
	for _, i := range imgs {
		if i == img {
			return true
		}
	}
	return false
}

func hasKind(objs []runtime.Object, kind string) bool {
	for _, obj := range objs {
		if u, ok := obj.(*unstructured.Unstructured); ok {
			if u.GetKind() == kind {
				return true
			}
		}
	}
	return false
}

func TestRender_SimpleChart(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("simple"), "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	objs, err := helmrender.Render(helmrender.RenderOptions{Chart: chrt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(objs) == 0 {
		t.Fatal("expected at least one rendered object")
	}
	imgs := extractImages(t, objs)
	if !containsImage(imgs, "nginx:1.25.3") {
		t.Error("expected nginx:1.25.3 in rendered output")
	}
	if !containsImage(imgs, "busybox:1.36") {
		t.Error("expected busybox:1.36 in rendered output")
	}
}

func TestRender_DisabledComponentExcluded(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-disabled-component"), "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	objs, err := helmrender.Render(helmrender.RenderOptions{Chart: chrt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	imgs := extractImages(t, objs)
	if containsImage(imgs, "redis:7.2") {
		t.Error("expected redis:7.2 to be absent when worker.enabled=false")
	}
	if !containsImage(imgs, "nginx:1.25.3") {
		t.Error("expected nginx:1.25.3 to be present")
	}
}

func TestRender_ValuesEnable_HiddenComponent(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-disabled-component"), "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	objs, err := helmrender.Render(helmrender.RenderOptions{
		Chart: chrt,
		Values: map[string]interface{}{
			"worker": map[string]interface{}{
				"enabled": true,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	imgs := extractImages(t, objs)
	if !containsImage(imgs, "redis:7.2") {
		t.Error("expected redis:7.2 after enabling worker")
	}
}

func TestRender_SetOverrides(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-disabled-component"), "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	objs, err := helmrender.Render(helmrender.RenderOptions{
		Chart:     chrt,
		SetValues: []string{"worker.enabled=true"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	imgs := extractImages(t, objs)
	if !containsImage(imgs, "redis:7.2") {
		t.Error("expected redis:7.2 after --set worker.enabled=true")
	}
}

func TestRender_DependencyDisabledViaValues(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-subcharts"), "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	// Explicitly disable the redis subchart via its condition value.
	objs, err := helmrender.Render(helmrender.RenderOptions{
		Chart: chrt,
		Values: map[string]interface{}{
			"redis": map[string]interface{}{
				"enabled": false,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	imgs := extractImages(t, objs)
	if containsImage(imgs, "redis:7.2") {
		t.Error("expected redis:7.2 absent when redis.enabled=false")
	}
}

func TestRender_DependencyEnabledViaValues(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-subcharts"), "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	// Enable the redis subchart via its condition value.
	objs, err := helmrender.Render(helmrender.RenderOptions{
		Chart: chrt,
		Values: map[string]interface{}{
			"redis": map[string]interface{}{
				"enabled": true,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	imgs := extractImages(t, objs)
	if !containsImage(imgs, "redis:7.2") {
		t.Error("expected redis:7.2 when redis.enabled=true")
	}
}

func TestRender_CustomResourcePresent(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-crds"), "")
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	objs, err := helmrender.Render(helmrender.RenderOptions{Chart: chrt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasKind(objs, "MyOperator") {
		t.Error("expected custom resource MyOperator in rendered output")
	}
}

func TestRender_NilChart_Error(t *testing.T) {
	_, err := helmrender.Render(helmrender.RenderOptions{Chart: nil})
	if err == nil {
		t.Error("expected error when Chart is nil")
	}
}
