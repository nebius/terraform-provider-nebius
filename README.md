# Terraform Provider for Nebius

This repository contains the Terraform provider for Nebius AI Cloud.

The repository already follows the public Terraform Registry naming convention for providers (`terraform-provider-nebius`), and the release assets in this repository are prepared for publication to the public Terraform Registry under the `nebius/nebius` source address.

The remaining blockers are operational rather than structural: the GitHub repository must be public, signed releases must be enabled, and the provider must be published from the Terraform Registry UI. The exact follow-up checklist is in [PUBLISHING.md](PUBLISHING.md).

## Build

Prerequisites:

- Go `1.26.1`
- `git`

Build the provider locally:

```bash
make build
```

Install the provider binary into your Go bin directory:

```bash
make install
```

## Documentation Generation

Terraform Registry documentation is generated from the provider schema, templates, and example configurations.

Generate docs:

```bash
make generate
```

Validate generated docs against Terraform Registry rules:

```bash
make docs-validate
```

Notes:

- `tfplugindocs` is invoked from the `tools/` helper module.
- If Terraform is not already installed locally, `tfplugindocs` may download it automatically during documentation generation.
- Generated documentation is written to `docs/`.

## Example Provider Configuration

Minimal token-based configuration:

```terraform
terraform {
  required_providers {
    nebius = {
      source = "nebius/nebius"
    }
  }
}

provider "nebius" {
  token = var.nebius_iam_token
}
```

Additional examples live in [`examples/provider/`](examples/provider).

## Release Process

Signed multi-platform release assets are configured through:

- [`.goreleaser.yml`](.goreleaser.yml)
- [`.github/workflows/release.yml`](.github/workflows/release.yml)
- [`terraform-registry-manifest.json`](terraform-registry-manifest.json)

Pushing a commit to `main` with a new value in [`provider/version/version.go`](provider/version/version.go) triggers the release workflow. The workflow creates the matching `vX.Y.Z` tag if it does not already exist, publishes the release artifacts from that tag, and skips the release if the tag is already present.
