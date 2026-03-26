# Post-Open-Source Publishing Checklist

Use this checklist after the repository is made public.

- Confirm the public GitHub repository name is exactly `terraform-provider-nebius` and remains lowercase.
- Confirm the default branch contains the release scaffold from [`.goreleaser.yml`](.goreleaser.yml), [`.github/workflows/release.yml`](.github/workflows/release.yml), and [`terraform-registry-manifest.json`](terraform-registry-manifest.json).
- Create a dedicated GPG signing key for provider releases. The Terraform Registry requires signed releases.
- Add the repository secrets `GPG_PRIVATE_KEY` and `PASSPHRASE` used by the release workflow.
- Enable goreleaser in [`.github/workflows/release.yml`](.github/workflows/release.yml).
- Export the matching ASCII-armored public key and add it in Terraform Registry `User Settings -> Signing Keys` for the `nebius` namespace.
- Install Go locally, then run `make generate` to generate `docs/` from the provider schema and commit the generated docs.
- Run `make docs-validate` and fix any documentation frontmatter or rendering issues before the first public tag.
- Note: leave the current `tfplugindocs` wiring as-is until the full OSS release path is available. Before that point, local generation or validation may fail because the schema export path still depends on the final public-installation workflow.
- Review the generated provider docs, especially resource, data source, and ephemeral resource pages, because this provider surface is mostly code-generated.
- Add any missing resource-specific examples under `examples/resources/`, `examples/data-sources/`, and `examples/ephemeral-resources/` before the first public release if you want richer registry docs than the schema-only defaults.
- Commit the release version to [`provider/version/version.go`](/home/complynx/tf-public/provider/version/version.go#L9) on `main`. The release workflow reads that constant, creates the matching SemVer tag if it does not already exist, and only then publishes the release artifacts.
- If the matching tag already exists in the repository, the workflow skips the release to avoid publishing the same version twice.
- Verify the GitHub release is published and contains the expected assets for each target platform.
- Publish the provider from the Terraform Registry UI by selecting the public GitHub repository under the `nebius` namespace.
- Confirm the Registry-created webhook exists on the GitHub repository and that later releases sync automatically.
- Test installation from a clean machine with `terraform init` using `source = "nebius/nebius"` before announcing availability.
