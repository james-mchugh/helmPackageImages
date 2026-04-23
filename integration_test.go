package main_test

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"helmPackageImages/pkg/extractor"
	helmrender "helmPackageImages/pkg/helm"
	"helmPackageImages/pkg/manifest"
)

func chartPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata/charts", name)
}

// imageDiscoveryPipeline runs the full manifest+fetch+render+extract pipeline and
// returns the discovered image list. Does not pull or archive anything.
func imageDiscoveryPipeline(t *testing.T, chartRef string, manifestOpts manifest.Options) []string {
	t.Helper()

	// Fetch chart.
	chrt, err := helmrender.Fetch(chartRef, "")
	if err != nil {
		t.Fatalf("helm.Fetch: %v", err)
	}

	// Load manifest — inject the fetched chart so embedded airgap.yaml is found.
	if manifestOpts.Chart == nil {
		manifestOpts.Chart = chrt
	}
	m, err := manifest.Load(manifestOpts)
	if err != nil {
		t.Fatalf("manifest.Load: %v", err)
	}

	// Render chart.
	docs, err := helmrender.Render(helmrender.RenderOptions{
		Chart:  chrt,
		Values: m.Values,
	})
	if err != nil {
		t.Fatalf("helm.Render: %v", err)
	}

	imgs, err := extractor.Extract(docs, m)
	if err != nil {
		t.Fatalf("extractor.Extract: %v", err)
	}
	return imgs
}

func TestIntegration_SimpleChart_KnownImages(t *testing.T) {
	imgs := imageDiscoveryPipeline(t, chartPath("simple"), manifest.Options{})
	want := []string{"busybox:1.36", "nginx:1.25.3"}
	sort.Strings(imgs)
	if !equal(imgs, want) {
		t.Errorf("got %v, want %v", imgs, want)
	}
}

func TestIntegration_DisabledComponent_ImageAbsent(t *testing.T) {
	imgs := imageDiscoveryPipeline(t, chartPath("with-disabled-component"), manifest.Options{})
	for _, img := range imgs {
		if strings.Contains(img, "redis") {
			t.Errorf("expected redis absent when worker.enabled=false, got %v", imgs)
		}
	}
	found := false
	for _, img := range imgs {
		if img == "nginx:1.25.3" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected nginx:1.25.3 present, got %v", imgs)
	}
}

func TestIntegration_DisabledComponent_EnabledViaManifestValues(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "airgap.yaml", `
values:
  worker:
    enabled: true
settings:
  includeChartDependencies: true
`)
	imgs := imageDiscoveryPipeline(t, chartPath("with-disabled-component"),
		manifest.Options{ManifestPath: filepath.Join(dir, "airgap.yaml")},
	)
	found := false
	for _, img := range imgs {
		if img == "redis:7.2" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected redis:7.2 after enabling worker via manifest values, got %v", imgs)
	}
}

func TestIntegration_CRDs_CustomResourceImageExtracted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "airgap.yaml", `
crds:
  - kind: MyOperator
    apiVersion: mygroup.io/v1alpha1
    imagePaths:
      - "{.spec.image}"
      - "{.spec.sidecarImage}"
`)
	imgs := imageDiscoveryPipeline(t, chartPath("with-crds"),
		manifest.Options{ManifestPath: filepath.Join(dir, "airgap.yaml")},
	)
	want := []string{
		"example.com/myapp:v2.0.0",
		"example.com/operator:v1.0.0",
		"example.com/sidecar:v1.1.0",
	}
	sort.Strings(imgs)
	if !equal(imgs, want) {
		t.Errorf("got %v, want %v", imgs, want)
	}
}

func TestIntegration_SubchartDependencies_Excluded(t *testing.T) {
	f := false
	imgs := imageDiscoveryPipeline(t, chartPath("with-subcharts"),
		manifest.Options{OverrideIncludeDeps: &f},
	)
	for _, img := range imgs {
		if strings.Contains(img, "redis") {
			t.Errorf("expected redis absent when dependencies excluded, got %v", imgs)
		}
	}
}

func TestIntegration_SubchartDependencies_IncludedAndEnabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "airgap.yaml", `
values:
  redis:
    enabled: true
settings:
  includeChartDependencies: true
`)
	imgs := imageDiscoveryPipeline(t, chartPath("with-subcharts"),
		manifest.Options{ManifestPath: filepath.Join(dir, "airgap.yaml")},
	)
	found := false
	for _, img := range imgs {
		if img == "redis:7.2" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected redis:7.2 when subchart enabled, got %v", imgs)
	}
}

func TestIntegration_EnvVarExtraction(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "airgap.yaml", `
settings:
  envVarPatterns:
    - ".*_IMAGE$"
`)
	imgs := imageDiscoveryPipeline(t, chartPath("configmap-images"),
		manifest.Options{ManifestPath: filepath.Join(dir, "airgap.yaml")},
	)
	// Builtin: myregistry.io/app:v1.0.0; env var: myregistry.io/sidecar:v1.0.0
	want := []string{"myregistry.io/app:v1.0.0", "myregistry.io/sidecar:v1.0.0"}
	sort.Strings(imgs)
	if !equal(imgs, want) {
		t.Errorf("got %v, want %v", imgs, want)
	}
}

func TestIntegration_ConfigMapExtraction_HeuristicMode(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "airgap.yaml", `
configMapRules:
  - namePattern: "app-images"
`)
	imgs := imageDiscoveryPipeline(t, chartPath("configmap-images"),
		manifest.Options{ManifestPath: filepath.Join(dir, "airgap.yaml")},
	)
	// Builtin: myregistry.io/app:v1.0.0; configmap: myregistry.io/extra:v1.0.0, myregistry.io/tool:v2.0.0
	want := []string{"myregistry.io/app:v1.0.0", "myregistry.io/extra:v1.0.0", "myregistry.io/tool:v2.0.0"}
	sort.Strings(imgs)
	if !equal(imgs, want) {
		t.Errorf("got %v, want %v", imgs, want)
	}
}

func TestIntegration_EnvVarAndConfigMapCombined(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "airgap.yaml", `
settings:
  envVarPatterns:
    - ".*_IMAGE$"
configMapRules:
  - namePattern: "app-images"
`)
	imgs := imageDiscoveryPipeline(t, chartPath("configmap-images"),
		manifest.Options{ManifestPath: filepath.Join(dir, "airgap.yaml")},
	)
	want := []string{
		"myregistry.io/app:v1.0.0",
		"myregistry.io/extra:v1.0.0",
		"myregistry.io/sidecar:v1.0.0",
		"myregistry.io/tool:v2.0.0",
	}
	sort.Strings(imgs)
	if !equal(imgs, want) {
		t.Errorf("got %v, want %v", imgs, want)
	}
}

// helpers

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
