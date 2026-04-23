package extractor_test

import (
	"testing"

	"helmPackageImages/pkg/extractor"
)

func TestExtractEnvVars_MatchingPattern_Extracted(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
        - name: app
          image: nginx:1.25.3
          env:
            - name: SIDECAR_IMAGE
              value: myregistry.io/sidecar:v1.0.0
            - name: APP_PORT
              value: "8080"
`
	imgs, err := extractor.ExtractEnvVars(parseObjects(t, yaml), []string{`.*_IMAGE$`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"myregistry.io/sidecar:v1.0.0"}
	if got := sorted(imgs); !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractEnvVars_NoMatchingPattern_Empty(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
        - name: app
          image: nginx:1.25.3
          env:
            - name: APP_PORT
              value: "8080"
`
	imgs, err := extractor.ExtractEnvVars(parseObjects(t, yaml), []string{`.*_IMAGE$`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected no images, got %v", imgs)
	}
}

func TestExtractEnvVars_ValueFrom_Skipped(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
        - name: app
          image: nginx:1.25.3
          env:
            - name: SIDECAR_IMAGE
              valueFrom:
                configMapKeyRef:
                  name: config
                  key: image
`
	imgs, err := extractor.ExtractEnvVars(parseObjects(t, yaml), []string{`.*_IMAGE$`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected ValueFrom env var to be skipped, got %v", imgs)
	}
}

func TestExtractEnvVars_InitContainers_Included(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      initContainers:
        - name: init
          image: busybox:1.36
          env:
            - name: TOOL_IMAGE
              value: myregistry.io/tool:v1.0.0
      containers:
        - name: app
          image: nginx:1.25.3
`
	imgs, err := extractor.ExtractEnvVars(parseObjects(t, yaml), []string{`.*_IMAGE$`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "myregistry.io/tool:v1.0.0" {
		t.Errorf("expected init container env var to be extracted, got %v", imgs)
	}
}

func TestExtractEnvVars_MultiplePatterns_AnyMatch(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
        - name: app
          image: nginx:1.25.3
          env:
            - name: SIDECAR_IMAGE
              value: myregistry.io/sidecar:v1.0.0
            - name: IMAGE_PULL
              value: myregistry.io/puller:v1.0.0
            - name: OTHER_VAR
              value: not-an-image
`
	imgs, err := extractor.ExtractEnvVars(parseObjects(t, yaml), []string{`.*_IMAGE$`, `^IMAGE_`})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := sorted([]string{"myregistry.io/sidecar:v1.0.0", "myregistry.io/puller:v1.0.0"})
	if got := sorted(imgs); !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractEnvVars_InvalidPattern_ReturnsError(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
        - name: app
          image: nginx:1.25.3
`
	_, err := extractor.ExtractEnvVars(parseObjects(t, yaml), []string{`[invalid`})
	if err == nil {
		t.Error("expected error for invalid pattern, got nil")
	}
}

func TestExtractEnvVars_EmptyPatterns_ReturnsNil(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: app
spec:
  template:
    spec:
      containers:
        - name: app
          image: nginx:1.25.3
          env:
            - name: SIDECAR_IMAGE
              value: myregistry.io/sidecar:v1.0.0
`
	imgs, err := extractor.ExtractEnvVars(parseObjects(t, yaml), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if imgs != nil {
		t.Errorf("expected nil for empty patterns, got %v", imgs)
	}
}
