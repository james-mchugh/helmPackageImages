package helm_test

import (
	"strings"
	"testing"

	helmrender "helmPackageImages/pkg/helm"
)

func TestRender_SimpleChart(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("simple"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	docs, err := helmrender.Render(helmrender.RenderOptions{Chart: chrt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(docs) == 0 {
		t.Fatal("expected at least one rendered document")
	}
	combined := strings.Join(docs, "\n")
	if !strings.Contains(combined, "nginx:1.25.3") {
		t.Error("expected nginx:1.25.3 in rendered output")
	}
	if !strings.Contains(combined, "busybox:1.36") {
		t.Error("expected busybox:1.36 in rendered output")
	}
}

func TestRender_DisabledComponentExcluded(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-disabled-component"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	docs, err := helmrender.Render(helmrender.RenderOptions{Chart: chrt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(docs, "\n")
	if strings.Contains(combined, "redis:7.2") {
		t.Error("expected redis:7.2 to be absent when worker.enabled=false")
	}
	if !strings.Contains(combined, "nginx:1.25.3") {
		t.Error("expected nginx:1.25.3 to be present")
	}
}

func TestRender_ValuesEnable_HiddenComponent(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-disabled-component"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	docs, err := helmrender.Render(helmrender.RenderOptions{
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
	combined := strings.Join(docs, "\n")
	if !strings.Contains(combined, "redis:7.2") {
		t.Error("expected redis:7.2 after enabling worker")
	}
}

func TestRender_SetOverrides(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-disabled-component"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	docs, err := helmrender.Render(helmrender.RenderOptions{
		Chart:     chrt,
		SetValues: []string{"worker.enabled=true"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(docs, "\n")
	if !strings.Contains(combined, "redis:7.2") {
		t.Error("expected redis:7.2 after --set worker.enabled=true")
	}
}

func TestRender_ExcludeDependencies(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-subcharts"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	docs, err := helmrender.Render(helmrender.RenderOptions{
		Chart:                    chrt,
		IncludeChartDependencies: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(docs, "\n")
	if strings.Contains(combined, "redis:7.2") {
		t.Error("expected redis:7.2 absent when dependencies excluded")
	}
}

func TestRender_IncludeDependencies(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-subcharts"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	docs, err := helmrender.Render(helmrender.RenderOptions{
		Chart:                    chrt,
		IncludeChartDependencies: true,
		Values: map[string]interface{}{
			"redis": map[string]interface{}{
				"enabled": true,
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(docs, "\n")
	if !strings.Contains(combined, "redis:7.2") {
		t.Error("expected redis:7.2 when dependencies included and enabled")
	}
}

func TestRender_CustomResourcePresent(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("with-crds"))
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	docs, err := helmrender.Render(helmrender.RenderOptions{Chart: chrt})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	combined := strings.Join(docs, "\n")
	if !strings.Contains(combined, "MyOperator") {
		t.Error("expected custom resource MyOperator in rendered output")
	}
}

func TestRender_NilChart_Error(t *testing.T) {
	_, err := helmrender.Render(helmrender.RenderOptions{Chart: nil})
	if err == nil {
		t.Error("expected error when Chart is nil")
	}
}
