# Nebius AI Cloud provider for Terraform

This repository contains the Nebius AI Cloud provider for Terraform source code.

See the [full documentation on installation, how-tos and usage here](https://docs.nebius.com/terraform-provider/). This readme explains basics of development and build process.

## Contributing

External contributions are appreciated. This repository is a public mirror, so pull requests may be closed without being merged directly here even when a change is later ported internally and published back to the mirror. See [CONTRIBUTING.md](CONTRIBUTING.md) for the workflow note.


## Build

Prerequisites:

- Go `1.26.1`
- `git`

Build the provider locally:

```bash
make build
```

### Debug

To start in debug mode, add the following variable to your debug env:

```bash
NEBIUS_TERRAFORM_PROVIDER_TEST=true
```

Then, copy the `TF_REATTACH_PROVIDERS` string from the output and run your Terraform with that string in its env.

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

**Notes:**

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
