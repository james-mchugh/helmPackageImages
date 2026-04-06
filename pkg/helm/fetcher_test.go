package helm_test

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	helmrender "helmPackageImages/pkg/helm"
)

func chartPath(name string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata", "charts", name)
}

func TestFetch_LocalPath_LoadsChart(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("simple"), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chrt == nil {
		t.Fatal("expected non-nil chart")
	}
	if chrt.Name() != "simple" {
		t.Errorf("expected chart name 'simple', got %q", chrt.Name())
	}
}

func TestFetch_LocalPath_NotExist_Error(t *testing.T) {
	_, err := helmrender.Fetch("/nonexistent/path/to/chart", "")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestFetch_RelativePath_Error(t *testing.T) {
	// Relative paths (starting with ./) are treated as local — load error expected.
	_, err := helmrender.Fetch("./nonexistent", "")
	if err == nil {
		t.Error("expected error for nonexistent relative path")
	}
	// Should be a load error, not "unknown reference type".
	if err != nil && strings.Contains(err.Error(), "unknown reference type") {
		t.Errorf("relative path misidentified as unknown reference: %v", err)
	}
}

func TestFetch_OCIRef_Identified(t *testing.T) {
	// We can't actually pull an OCI chart in unit tests, but we verify
	// the ref is routed as OCI (network error expected, not misrouting error).
	_, err := helmrender.Fetch("oci://registry.example.invalid/charts/myapp", "1.0.0")
	if err == nil {
		t.Error("expected error for unreachable OCI registry")
	}
	if strings.Contains(err.Error(), "unknown reference type") {
		t.Errorf("OCI ref misidentified: %v", err)
	}
}

func TestFetch_OCI_RootIsEmpty(t *testing.T) {
	if !helmrender.IsOCIRef("oci://registry.example.com/chart:1.0.0") {
		t.Error("expected oci:// prefix to be identified as OCI ref")
	}
	if helmrender.IsOCIRef("stable/nginx") {
		t.Error("expected stable/nginx NOT to be identified as OCI ref")
	}
	if helmrender.IsOCIRef("./local/chart") {
		t.Error("expected local path NOT to be identified as OCI ref")
	}
}
