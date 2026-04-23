package extractor

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/name"
)

// ValidateImages checks every image reference using the go-containerregistry parser.
// Invalid references are always logged via logf and removed from the returned slice.
// When strict is true, an error is returned listing all invalid references.
func ValidateImages(images []string, strict bool, logf func(string, ...any)) ([]string, error) {
	valid := make([]string, 0, len(images))
	var invalid []string

	for _, img := range images {
		if _, err := name.ParseReference(img, name.WeakValidation); err != nil {
			logf("warning: skipping invalid image reference %q: %v", img, err)
			invalid = append(invalid, img)
		} else {
			valid = append(valid, img)
		}
	}

	if strict && len(invalid) > 0 {
		return nil, fmt.Errorf("invalid image references: %s", strings.Join(invalid, ", "))
	}
	return valid, nil
}
