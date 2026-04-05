package auth

import (
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/authn"
)

// Options controls credential resolution.
type Options struct{}

// Resolve returns a keychain for authenticating image pulls.
// Credential lookup order:
//  1. $DOCKER_CONFIG/config.json (or ~/.docker/config.json)
//  2. $HELM_REGISTRY_CONFIG (or ~/.config/helm/registry/config.json)
//
// If no credentials are found for a given registry, authn.Anonymous is used.
func Resolve(_ Options) (authn.Keychain, error) {
	chain := authn.NewMultiKeychain(
		&dirKeychain{configDir: dockerConfigDir()},
		&dirKeychain{configDir: helmConfigDir()},
	)
	return chain, nil
}

func dockerConfigDir() string {
	if d := os.Getenv("DOCKER_CONFIG"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".docker")
}

func helmConfigDir() string {
	if p := os.Getenv("HELM_REGISTRY_CONFIG"); p != "" {
		return filepath.Dir(p)
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "helm", "registry")
}

// dirKeychain resolves credentials from a Docker-format config.json in configDir
// by temporarily setting DOCKER_CONFIG and delegating to authn.DefaultKeychain.
type dirKeychain struct {
	configDir string
}

func (d *dirKeychain) Resolve(resource authn.Resource) (authn.Authenticator, error) {
	orig, origSet := os.LookupEnv("DOCKER_CONFIG")
	os.Setenv("DOCKER_CONFIG", d.configDir)
	defer func() {
		if origSet {
			os.Setenv("DOCKER_CONFIG", orig)
		} else {
			os.Unsetenv("DOCKER_CONFIG")
		}
	}()
	return authn.DefaultKeychain.Resolve(resource)
}
