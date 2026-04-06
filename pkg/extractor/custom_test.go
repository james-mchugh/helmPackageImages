package extractor_test

import (
	"testing"

	"helmPackageImages/pkg/extractor"
	"helmPackageImages/pkg/manifest"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
)

func TestCustom_MatchingCR(t *testing.T) {
	yaml := `
apiVersion: mygroup.io/v1alpha1
kind: MyOperator
metadata:
  name: example
spec:
  image: example.com/myapp:v2.0.0
  sidecarImage: example.com/sidecar:v1.1.0
`
	crds := []manifest.CRDEntry{
		{
			TypeMeta:   k8sruntime.TypeMeta{Kind: "MyOperator", APIVersion: "mygroup.io/v1alpha1"},
			ImagePaths: []string{"{.spec.image}", "{.spec.sidecarImage}"},
		},
	}
	imgs, err := extractor.ExtractCustom(parseObjects(t, yaml), crds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := sorted([]string{"example.com/myapp:v2.0.0", "example.com/sidecar:v1.1.0"})
	if got := sorted(imgs); !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCustom_NonMatchingAPIVersion_Skipped(t *testing.T) {
	yaml := `
apiVersion: mygroup.io/v1beta1
kind: MyOperator
metadata:
  name: example
spec:
  image: example.com/myapp:v2.0.0
`
	crds := []manifest.CRDEntry{
		{
			TypeMeta:   k8sruntime.TypeMeta{Kind: "MyOperator", APIVersion: "mygroup.io/v1alpha1"},
			ImagePaths: []string{"{.spec.image}"},
		},
	}
	imgs, err := extractor.ExtractCustom(parseObjects(t, yaml), crds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected no images for non-matching apiVersion, got %v", imgs)
	}
}

func TestCustom_MissingField_SkippedGracefully(t *testing.T) {
	yaml := `
apiVersion: mygroup.io/v1alpha1
kind: MyOperator
metadata:
  name: example
spec:
  name: no-image-here
`
	crds := []manifest.CRDEntry{
		{
			TypeMeta:   k8sruntime.TypeMeta{Kind: "MyOperator", APIVersion: "mygroup.io/v1alpha1"},
			ImagePaths: []string{"{.spec.image}"},
		},
	}
	imgs, err := extractor.ExtractCustom(parseObjects(t, yaml), crds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected no images for missing field, got %v", imgs)
	}
}

func TestCustom_MultipleCRDEntries(t *testing.T) {
	yaml := `
apiVersion: mygroup.io/v1alpha1
kind: MyOperator
metadata:
  name: op
spec:
  image: example.com/op:v1
---
apiVersion: other.io/v1
kind: OtherResource
metadata:
  name: other
spec:
  containerImage: example.com/other:v2
`
	crds := []manifest.CRDEntry{
		{
			TypeMeta:   k8sruntime.TypeMeta{Kind: "MyOperator", APIVersion: "mygroup.io/v1alpha1"},
			ImagePaths: []string{"{.spec.image}"},
		},
		{
			TypeMeta:   k8sruntime.TypeMeta{Kind: "OtherResource", APIVersion: "other.io/v1"},
			ImagePaths: []string{"{.spec.containerImage}"},
		},
	}
	imgs, err := extractor.ExtractCustom(parseObjects(t, yaml), crds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := sorted([]string{"example.com/op:v1", "example.com/other:v2"})
	if got := sorted(imgs); !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestCustom_EmptyCRDs_ReturnsEmpty(t *testing.T) {
	yaml := `
apiVersion: mygroup.io/v1alpha1
kind: MyOperator
spec:
  image: example.com/myapp:v1
`
	imgs, err := extractor.ExtractCustom(parseObjects(t, yaml), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected no images with no CRD entries, got %v", imgs)
	}
}
