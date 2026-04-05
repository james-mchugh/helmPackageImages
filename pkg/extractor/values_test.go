package extractor_test

import (
	"testing"

	"helmPackageImages/pkg/extractor"
)

func TestScrapeValues_Disabled(t *testing.T) {
	values := map[string]interface{}{
		"image": "nginx:1.25.3",
	}
	imgs := extractor.ScrapeValues(values, false)
	if len(imgs) != 0 {
		t.Errorf("expected no images when disabled, got %v", imgs)
	}
}

func TestScrapeValues_FullImageString(t *testing.T) {
	values := map[string]interface{}{
		"image": "nginx:1.25.3",
	}
	imgs := extractor.ScrapeValues(values, true)
	if len(imgs) != 1 || imgs[0] != "nginx:1.25.3" {
		t.Errorf("got %v", imgs)
	}
}

func TestScrapeValues_RepositoryAndTag(t *testing.T) {
	values := map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "nginx",
			"tag":        "1.25.3",
		},
	}
	imgs := extractor.ScrapeValues(values, true)
	if len(imgs) != 1 || imgs[0] != "nginx:1.25.3" {
		t.Errorf("got %v", imgs)
	}
}

func TestScrapeValues_RegistryRepositoryAndTag(t *testing.T) {
	values := map[string]interface{}{
		"image": map[string]interface{}{
			"registry":   "registry.example.com",
			"repository": "myapp",
			"tag":        "v1.0",
		},
	}
	imgs := extractor.ScrapeValues(values, true)
	if len(imgs) != 1 || imgs[0] != "registry.example.com/myapp:v1.0" {
		t.Errorf("got %v", imgs)
	}
}

func TestScrapeValues_NonImageStrings_Excluded(t *testing.T) {
	values := map[string]interface{}{
		"name":        "myapp",
		"replicas":    3,
		"description": "just a string",
		"url":         "https://example.com",
	}
	imgs := extractor.ScrapeValues(values, true)
	if len(imgs) != 0 {
		t.Errorf("expected no images for non-image strings, got %v", imgs)
	}
}

func TestScrapeValues_NestedValues(t *testing.T) {
	values := map[string]interface{}{
		"component": map[string]interface{}{
			"enabled": true,
			"image": map[string]interface{}{
				"repository": "redis",
				"tag":        "7.2",
			},
		},
	}
	imgs := extractor.ScrapeValues(values, true)
	if len(imgs) != 1 || imgs[0] != "redis:7.2" {
		t.Errorf("got %v", imgs)
	}
}

func TestScrapeValues_RepositoryWithoutTag(t *testing.T) {
	values := map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "nginx",
		},
	}
	imgs := extractor.ScrapeValues(values, true)
	// repository without tag should still be captured as-is
	if len(imgs) != 1 || imgs[0] != "nginx" {
		t.Errorf("got %v", imgs)
	}
}

func TestScrapeValues_FullImageStringWithRegistry(t *testing.T) {
	values := map[string]interface{}{
		"image": "registry.example.com/myapp:v1.0",
	}
	imgs := extractor.ScrapeValues(values, true)
	if len(imgs) != 1 || imgs[0] != "registry.example.com/myapp:v1.0" {
		t.Errorf("got %v", imgs)
	}
}
