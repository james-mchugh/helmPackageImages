package extractor_test

import (
	"sort"
	"testing"

	"helmPackageImages/pkg/extractor"
)

func sorted(s []string) []string {
	out := make([]string, len(s))
	copy(out, s)
	sort.Strings(out)
	return out
}

func TestBuiltin_Deployment(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
spec:
  template:
    spec:
      initContainers:
        - name: init
          image: busybox:1.36
      containers:
        - name: web
          image: nginx:1.25.3
`
	imgs, err := extractor.ExtractBuiltin(parseObjects(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := sorted([]string{"busybox:1.36", "nginx:1.25.3"})
	if got := sorted(imgs); !equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestBuiltin_StatefulSet(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: db
spec:
  template:
    spec:
      containers:
        - name: db
          image: postgres:15
`
	imgs, err := extractor.ExtractBuiltin(parseObjects(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "postgres:15" {
		t.Errorf("got %v", imgs)
	}
}

func TestBuiltin_DaemonSet(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: agent
spec:
  template:
    spec:
      containers:
        - name: agent
          image: datadog/agent:7
`
	imgs, err := extractor.ExtractBuiltin(parseObjects(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "datadog/agent:7" {
		t.Errorf("got %v", imgs)
	}
}

func TestBuiltin_Job(t *testing.T) {
	yaml := `
apiVersion: batch/v1
kind: Job
metadata:
  name: migrate
spec:
  template:
    spec:
      containers:
        - name: migrate
          image: myapp:migrate
`
	imgs, err := extractor.ExtractBuiltin(parseObjects(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "myapp:migrate" {
		t.Errorf("got %v", imgs)
	}
}

func TestBuiltin_CronJob(t *testing.T) {
	yaml := `
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cleanup
spec:
  jobTemplate:
    spec:
      template:
        spec:
          containers:
            - name: cleanup
              image: myapp:cleanup
`
	imgs, err := extractor.ExtractBuiltin(parseObjects(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "myapp:cleanup" {
		t.Errorf("got %v", imgs)
	}
}

func TestBuiltin_Pod(t *testing.T) {
	yaml := `
apiVersion: v1
kind: Pod
metadata:
  name: standalone
spec:
  containers:
    - name: app
      image: myapp:latest
`
	imgs, err := extractor.ExtractBuiltin(parseObjects(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 || imgs[0] != "myapp:latest" {
		t.Errorf("got %v", imgs)
	}
}

func TestBuiltin_NonWorkload_ReturnsEmpty(t *testing.T) {
	yaml := `
apiVersion: v1
kind: ConfigMap
metadata:
  name: config
data:
  key: value
`
	imgs, err := extractor.ExtractBuiltin(parseObjects(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 0 {
		t.Errorf("expected no images for ConfigMap, got %v", imgs)
	}
}

func TestBuiltin_Deduplication(t *testing.T) {
	yaml := `
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
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: b
spec:
  template:
    spec:
      containers:
        - name: b
          image: nginx:1.25.3
`
	imgs, err := extractor.ExtractBuiltin(parseObjects(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(imgs) != 1 {
		t.Errorf("expected deduplication to 1 image, got %v", imgs)
	}
}

func TestBuiltin_ReplicaSet_NotExtracted(t *testing.T) {
	yaml := `
apiVersion: apps/v1
kind: ReplicaSet
metadata:
  name: rs
spec:
  template:
    spec:
      containers:
        - name: app
          image: myapp:v1
`
	imgs, err := extractor.ExtractBuiltin(parseObjects(t, yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// ReplicaSet is not in the supported workload types; images should not be extracted.
	if len(imgs) != 0 {
		t.Errorf("expected no images for ReplicaSet, got %v", imgs)
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
