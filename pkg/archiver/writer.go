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
)

// Write pulls all images (already fetched as v1.Image values) into a single
// OCI image layout and tars the result to outPath.
// The map key is the image reference string used to annotate the image in the layout.
func Write(outPath string, images map[string]v1.Image) error {
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return fmt.Errorf("creating output directory: %w", err)
	}

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
