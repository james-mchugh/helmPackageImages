package extractor_test

import (
	"testing"

	"helmPackageImages/pkg/extractor"
	"helmPackageImages/pkg/manifest"
)

func TestExtractConfigMaps_HeuristicMode_TokensScanned(t *testing.T) {
	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  images.txt: |
    myregistry.io/app:v1.0.0
    myregistry.io/sidecar:v1.0.0
  readme.txt: "just plain text with no image refs"
`
	rules := []manifest.ConfigMapRule{{}}
	imgs, err := extractor.ExtractConfigMaps(parseObjects(t, yaml), rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := sorted([]string{"myregistry.io/app:v1.0.0", "myregistry.io/sidecar:v1.0.0"})
	if got := sorted(imgs); !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractConfigMaps_RegexMode_MatchesFound(t *testing.T) {
	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  config.properties: |
    image myregistry.io/app:v1.0.0
    port 8080
    sidecar myregistry.io/sidecar:v1.0.0
`
	rules := []manifest.ConfigMapRule{
		{Regex: `\S+/\S+:\S+`},
	}
	imgs, err := extractor.ExtractConfigMaps(parseObjects(t, yaml), rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := sorted([]string{"myregistry.io/app:v1.0.0", "myregistry.io/sidecar:v1.0.0"})
	if got := sorted(imgs); !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtractConfigMaps_JSONPathMode_ParsedAndExtracted(t *testing.T) {
	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  config.yaml: |
    image: myregistry.io/app:v1.0.0
    replicas: 3
`
	rules := []manifest.ConfigMapRule{
		{JSONPath: "{.image}"},
	}
	imgs, err := extractor.ExtractConfigMaps(parseObjects(t, yaml), rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "myregistry.io/app:v1.0.0" {
		t.Errorf("got %v, want [myregistry.io/app:v1.0.0]", imgs)
	}
}

func TestExtractConfigMaps_NamePattern_FiltersNonMatching(t *testing.T) {
	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  images.txt: "myregistry.io/app:v1.0.0"
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: other-config
data:
  images.txt: "myregistry.io/other:v1.0.0"
`
	rules := []manifest.ConfigMapRule{
		{NamePattern: `^app-`},
	}
	imgs, err := extractor.ExtractConfigMaps(parseObjects(t, yaml), rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "myregistry.io/app:v1.0.0" {
		t.Errorf("got %v, want only app-config image", imgs)
	}
}

func TestExtractConfigMaps_KeyPattern_FiltersNonMatching(t *testing.T) {
	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  images.yaml: "myregistry.io/app:v1.0.0"
  config.properties: "port=8080"
`
	rules := []manifest.ConfigMapRule{
		{KeyPattern: `\.yaml$`},
	}
	imgs, err := extractor.ExtractConfigMaps(parseObjects(t, yaml), rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "myregistry.io/app:v1.0.0" {
		t.Errorf("got %v, want only images.yaml content", imgs)
	}
}

func TestExtractConfigMaps_NonConfigMap_Skipped(t *testing.T) {
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
	rules := []manifest.ConfigMapRule{{}}
	imgs, err := extractor.ExtractConfigMaps(parseObjects(t, yaml), rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected no images from non-ConfigMap, got %v", imgs)
	}
}

func TestExtractConfigMaps_InvalidRegex_ReturnsError(t *testing.T) {
	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  key: value
`
	rules := []manifest.ConfigMapRule{{Regex: `[invalid`}}
	_, err := extractor.ExtractConfigMaps(parseObjects(t, yaml), rules)
	if err == nil {
		t.Error("expected error for invalid regex, got nil")
	}
}

func TestExtractConfigMaps_JSONPathNonObject_Skipped(t *testing.T) {
	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  plain.txt: "just a plain string"
`
	rules := []manifest.ConfigMapRule{{JSONPath: "{.image}"}}
	imgs, err := extractor.ExtractConfigMaps(parseObjects(t, yaml), rules)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected no images when value is not a YAML object, got %v", imgs)
	}
}
