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
- Remove this file.

### List Third-Party Dependencies:

Inspected on 2026-03-27.

Scope:
- The source tree does not vendor third-party source code.
- Release archives do distribute a statically linked provider binary, so runtime library licenses still matter.
- This list covers dependencies explicitly pinned in [`go.mod`](go.mod) plus explicit generator / release / CI tooling references found in [`tools/generate.go`](tools/generate.go), [`GNUmakefile`](GNUmakefile), and [`.github/workflows/release.yml`](.github/workflows/release.yml).
- `tools/go.mod` does not pin any additional tool versions.
- Unpinned `@latest` or major-tag references are called out as such instead of being treated as exact pins.

#### Special license notes

1. `MPL-2.0` is a file-level copyleft license. This repo uses several HashiCorp runtime modules under MPL-2.0, and those modules are compiled into the released provider binary. Preserve license / notice text, and if you modify MPL-covered files you must make those modified files available under MPL-2.0 when distributing the resulting work. This is a compliance note, not legal advice.
2. `BSL-1.1` is source-available rather than OSI open source. In this repo it only appears as an optional docs-generation dependency when `tfplugindocs` auto-downloads Terraform. It is not part of the provider runtime build unless you separately redistribute that CLI.
3. GitHub Actions pinned only to moving major tags such as `@v4` or `@v6` are not immutable supply-chain pins. If you need stricter provenance, pin them to a full commit SHA.

#### Runtime deps

* buf.build/go/protovalidate v1.1.3 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/buf.build/go/protovalidate@v1.1.3)
* github.com/blang/semver/v4 v4.0.0 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/blang/semver/v4@v4.0.0)
* github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/github.com/grpc-ecosystem/go-grpc-middleware/v2@v2.3.3)
* github.com/hashicorp/terraform-plugin-framework v1.19.0 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework@v1.19.0)
* github.com/hashicorp/terraform-plugin-framework-jsontypes v0.2.0 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework-jsontypes@v0.2.0)
* github.com/hashicorp/terraform-plugin-framework-validators v0.19.0 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-framework-validators@v0.19.0)
* github.com/hashicorp/terraform-plugin-go v0.31.0 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-go@v0.31.0)
* github.com/hashicorp/terraform-plugin-log v0.10.0 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/terraform-plugin-log@v0.10.0)
* github.com/nebius/gosdk v0.2.8 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/nebius/gosdk@v0.2.8)
* github.com/osteele/liquid v1.8.1 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/osteele/liquid@v1.8.1)
* google.golang.org/grpc v1.79.3 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/google.golang.org/grpc@v1.79.3)
* google.golang.org/protobuf v1.36.11 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/google.golang.org/protobuf@v1.36.11)

#### Runtime transitive deps

* buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go v1.36.11-20260209202127-80ab13bee0bf.1 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/buf.build/gen/go/bufbuild/protovalidate/protocolbuffers/go@v1.36.11-20260209202127-80ab13bee0bf.1)
* cel.dev/expr v0.25.1 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/cel.dev/expr@v0.25.1)
* github.com/antlr4-go/antlr/v4 v4.13.1 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/github.com/antlr4-go/antlr/v4@v4.13.1)
* github.com/cenkalti/backoff/v4 v4.3.0 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/cenkalti/backoff/v4@v4.3.0)
* github.com/fatih/color v1.19.0 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/fatih/color@v1.19.0)
* github.com/gofrs/flock v0.13.0 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/github.com/gofrs/flock@v0.13.0)
* github.com/golang-jwt/jwt/v4 v4.5.2 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/golang-jwt/jwt/v4@v4.5.2)
* github.com/golang/protobuf v1.5.4 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/github.com/golang/protobuf@v1.5.4)
* github.com/google/cel-go v0.27.0 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/github.com/google/cel-go@v0.27.0)
* github.com/google/uuid v1.6.0 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/github.com/google/uuid@v1.6.0)
* github.com/hashicorp/go-hclog v1.6.3 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/go-hclog@v1.6.3)
* github.com/hashicorp/go-plugin v1.7.0 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/go-plugin@v1.7.0)
* github.com/hashicorp/go-uuid v1.0.3 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/go-uuid@v1.0.3)
* github.com/hashicorp/terraform-registry-address v0.4.0 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/terraform-registry-address@v0.4.0)
* github.com/hashicorp/terraform-svchost v0.2.1 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/terraform-svchost@v0.2.1)
* github.com/hashicorp/yamux v0.1.2 MPL-2.0. Special; see Note 1. [pkg.go.dev](https://pkg.go.dev/github.com/hashicorp/yamux@v0.1.2)
* github.com/mattn/go-colorable v0.1.14 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/mattn/go-colorable@v0.1.14)
* github.com/mattn/go-isatty v0.0.20 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/mattn/go-isatty@v0.0.20)
* github.com/mitchellh/go-testing-interface v1.14.1 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/mitchellh/go-testing-interface@v1.14.1)
* github.com/oklog/run v1.2.0 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/github.com/oklog/run@v1.2.0)
* github.com/osteele/tuesday v1.0.4 MIT. [pkg.go.dev](https://pkg.go.dev/github.com/osteele/tuesday@v1.0.4)
* github.com/vmihailenco/msgpack/v5 v5.4.1 BSD-2-Clause. [pkg.go.dev](https://pkg.go.dev/github.com/vmihailenco/msgpack/v5@v5.4.1)
* github.com/vmihailenco/tagparser/v2 v2.0.0 BSD-2-Clause. [pkg.go.dev](https://pkg.go.dev/github.com/vmihailenco/tagparser/v2@v2.0.0)
* golang.org/x/exp v0.0.0-20260312153236-7ab1446f8b90 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/golang.org/x/exp@v0.0.0-20260312153236-7ab1446f8b90)
* golang.org/x/mod v0.34.0 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/golang.org/x/mod@v0.34.0)
* golang.org/x/net v0.52.0 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/golang.org/x/net@v0.52.0)
* golang.org/x/sync v0.20.0 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/golang.org/x/sync@v0.20.0)
* golang.org/x/sys v0.42.0 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/golang.org/x/sys@v0.42.0)
* golang.org/x/text v0.35.0 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/golang.org/x/text@v0.35.0)
* golang.org/x/tools v0.43.0 BSD-3-Clause. [pkg.go.dev](https://pkg.go.dev/golang.org/x/tools@v0.43.0)
* google.golang.org/genproto/googleapis/api v0.0.0-20260319201613-d00831a3d3e7 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/google.golang.org/genproto/googleapis/api@v0.0.0-20260319201613-d00831a3d3e7)
* google.golang.org/genproto/googleapis/rpc v0.0.0-20260319201613-d00831a3d3e7 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/google.golang.org/genproto/googleapis/rpc@v0.0.0-20260319201613-d00831a3d3e7)
* gopkg.in/yaml.v2 v2.4.0 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/gopkg.in/yaml.v2@v2.4.0)
* gopkg.in/yaml.v3 v3.0.1 Apache-2.0. [pkg.go.dev](https://pkg.go.dev/gopkg.in/yaml.v3@v3.0.1)

#### Dev deps

* None explicitly pinned outside the Go module graph above.

#### Generator / docs deps

* github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs `@latest` MPL-2.0. Special; see Note 1. Not version-pinned in this repo; referenced from `tools/generate.go` and `GNUmakefile`. [GitHub](https://github.com/hashicorp/terraform-plugin-docs)
* hashicorp/terraform `latest` when auto-downloaded by `tfplugindocs` BSL-1.1. Special; see Note 2. Not version-pinned in this repo; only needed for docs generation when Terraform is absent locally. [GitHub](https://github.com/hashicorp/terraform)

#### Release / CI deps

* actions/checkout `@v4` MIT. Moving major tag, not immutable; see Note 3. Referenced from `.github/workflows/release.yml`. [GitHub](https://github.com/actions/checkout)
* actions/setup-go `@v5` MIT. Moving major tag, not immutable; see Note 3. Referenced from `.github/workflows/release.yml`. [GitHub](https://github.com/actions/setup-go)
* crazy-max/ghaction-import-gpg `@v6` MIT. Moving major tag, not immutable; see Note 3. Referenced from `.github/workflows/release.yml`. [GitHub](https://github.com/crazy-max/ghaction-import-gpg)
* goreleaser/goreleaser-action `@v6` MIT. Moving major tag, not immutable; see Note 3. Referenced from `.github/workflows/release.yml`. [GitHub](https://github.com/goreleaser/goreleaser-action)
* goreleaser CLI `latest` MIT. Not version-pinned in this repo: `GNUmakefile` calls local `goreleaser`, and the commented workflow step sets `version: latest`. [GitHub](https://github.com/goreleaser/goreleaser)
