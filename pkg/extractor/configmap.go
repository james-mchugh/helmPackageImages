package extractor

import (
	"fmt"
	"regexp"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"helmPackageImages/pkg/manifest"
)

type compiledCMRule struct {
	nameRe  *regexp.Regexp // nil = match all
	keyRe   *regexp.Regexp // nil = match all
	valueRe *regexp.Regexp // nil = heuristic (when JSONPath also empty)
	rule    manifest.ConfigMapRule
}

// ExtractConfigMaps scans ConfigMap data values for image references according to the
// given rules. Each rule may specify name/key filters and an extraction mode:
// JSONPath (parse value as YAML/JSON), regex (scan raw string), or heuristic (looksLikeImage).
func ExtractConfigMaps(docs []runtime.Object, rules []manifest.ConfigMapRule) ([]string, error) {
	if len(rules) == 0 {
		return nil, nil
	}

	compiled, err := compileCMRules(rules)
	if err != nil {
		return nil, err
	}

	seen := map[string]struct{}{}
	for _, obj := range docs {
		cm, ok := obj.(*corev1.ConfigMap)
		if !ok {
			continue
		}
		for _, cr := range compiled {
			if cr.nameRe != nil && !cr.nameRe.MatchString(cm.Name) {
				continue
			}
			for key, value := range cm.Data {
				if cr.keyRe != nil && !cr.keyRe.MatchString(key) {
					continue
				}
				imgs, err := extractFromValue(value, cr)
				if err != nil {
					return nil, fmt.Errorf("configmap %q key %q: %w", cm.Name, key, err)
				}
				for _, img := range imgs {
					if img != "" {
						seen[img] = struct{}{}
					}
				}
			}
		}
	}
	return setToSlice(seen), nil
}

func compileCMRules(rules []manifest.ConfigMapRule) ([]compiledCMRule, error) {
	compiled := make([]compiledCMRule, 0, len(rules))
	for _, r := range rules {
		cr := compiledCMRule{rule: r}
		var err error
		if r.NamePattern != "" {
			if cr.nameRe, err = regexp.Compile(r.NamePattern); err != nil {
				return nil, fmt.Errorf("invalid namePattern %q: %w", r.NamePattern, err)
			}
		}
		if r.KeyPattern != "" {
			if cr.keyRe, err = regexp.Compile(r.KeyPattern); err != nil {
				return nil, fmt.Errorf("invalid keyPattern %q: %w", r.KeyPattern, err)
			}
		}
		if r.Regex != "" {
			if cr.valueRe, err = regexp.Compile(r.Regex); err != nil {
				return nil, fmt.Errorf("invalid regex %q: %w", r.Regex, err)
			}
		}
		compiled = append(compiled, cr)
	}
	return compiled, nil
}

func extractFromValue(value string, cr compiledCMRule) ([]string, error) {
	switch {
	case cr.rule.JSONPath != "":
		var parsed interface{}
		if err := yaml.Unmarshal([]byte(value), &parsed); err != nil {
			return nil, fmt.Errorf("parsing as YAML for JSONPath: %w", err)
		}
		m, ok := parsed.(map[string]interface{})
		if !ok {
			return nil, nil
		}
		return extractPaths(m, cr.rule.JSONPath)

	case cr.valueRe != nil:
		return cr.valueRe.FindAllString(value, -1), nil

	default:
		var imgs []string
		for _, token := range strings.Fields(value) {
			if looksLikeImage(token) {
				imgs = append(imgs, token)
			}
		}
		return imgs, nil
	}
}
