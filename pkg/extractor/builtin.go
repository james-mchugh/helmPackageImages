package extractor

import (
	"strings"

	"sigs.k8s.io/yaml"
)

// resource is a minimal representation of a Kubernetes manifest for GVK detection.
type resource struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Spec       struct {
		// Pod-level containers (kind: Pod).
		Containers          []container `json:"containers"`
		InitContainers      []container `json:"initContainers"`
		EphemeralContainers []container `json:"ephemeralContainers"`
		// Workload wrapper (Deployment, StatefulSet, DaemonSet, Job, ReplicaSet).
		Template *podTemplateSpec `json:"template"`
		// CronJob wrapper.
		JobTemplate *struct {
			Spec struct {
				Template podTemplateSpec `json:"template"`
			} `json:"spec"`
		} `json:"jobTemplate"`
	} `json:"spec"`
}

type podTemplateSpec struct {
	Spec struct {
		Containers          []container `json:"containers"`
		InitContainers      []container `json:"initContainers"`
		EphemeralContainers []container `json:"ephemeralContainers"`
	} `json:"spec"`
}

type container struct {
	Image string `json:"image"`
}

// builtinKinds lists GVK combinations (kind, apiVersion prefix) that contain pod specs.
var builtinKinds = map[string]bool{
	"Deployment":  true,
	"StatefulSet": true,
	"DaemonSet":   true,
	"Job":         true,
	"CronJob":     true,
	"ReplicaSet":  true,
	"Pod":         true,
}

// ExtractBuiltin scans rendered YAML documents for images in known workload types.
// Duplicate images are removed before returning.
func ExtractBuiltin(docs []string) ([]string, error) {
	seen := map[string]struct{}{}
	for _, doc := range docs {
		// A single doc string may contain multiple YAML documents separated by "---".
		for _, part := range splitYAMLDocs(doc) {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			var r resource
			if err := yaml.Unmarshal([]byte(part), &r); err != nil {
				continue // skip non-YAML fragments
			}
			if !builtinKinds[r.Kind] {
				continue
			}
			for _, img := range imagesFromResource(r) {
				if img != "" {
					seen[img] = struct{}{}
				}
			}
		}
	}
	return setToSlice(seen), nil
}

func imagesFromResource(r resource) []string {
	var imgs []string
	switch r.Kind {
	case "Pod":
		imgs = append(imgs, containerImages(r.Spec.Containers)...)
		imgs = append(imgs, containerImages(r.Spec.InitContainers)...)
		imgs = append(imgs, containerImages(r.Spec.EphemeralContainers)...)
	case "CronJob":
		if r.Spec.JobTemplate != nil {
			imgs = append(imgs, podSpecImages(r.Spec.JobTemplate.Spec.Template)...)
		}
	default:
		if r.Spec.Template != nil {
			imgs = append(imgs, podSpecImages(*r.Spec.Template)...)
		}
	}
	return imgs
}

func podSpecImages(t podTemplateSpec) []string {
	var imgs []string
	imgs = append(imgs, containerImages(t.Spec.Containers)...)
	imgs = append(imgs, containerImages(t.Spec.InitContainers)...)
	imgs = append(imgs, containerImages(t.Spec.EphemeralContainers)...)
	return imgs
}

func containerImages(cs []container) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		if c.Image != "" {
			out = append(out, c.Image)
		}
	}
	return out
}

// splitYAMLDocs splits a string on "---" document separators.
func splitYAMLDocs(s string) []string {
	return strings.Split(s, "\n---")
}

func setToSlice(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
