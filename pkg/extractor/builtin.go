package extractor

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ExtractBuiltin scans Kubernetes objects for images in known workload types.
// Duplicate images are removed before returning.
func ExtractBuiltin(objs []runtime.Object) ([]string, error) {
	seen := map[string]struct{}{}
	for _, obj := range objs {
		for _, img := range imagesFromResource(obj) {
			if img != "" {
				seen[img] = struct{}{}
			}
		}
	}
	return setToSlice(seen), nil
}

func imagesFromResource(obj runtime.Object) []string {
	var imgs []string
	switch o := obj.(type) {
	case *appsv1.Deployment:
		imgs = podSpecImages(o.Spec.Template.Spec)
	case *appsv1.StatefulSet:
		imgs = podSpecImages(o.Spec.Template.Spec)
	case *appsv1.DaemonSet:
		imgs = podSpecImages(o.Spec.Template.Spec)
	case *batchv1.CronJob:
		imgs = podSpecImages(o.Spec.JobTemplate.Spec.Template.Spec)
	case *batchv1.Job:
		imgs = podSpecImages(o.Spec.Template.Spec)
	case *corev1.Pod:
		imgs = podSpecImages(o.Spec)
	}
	return imgs
}

func podSpecImages(t corev1.PodSpec) []string {
	var imgs []string
	imgs = append(imgs, containerImages(t.Containers)...)
	imgs = append(imgs, containerImages(t.InitContainers)...)
	return imgs
}

func containerImages(cs []corev1.Container) []string {
	out := make([]string, 0, len(cs))
	for _, c := range cs {
		if c.Image != "" {
			out = append(out, c.Image)
		}
	}
	return out
}

func setToSlice(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
