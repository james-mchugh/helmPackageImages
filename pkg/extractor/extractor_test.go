package extractor_test

import (
	"sort"
	"testing"

	"helmPackageImages/pkg/extractor"
	"helmPackageImages/pkg/manifest"
)

func TestExtract_AllSources(t *testing.T) {
	docs := []string{`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  template:
    spec:
      containers:
        - name: web
          image: nginx:1.25.3
---
apiVersion: mygroup.io/v1alpha1
kind: MyOperator
metadata:
  name: op
spec:
  image: example.com/op:v1
`}
	values := map[string]interface{}{
		"image": map[string]interface{}{
			"repository": "redis",
			"tag":        "7.2",
		},
	}
	m := &manifest.Manifest{
		CRDs: []manifest.CRDEntry{
			{Kind: "MyOperator", APIVersion: "mygroup.io/v1alpha1", ImagePaths: []string{".spec.image"}},
		},
		Values: values,
		Settings: manifest.Settings{
			ScrapeValues: true,
		},
	}
	imgs, err := extractor.Extract(docs, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"example.com/op:v1", "nginx:1.25.3", "redis:7.2"}
	got := imgs
	sort.Strings(got)
	if !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestExtract_Deduplication(t *testing.T) {
	docs := []string{`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: a
spec:
  template:
    spec:
      containers:
        - name: a
          image: nginx:1.25.3
`}
	values := map[string]interface{}{
		"image": "nginx:1.25.3",
	}
	m := &manifest.Manifest{
		Values:   values,
		Settings: manifest.Settings{ScrapeValues: true},
	}
	imgs, err := extractor.Extract(docs, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "nginx:1.25.3" {
		t.Errorf("expected deduplication to 1 image, got %v", imgs)
	}
}

func TestExtract_ScrapeValuesDisabled(t *testing.T) {
	docs := []string{}
	values := map[string]interface{}{
		"image": "nginx:1.25.3",
	}
	m := &manifest.Manifest{
		Values:   values,
		Settings: manifest.Settings{ScrapeValues: false},
	}
	imgs, err := extractor.Extract(docs, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected no images when scrapeValues disabled, got %v", imgs)
	}
}

func TestExtract_NoCRDs_SkipsCustomExtractor(t *testing.T) {
	docs := []string{`
apiVersion: mygroup.io/v1alpha1
kind: MyOperator
spec:
  image: example.com/op:v1
`}
	m := &manifest.Manifest{
		CRDs: nil,
	}
	imgs, err := extractor.Extract(docs, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected no images without CRD entries, got %v", imgs)
	}
}

func TestExtract_ResultIsSorted(t *testing.T) {
	docs := []string{`
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  template:
    spec:
      containers:
        - name: b
          image: zz-image:1
        - name: a
          image: aa-image:1
`}
	m := &manifest.Manifest{}
	imgs, err := extractor.Extract(docs, m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !sort.StringsAreSorted(imgs) {
		t.Errorf("expected sorted result, got %v", imgs)
	}
}
