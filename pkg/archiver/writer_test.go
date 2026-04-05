package archiver_test

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/random"
	"helmPackageImages/pkg/archiver"
)

// makeImage creates a random in-memory OCI image for testing.
func makeImage(t *testing.T) v1.Image {
	t.Helper()
	img, err := random.Image(128, 1)
	if err != nil {
		t.Fatalf("random.Image: %v", err)
	}
	return img
}

func TestWrite_SingleImage_ValidOCITar(t *testing.T) {
	img := makeImage(t)
	outPath := filepath.Join(t.TempDir(), "images.tar")

	if err := archiver.Write(outPath, map[string]v1.Image{"nginx:1.25.3": img}); err != nil {
		t.Fatalf("Write: %v", err)
	}

	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}

	// Verify the tar contains OCI layout files.
	entries := tarEntries(t, outPath)
	if !containsPrefix(entries, "oci-layout") {
		t.Errorf("expected oci-layout in tar, got %v", entries)
	}
	if !containsPrefix(entries, "index.json") {
		t.Errorf("expected index.json in tar, got %v", entries)
	}
}

func TestWrite_MultipleImages_AllPresent(t *testing.T) {
	images := map[string]v1.Image{
		"nginx:1.25.3": makeImage(t),
		"redis:7.2":    makeImage(t),
		"busybox:1.36": makeImage(t),
	}
	outPath := filepath.Join(t.TempDir(), "combined.tar")

	if err := archiver.Write(outPath, images); err != nil {
		t.Fatalf("Write: %v", err)
	}

	entries := tarEntries(t, outPath)
	if !containsPrefix(entries, "index.json") {
		t.Errorf("expected index.json in combined tar, got %v", entries)
	}
	// All blobs should be present.
	blobCount := 0
	for _, e := range entries {
		if strings.HasPrefix(e, "blobs/") {
			blobCount++
		}
	}
	if blobCount == 0 {
		t.Errorf("expected blob entries in tar, got %v", entries)
	}
}

func TestWrite_CreatesOutputAtPath(t *testing.T) {
	img := makeImage(t)
	dir := t.TempDir()
	outPath := filepath.Join(dir, "subdir", "images.tar")
	// subdir does not exist yet — Write should create it.
	if err := archiver.Write(outPath, map[string]v1.Image{"myapp:v1": img}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("output file not created: %v", err)
	}
}

// tarEntries returns the list of file names inside a tar archive.
func tarEntries(t *testing.T, path string) []string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open tar: %v", err)
	}
	defer f.Close()

	var entries []string
	tr := tar.NewReader(f)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar.Next: %v", err)
		}
		entries = append(entries, hdr.Name)
	}
	return entries
}

func containsPrefix(entries []string, prefix string) bool {
	for _, e := range entries {
		if e == prefix || strings.HasPrefix(e, prefix) {
			return true
		}
	}
	return false
}
