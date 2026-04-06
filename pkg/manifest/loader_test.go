package manifest_test

import (
	"os"
	"path/filepath"
	"testing"

	"helm.sh/helm/v3/pkg/chart"

	"helmPackageImages/pkg/manifest"
)

// chartWithManifest builds a minimal *chart.Chart with airgap.yaml embedded.
func chartWithManifest(content string) *chart.Chart {
	return &chart.Chart{
		Files: []*chart.File{
			{Name: "airgap.yaml", Data: []byte(content)},
		},
	}
}

func TestLoad_MinimalManifest(t *testing.T) {
	m, err := manifest.Load(manifest.Options{
		Chart: chartWithManifest(`
crds:
  - kind: MyOp
    apiVersion: mygroup.io/v1alpha1
    imagePaths:
      - .spec.image
values:
  component:
    enabled: true
settings:
  platform: linux/amd64
  includeChartDependencies: false
  scrapeValues: true
`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.CRDs) != 1 || m.CRDs[0].Kind != "MyOp" {
		t.Errorf("expected 1 CRD with kind MyOp, got %+v", m.CRDs)
	}
	if m.Settings.Platform != "linux/amd64" {
		t.Errorf("expected platform linux/amd64, got %q", m.Settings.Platform)
	}
	if !m.Settings.ScrapeValues {
		t.Error("expected scrapeValues true")
	}
}

func TestLoad_MissingManifest_ReturnsDefaults(t *testing.T) {
	m, err := manifest.Load(manifest.Options{Chart: &chart.Chart{}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.CRDs) != 0 {
		t.Errorf("expected no CRDs, got %v", m.CRDs)
	}
	if m.Settings.ScrapeValues {
		t.Error("expected scrapeValues false by default")
	}
}

func TestLoad_ExplicitManifestPath(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "custom.yaml", `
settings:
  platform: linux/arm64
`)
	m, err := manifest.Load(manifest.Options{ManifestPath: path})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Settings.Platform != "linux/arm64" {
		t.Errorf("expected linux/arm64, got %q", m.Settings.Platform)
	}
}

func TestLoad_ProfileMerge_Settings(t *testing.T) {
	m, err := manifest.Load(manifest.Options{
		Chart: chartWithManifest(`
settings:
  platform: linux/amd64
  includeChartDependencies: true
  scrapeValues: false
profiles:
  multi-arch:
    settings:
      platform: linux/amd64,linux/arm64
`),
		Profile: "multi-arch",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Settings.Platform != "linux/amd64,linux/arm64" {
		t.Errorf("expected multi-arch platform, got %q", m.Settings.Platform)
	}
}

func TestLoad_ProfileMerge_Values(t *testing.T) {
	m, err := manifest.Load(manifest.Options{
		Chart: chartWithManifest(`
values:
  component:
    enabled: true
  other: foo
profiles:
  disable-component:
    values:
      component:
        enabled: false
`),
		Profile: "disable-component",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	comp, ok := m.Values["component"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected component to be a map, got %T", m.Values["component"])
	}
	if comp["enabled"] != false {
		t.Errorf("expected component.enabled=false after profile merge, got %v", comp["enabled"])
	}
	// Base key not overridden by profile should remain.
	if m.Values["other"] != "foo" {
		t.Errorf("expected other=foo to survive profile merge, got %v", m.Values["other"])
	}
}

func TestLoad_ProfileMerge_CRDs_Replaced(t *testing.T) {
	m, err := manifest.Load(manifest.Options{
		Chart: chartWithManifest(`
crds:
  - kind: Base
    apiVersion: a/v1
    imagePaths: [.spec.image]
profiles:
  no-crds:
    crds: []
`),
		Profile: "no-crds",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.CRDs) != 0 {
		t.Errorf("expected profile to replace CRDs with empty list, got %v", m.CRDs)
	}
}

func TestLoad_ProfileMerge_CRDs_NotApplied_WhenAbsent(t *testing.T) {
	m, err := manifest.Load(manifest.Options{
		Chart: chartWithManifest(`
crds:
  - kind: Base
    apiVersion: a/v1
    imagePaths: [.spec.image]
profiles:
  settings-only:
    settings:
      scrapeValues: true
`),
		Profile: "settings-only",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m.CRDs) != 1 {
		t.Errorf("expected base CRDs to be preserved when profile has none, got %v", m.CRDs)
	}
}

func TestLoad_CLIOverrides(t *testing.T) {
	trueVal := true
	m, err := manifest.Load(manifest.Options{
		Chart: chartWithManifest(`
settings:
  platform: linux/amd64
  scrapeValues: false
  includeChartDependencies: true
`),
		OverridePlatform:     "linux/arm64",
		OverrideScrapeValues: &trueVal,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.Settings.Platform != "linux/arm64" {
		t.Errorf("expected CLI platform override, got %q", m.Settings.Platform)
	}
	if !m.Settings.ScrapeValues {
		t.Error("expected CLI scrapeValues override to true")
	}
}

func TestLoad_UnknownProfile_ReturnsError(t *testing.T) {
	_, err := manifest.Load(manifest.Options{
		Chart:   chartWithManifest("profiles:\n  real: {}\n"),
		Profile: "nonexistent",
	})
	if err == nil {
		t.Error("expected error for unknown profile")
	}
}

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeFile: %v", err)
	}
	return path
}

func boolPtr(b bool) *bool { return &b }
