package extractor_test

import (
	"fmt"
	"testing"

	"helmPackageImages/pkg/extractor"
)

func noopLogf(string, ...any) {}

func collectingLogf(collected *[]string) func(string, ...any) {
	return func(format string, args ...any) {
		*collected = append(*collected, fmt.Sprintf(format, args...))
	}
}

func TestValidateImages_AllValid_Unchanged(t *testing.T) {
	imgs := []string{"nginx:1.25.3", "docker.io/library/nginx:latest", "myregistry.io/app:v1"}
	got, err := extractor.ValidateImages(imgs, false, noopLogf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !equal(sorted(got), sorted(imgs)) {
		t.Errorf("got %v, want %v", got, imgs)
	}
}

func TestValidateImages_InvalidRef_WarnedAndFiltered(t *testing.T) {
	var warnings []string
	imgs := []string{"nginx:1.25.3", "not a valid ref!!"}
	got, err := extractor.ValidateImages(imgs, false, collectingLogf(&warnings))
	if err != nil {
		t.Fatalf("unexpected error in non-strict mode: %v", err)
	}
	if len(got) != 1 || got[0] != "nginx:1.25.3" {
		t.Errorf("expected only valid image in result, got %v", got)
	}
	if len(warnings) != 1 {
		t.Errorf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
}

func TestValidateImages_Strict_ReturnsError(t *testing.T) {
	imgs := []string{"nginx:1.25.3", "not a valid ref!!"}
	_, err := extractor.ValidateImages(imgs, true, noopLogf)
	if err == nil {
		t.Error("expected error in strict mode for invalid ref, got nil")
	}
}

func TestValidateImages_Strict_AllValid_NoError(t *testing.T) {
	imgs := []string{"nginx:1.25.3", "myregistry.io/app:v1"}
	got, err := extractor.ValidateImages(imgs, true, noopLogf)
	if err != nil {
		t.Fatalf("unexpected error when all refs are valid: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 images, got %v", got)
	}
}

func TestValidateImages_WeakValidation_TaglessRefAccepted(t *testing.T) {
	// "nginx" with no tag should be accepted (treated as nginx:latest by WeakValidation).
	got, err := extractor.ValidateImages([]string{"nginx"}, false, noopLogf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0] != "nginx" {
		t.Errorf("expected tagless ref to pass WeakValidation, got %v", got)
	}
}

func TestValidateImages_EmptyInput_ReturnsEmpty(t *testing.T) {
	got, err := extractor.ValidateImages(nil, false, noopLogf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result, got %v", got)
	}
}

func TestValidateImages_Strict_LogsBeforeReturningError(t *testing.T) {
	var warnings []string
	imgs := []string{"nginx:1.25.3", "bad ref!!"}
	_, err := extractor.ValidateImages(imgs, true, collectingLogf(&warnings))
	if err == nil {
		t.Fatal("expected error in strict mode")
	}
	if len(warnings) != 1 {
		t.Errorf("expected warning logged even in strict mode, got %d warnings", len(warnings))
	}
}
