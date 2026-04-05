package archiver

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/google/go-containerregistry/pkg/name"
	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/empty"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	"github.com/google/go-containerregistry/pkg/v1/tarball"
)

// Format specifies the output archive format.
type Format string

const (
	// FormatOCI writes an OCI Image Layout tar archive (default).
	FormatOCI Format = "oci"
	// FormatDocker writes a Docker-compatible tarball loadable via "docker load".
	FormatDocker Format = "docker"
)

// WriteOptions controls how the archive is written.
type WriteOptions struct {
	Format Format // default (zero value) is FormatOCI
}

// Write packages images (already fetched as v1.Image values) into a tar archive
// at outPath. The map key is the image reference string. Format is controlled by opts.
func Write(outPath string, images map[string]v1.Image, opts WriteOptions) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}
	switch opts.Format {
	case FormatDocker:
		return writeDockerTar(outPath, images)
	default: // FormatOCI or ""
		return writeOCILayout(outPath, images)
	}
}

// writeOCILayout writes images as an OCI Image Layout tar archive.
func writeOCILayout(outPath string, images map[string]v1.Image) error {
	// Build OCI layout in a temp directory.
	tmpDir, err := os.MkdirTemp("", "helm-package-images-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	lp, err := layout.Write(tmpDir, empty.Index)
	if err != nil {
		return fmt.Errorf("initialising OCI layout: %w", err)
	}

	for ref, img := range images {
		tag, err := name.NewTag(ref, name.WeakValidation)
		if err != nil {
			return fmt.Errorf("parsing image reference %q: %w", ref, err)
		}
		if err := lp.AppendImage(img, layout.WithAnnotations(map[string]string{
			"org.opencontainers.image.ref.name": tag.String(),
		})); err != nil {
			return fmt.Errorf("appending image %q: %w", ref, err)
		}
	}

	// Tar the OCI layout directory.
	return tarDir(tmpDir, outPath)
}

// writeDockerTar writes images as a Docker-compatible tarball (docker load format).
func writeDockerTar(outPath string, images map[string]v1.Image) error {
	refs := make(map[name.Reference]v1.Image, len(images))
	for ref, img := range images {
		r, err := name.ParseReference(ref, name.WeakValidation)
		if err != nil {
			return fmt.Errorf("parsing reference %q: %w", ref, err)
		}
		refs[r] = img
	}
	return tarball.MultiRefWriteToFile(outPath, refs)
}

// tarDir creates a tar archive of srcDir at dstPath.
func tarDir(srcDir, dstPath string) error {
	out, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("creating output file: %w", err)
	}
	defer out.Close()

	tw := tar.NewWriter(out)
	defer tw.Close()

	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = rel

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
}
