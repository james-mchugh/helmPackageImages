package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"helmPackageImages/pkg/archiver"
	"helmPackageImages/pkg/auth"
	"helmPackageImages/pkg/extractor"
	helmrender "helmPackageImages/pkg/helm"
	"helmPackageImages/pkg/manifest"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

type options struct {
	manifestPath string
	profile      string
	output       string
	format       string
	platform     string
	dryRun       bool
	setValues    []string
	scrapeValues bool
	verbose      bool
}

func newRootCmd() *cobra.Command {
	var opt options

	cmd := &cobra.Command{
		Use:   "helm package-images [chart...]",
		Short: "Package container images from a Helm chart into an OCI archive",
		Long: `Renders one or more Helm charts, discovers all container image references,
pulls them, and packages them into an OCI-compatible tar archive for transfer
to air-gapped environments.

Remote charts (e.g. stable/nginx, oci://registry/chart) require the repository
to be registered via 'helm repo add' first.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(args, opt)
		},
	}

	flags := cmd.Flags()
	flags.StringVarP(&opt.manifestPath, "manifest", "m", "", "Path to airgap.yaml (default: <chart-root>/airgap.yaml)")
	flags.StringVarP(&opt.profile, "profile", "p", "", "Profile name to activate")
	flags.StringVarP(&opt.output, "output", "o", "", "Output archive path (required when multiple charts given)")
	flags.StringVar(
		&opt.platform,
		"platform",
		"",
		"Comma-separated platforms, e.g. linux/amd64,linux/arm64 (default: current system)",
	)
	flags.StringVar(
		&opt.format,
		"format",
		"oci",
		`Output archive format: "oci" (OCI Image Layout) or "docker" (Docker tarball, loadable via docker load)`,
	)
	flags.BoolVar(&opt.dryRun, "dry-run", false, "List discovered images without pulling or archiving")
	flags.StringArrayVar(&opt.setValues, "set", nil, "Helm value overrides (may be repeated)")
	flags.BoolVar(&opt.scrapeValues, "scrape-values", false, "Naively scan values.yaml for image-like strings")
	flags.BoolVarP(&opt.verbose, "verbose", "v", false, "Print detailed progress to stderr")

	return cmd
}

func run(charts []string, opt options) error {
	if len(charts) > 1 && opt.output == "" && !opt.dryRun {
		return fmt.Errorf("--output is required when multiple charts are specified")
	}
	if opt.format != "oci" && opt.format != "docker" {
		return fmt.Errorf("--format must be \"oci\" or \"docker\", got %q", opt.format)
	}

	// logf always prints to stderr; verbf only prints when --verbose is set.
	logf := func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
	verbf := func(format string, args ...any) {
		if opt.verbose {
			fmt.Fprintf(os.Stderr, format+"\n", args...)
		}
	}

	// Resolve CLI-level bool overrides (tri-state: unset = nil).
	var overrideScrapeValues *bool
	if opt.scrapeValues {
		t := true
		overrideScrapeValues = &t
	}

	seen := map[string]struct{}{}
	var firstName string

	total := len(charts)
	for i, ref := range charts {
		if total > 1 {
			logf("[%d/%d] Processing chart %q...", i+1, total, ref)
		} else {
			logf("Processing chart %q...", ref)
		}

		name, imgs, err := processChart(ref, opt, overrideScrapeValues, verbf)
		if err != nil {
			return fmt.Errorf("chart %q: %w", ref, err)
		}
		if firstName == "" {
			firstName = name
		}
		verbf("  Found %d image(s) in chart %q", len(imgs), name)
		for _, img := range imgs {
			seen[img] = struct{}{}
		}
	}

	var allImages []string
	for img := range seen {
		allImages = append(allImages, img)
	}

	if opt.dryRun {
		logf("Discovered %d unique image(s):", len(allImages))
		for _, img := range allImages {
			fmt.Println(img)
		}
		return nil
	}

	// Determine output path.
	outPath := opt.output
	if outPath == "" {
		// Single chart — use the chart's own name from Chart.yaml.
		outPath = firstName + ".tar"
	}

	// Resolve auth keychain.
	verbf("Resolving registry credentials...")
	kc, err := auth.Resolve(auth.Options{})
	if err != nil {
		return fmt.Errorf("resolving credentials: %w", err)
	}

	// Pull images.
	platform := opt.platform
	if platform == "" {
		platform = "linux/" + runtime.GOARCH
	}
	logf("Pulling %d unique image(s) for platform %s...", len(allImages), platform)
	pulled, err := archiver.Pull(
		allImages, archiver.PullOptions{
			Platforms: platform,
			Keychain:  kc,
			ProgressFunc: func(current, total int, ref string) {
				logf("  [%d/%d] %s", current, total, ref)
			},
		},
	)
	if err != nil {
		return fmt.Errorf("pulling images: %w", err)
	}

	// Write archive.
	verbf("Writing archive to %q (format: %s)...", outPath, opt.format)
	if err := archiver.Write(outPath, pulled, archiver.WriteOptions{Format: archiver.Format(opt.format)}); err != nil {
		return fmt.Errorf("writing archive: %w", err)
	}
	fmt.Printf("Archive written to %s (%d images)\n", outPath, len(pulled))
	return nil
}

func processChart(
	ref string,
	opt options,
	overrideScrapeValues *bool,
	verbf func(string, ...any),
) (string, []string, error) {
	// Fetch chart (local path, OCI, or HTTP repo).
	// Version is specified inline in the ref: stable/nginx:1.2.3 or oci://reg/chart:tag.
	verbf("  Fetching chart %q...", ref)
	chrt, err := helmrender.Fetch(ref)
	if err != nil {
		return "", nil, fmt.Errorf("fetching chart: %w", err)
	}

	// Load manifest — ChartRoot is only meaningful for local/HTTP repo charts.
	// For OCI in-memory charts, chartRoot is empty and --manifest must be used explicitly.
	verbf("  Loading manifest for %q...", chrt.Name())
	m, err := manifest.Load(
		manifest.Options{
			ManifestPath:         opt.manifestPath,
			Profile:              opt.profile,
			Chart:                chrt,
			OverridePlatform:     opt.platform,
			OverrideScrapeValues: overrideScrapeValues,
		},
	)
	if err != nil {
		return "", nil, fmt.Errorf("loading manifest: %w", err)
	}

	// Render chart.
	verbf("  Rendering templates for %q...", chrt.Name())
	docs, err := helmrender.Render(
		helmrender.RenderOptions{
			Chart:     chrt,
			Values:    m.Values,
			SetValues: opt.setValues,
		},
	)
	if err != nil {
		return "", nil, fmt.Errorf("rendering chart: %w", err)
	}

	// Extract images.
	verbf("  Extracting image references from %q...", chrt.Name())
	imgs, err := extractor.Extract(docs, m)
	return chrt.Name(), imgs, err
}
