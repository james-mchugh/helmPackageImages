package auth_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-containerregistry/pkg/authn"
	"helmPackageImages/pkg/auth"
)

type dockerConfig struct {
	Auths map[string]dockerAuth `json:"auths"`
}

type dockerAuth struct {
	Auth string `json:"auth"`
}

func writeDockerConfig(t *testing.T, dir string, cfg dockerConfig) string {
	t.Helper()
	data, _ := json.Marshal(cfg)
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("writeDockerConfig: %v", err)
	}
	return dir
}

func TestResolve_DockerConfigPresent(t *testing.T) {
	dir := t.TempDir()
	writeDockerConfig(t, dir, dockerConfig{
		Auths: map[string]dockerAuth{
			"registry.example.com": {Auth: "dXNlcjpwYXNz"}, // user:pass base64
		},
	})
	t.Setenv("DOCKER_CONFIG", dir)

	kc, err := auth.Resolve(auth.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if kc == nil {
		t.Fatal("expected non-nil keychain")
	}
	// Verify it can look up credentials (returns something, not Anonymous).
	resource := &fakeResource{registry: "registry.example.com"}
	authenticator, err := kc.Resolve(resource)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if authenticator == authn.Anonymous {
		t.Error("expected non-anonymous authenticator when docker config is present")
	}
}

func TestResolve_NeitherConfigPresent_ReturnsAnonymous(t *testing.T) {
	// Point both config paths to nonexistent dirs.
	t.Setenv("DOCKER_CONFIG", t.TempDir())
	t.Setenv("HELM_REGISTRY_CONFIG", filepath.Join(t.TempDir(), "config.json"))

	kc, err := auth.Resolve(auth.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resource := &fakeResource{registry: "registry.example.com"}
	authenticator, err := kc.Resolve(resource)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if authenticator != authn.Anonymous {
		t.Errorf("expected Anonymous authenticator, got %v", authenticator)
	}
}

func TestResolve_DockerConfigEnvRespected(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	// Write config only in dir2.
	writeDockerConfig(t, dir2, dockerConfig{
		Auths: map[string]dockerAuth{
			"myregistry.io": {Auth: "dXNlcjpwYXNz"},
		},
	})
	t.Setenv("DOCKER_CONFIG", dir1) // empty dir — config not here
	_ = dir2                        // not used by env, just proves env wins

	kc, err := auth.Resolve(auth.Options{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resource := &fakeResource{registry: "myregistry.io"}
	authenticator, err := kc.Resolve(resource)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// dir1 has no config, so should be anonymous.
	if authenticator != authn.Anonymous {
		t.Error("expected Anonymous when DOCKER_CONFIG points to empty dir")
	}
}

// fakeResource implements authn.Resource for testing.
type fakeResource struct{ registry string }

func (f *fakeResource) RegistryStr() string { return f.registry }
func (f *fakeResource) String() string      { return f.registry }
