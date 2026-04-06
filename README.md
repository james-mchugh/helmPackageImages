# helm package-images

A Helm plugin that discovers all container images referenced by a chart and bundles them into a portable archive for
air-gapped environments.

## Overview

Deploying to a disconnected or air-gapped environment requires shipping every container image alongside the chart.
`helm package-images` automates this: it renders your chart with your values, discovers every image reference across all
templates and subcharts, pulls those images, and writes them to a single tar archive — ready to copy across the network
boundary and load into a private registry.

## Installation

```bash
helm plugin install https://github.com/james-mchugh/helmPackageImages
```

Pre-built binaries are downloaded automatically for your OS and architecture. If no pre-built binary is available, the
plugin falls back to building from source (requires Go).

## Usage

### Package a local chart

```bash
helm package-images ./my-chart
# Writes: my-chart.tar  (OCI Image Layout format)
```

### Specify the output path

```bash
helm package-images ./my-chart -o images.tar
```

### Docker format (loadable via `docker load`)

```bash
helm package-images ./my-chart --format docker -o images.tar
docker load -i images.tar
```

### Multi-platform images

```bash
helm package-images ./my-chart --platform linux/amd64,linux/arm64
```

### Remote chart from an HTTP repository

```bash
helm repo add bitnami https://charts.bitnami.com/bitnami
helm package-images bitnami/nginx
```

### Remote chart from an OCI registry

```bash
helm package-images oci://registry-1.docker.io/bitnamicharts/nginx
```

### Dry run — list images without pulling

```bash
helm package-images ./my-chart --dry-run
```

### Override Helm values

```bash
helm package-images ./my-chart --set image.tag=v2.0 --set global.registry=myregistry.io
```

### Activate a profile

Profiles let you define environment-specific settings in `airgap.yaml` (see below).

```bash
helm package-images ./my-chart --profile production
```

## `airgap.yaml` configuration

Place an `airgap.yaml` file at the chart root (or point to one with `--manifest`) to configure discovery behavior
without repeating flags on every invocation.

```yaml
# airgap.yaml

# Helm values merged over chart defaults before rendering.
values:
  image:
    registry: my-registry.example.com

settings:
  # Target platforms for multi-arch image pulling.
  platform: linux/amd64,linux/arm64

  # Include subchart (dependency) images. Default: true.
  includeChartDependencies: true

  # Heuristically scan values.yaml for image-like strings.
  # Useful for charts that construct image refs outside of standard fields.
  scrapeValues: false

# Image paths for custom resources (CRDs) deployed by the chart.
crds:
  - kind: MyApp
    apiVersion: example.com/v1
    imagePaths:
      - "{.spec.image}"
      - "{.spec.sidecar.image}"

# Named profiles — override base settings for specific environments.
profiles:
  production:
    settings:
      platform: linux/amd64
    values:
      replicaCount: 3
```

## Flags

| Flag              | Default                    | Description                                                         |
|-------------------|----------------------------|---------------------------------------------------------------------|
| `-m, --manifest`  | `<chart-root>/airgap.yaml` | Path to airgap.yaml                                                 |
| `-p, --profile`   | —                          | Profile name to activate                                            |
| `-o, --output`    | `<chart-name>.tar`         | Output archive path                   |
| `--format`        | `oci`                      | Output format: `oci` (OCI Image Layout) or `docker` (Docker tarball) |
| `--platform`      | current system             | Comma-separated platforms, e.g. `linux/amd64,linux/arm64`           |
| `--dry-run`       | `false`                    | Print discovered image references without pulling                   |
| `--set`           | —                          | Helm value overrides, e.g. `--set image.tag=v2` (repeatable)        |
| `--scrape-values` | `false`                    | Heuristically scan `values.yaml` for image-like strings             |

## Transferring to an air-gapped environment

Copy the archive to the target environment, then use a registry tool to push the images into your internal registry:

```bash
# OCI format — push with oras
oras copy --from-oci-layout images.tar --to myregistry.internal

# Docker format — load into the local Docker daemon, then push
docker load -i images.tar
docker push myregistry.internal/nginx:latest
```
