package helm_test

import (
	"strings"
	"testing"

	helmrender "helmPackageImages/pkg/helm"
)

func TestFetch_LocalPath_LoadsChart(t *testing.T) {
	chrt, err := helmrender.Fetch(chartPath("simple"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if chrt == nil {
		t.Fatal("expected non-nil chart")
	}
	if chrt.Name() != "simple" {
		t.Errorf("expected chart name 'simple', got %q", chrt.Name())
	}
	if root == "" {
		t.Error("expected non-empty chart root for local chart")
	}
}

func TestFetch_LocalPath_NotExist_Error(t *testing.T) {
	_, err := helmrender.Fetch("/nonexistent/path/to/chart")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestFetch_RelativePath_Error(t *testing.T) {
	// Relative paths (starting with ./) are treated as local — load error expected.
	_, err := helmrender.Fetch("./nonexistent")
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
	_, err := helmrender.Fetch("oci://registry.example.invalid/charts/myapp:1.0.0")
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

// Version parsing tests — splitVersion is the internal function we expose for testing.

func TestSplitVersion_HTTPRepo_WithVersion(t *testing.T) {
	ref, version := helmrender.SplitVersion("stable/nginx:1.25.0")
	if ref != "stable/nginx" {
		t.Errorf("expected ref 'stable/nginx', got %q", ref)
	}
	if version != "1.25.0" {
		t.Errorf("expected version '1.25.0', got %q", version)
	}
}

func TestSplitVersion_HTTPRepo_NoVersion(t *testing.T) {
	ref, version := helmrender.SplitVersion("stable/nginx")
	if ref != "stable/nginx" {
		t.Errorf("expected ref 'stable/nginx', got %q", ref)
	}
	if version != "" {
		t.Errorf("expected empty version, got %q", version)
	}
}

func TestSplitVersion_OCI_Unchanged(t *testing.T) {
	// OCI refs use : natively for the tag — SplitVersion should NOT strip it.
	ref, version := helmrender.SplitVersion("oci://registry.example.com/charts/nginx:1.25.0")
	if ref != "oci://registry.example.com/charts/nginx:1.25.0" {
		t.Errorf("expected OCI ref unchanged, got %q", ref)
	}
	if version != "" {
		t.Errorf("expected empty version for OCI ref (tag is part of ref), got %q", version)
	}
}

func TestSplitVersion_LocalPath_Unchanged(t *testing.T) {
	// Local paths should not be split even if they contain a colon (edge case on Linux).
	ref, version := helmrender.SplitVersion("./my-chart")
	if ref != "./my-chart" {
		t.Errorf("expected local path unchanged, got %q", ref)
	}
	if version != "" {
		t.Errorf("expected empty version for local path, got %q", version)
	}
}

func TestSplitVersion_AbsLocalPath_Unchanged(t *testing.T) {
	ref, version := helmrender.SplitVersion("/home/user/charts/my-chart")
	if ref != "/home/user/charts/my-chart" {
		t.Errorf("expected abs path unchanged, got %q", ref)
	}
	if version != "" {
		t.Errorf("expected empty version for abs path, got %q", version)
	}
}
