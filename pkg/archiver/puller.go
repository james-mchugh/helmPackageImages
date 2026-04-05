package archiver

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/google/go-containerregistry/pkg/v1/types"
)

// PullOptions controls image pulling.
type PullOptions struct {
	// Platforms is a comma-separated list of os/arch strings (e.g. "linux/amd64,linux/arm64").
	// When empty, the image is pulled as-is (no platform filtering).
	Platforms string

	// Keychain is used to authenticate with registries.
	Keychain authn.Keychain
}

// Pull fetches all images for each reference × platform combination.
// Returns a map from "ref@platform" (or just "ref" for single-platform) to v1.Image.
func Pull(refs []string, opt PullOptions) (map[string]v1.Image, error) {
	platforms := parsePlatforms(opt.Platforms)
	kc := opt.Keychain
	if kc == nil {
		kc = authn.DefaultKeychain
	}

	result := make(map[string]v1.Image)
	for _, ref := range refs {
		tag, err := name.ParseReference(ref, name.WeakValidation)
		if err != nil {
			return nil, fmt.Errorf("parsing reference %q: %w", ref, err)
		}

		desc, err := remote.Get(tag, remote.WithAuthFromKeychain(kc))
		if err != nil {
			return nil, fmt.Errorf("fetching %q: %w", ref, err)
		}

		switch desc.MediaType {
		case types.OCIImageIndex, types.DockerManifestList:
			// Multi-arch index — pull each requested platform.
			idx, err := desc.ImageIndex()
			if err != nil {
				return nil, fmt.Errorf("image index for %q: %w", ref, err)
			}
			for _, p := range platforms {
				if p == "" {
					// No platform filter specified; skip multi-arch images to avoid
					// matching against an empty OS string.
					continue
				}
				img, err := imageForPlatform(idx, p)
				if err != nil {
					return nil, fmt.Errorf("platform %s for %q: %w", p, ref, err)
				}
				key := ref + "-" + strings.ReplaceAll(p, "/", "-")
				result[key] = img
			}
		default:
			// Single-arch image.
			img, err := desc.Image()
			if err != nil {
				return nil, fmt.Errorf("image for %q: %w", ref, err)
			}
			result[ref] = img
		}
	}
	return result, nil
}

func parsePlatforms(s string) []string {
	if s == "" {
		return []string{""}
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func imageForPlatform(idx v1.ImageIndex, platform string) (v1.Image, error) {
	manifest, err := idx.IndexManifest()
	if err != nil {
		return nil, err
	}
	parts := strings.SplitN(platform, "/", 2)
	os, arch := parts[0], ""
	if len(parts) == 2 {
		arch = parts[1]
	}
	for _, m := range manifest.Manifests {
		if m.Platform == nil {
			continue
		}
		if m.Platform.OS == os && (arch == "" || m.Platform.Architecture == arch) {
			return idx.Image(m.Digest)
		}
	}
	return nil, fmt.Errorf("no manifest found for platform %q", platform)
}
