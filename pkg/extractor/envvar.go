package extractor

import (
	"fmt"
	"regexp"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ExtractEnvVars scans workload container env vars whose names match any of the given
// regex patterns and collects their values as image references.
// ValueFrom entries (ConfigMap/Secret refs) are skipped since values aren't statically available.
func ExtractEnvVars(docs []runtime.Object, patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, nil
	}

	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		r, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid envVarPattern %q: %w", p, err)
		}
		compiled = append(compiled, r)
	}

	seen := map[string]struct{}{}
	for _, obj := range docs {
		for _, img := range envVarsFromResource(obj, compiled) {
			if img != "" {
				seen[img] = struct{}{}
			}
		}
	}
	return setToSlice(seen), nil
}

func envVarsFromResource(obj runtime.Object, patterns []*regexp.Regexp) []string {
	switch o := obj.(type) {
	case *appsv1.Deployment:
		return podSpecEnvVarImages(o.Spec.Template.Spec, patterns)
	case *appsv1.StatefulSet:
		return podSpecEnvVarImages(o.Spec.Template.Spec, patterns)
	case *appsv1.DaemonSet:
		return podSpecEnvVarImages(o.Spec.Template.Spec, patterns)
	case *batchv1.CronJob:
		return podSpecEnvVarImages(o.Spec.JobTemplate.Spec.Template.Spec, patterns)
	case *batchv1.Job:
		return podSpecEnvVarImages(o.Spec.Template.Spec, patterns)
	case *corev1.Pod:
		return podSpecEnvVarImages(o.Spec, patterns)
	}
	return nil
}

func podSpecEnvVarImages(spec corev1.PodSpec, patterns []*regexp.Regexp) []string {
	var imgs []string
	imgs = append(imgs, containerEnvVarImages(spec.Containers, patterns)...)
	imgs = append(imgs, containerEnvVarImages(spec.InitContainers, patterns)...)
	return imgs
}

func containerEnvVarImages(cs []corev1.Container, patterns []*regexp.Regexp) []string {
	var imgs []string
	for _, c := range cs {
		for _, env := range c.Env {
			if env.ValueFrom != nil {
				continue
			}
			if env.Value != "" && matchesAny(env.Name, patterns) {
				imgs = append(imgs, env.Value)
			}
		}
	}
	return imgs
}

func matchesAny(s string, patterns []*regexp.Regexp) bool {
	for _, r := range patterns {
		if r.MatchString(s) {
			return true
		}
	}
	return false
}
