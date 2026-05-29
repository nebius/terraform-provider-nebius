# Support

This document explains how to get help with the Nebius Terraform provider and
what information helps maintainers triage issues quickly.

## Where to Get Help

* Documentation: start with the [Terraform provider documentation](https://docs.nebius.com/terraform-provider/),
  including authentication, sensitive values, provider configuration, state
  storage, resource reference and release notes.
* Public bugs and feature requests: open a GitHub issue in this repository.
* Security vulnerabilities: follow [SECURITY.md](SECURITY.md) and use a private
  GitHub Security Advisory report.
* Nebius customers with production impact: create a ticket in the
  [Nebius support center](https://docs.nebius.com/overview/support). Choose the
  priority that matches the business impact. If the support center is not
  available, use the fallback email listed in the Nebius support documentation.

GitHub issues are suitable for provider bugs, documentation gaps, feature
requests and reproducible Terraform behavior. Nebius support tickets are the
right path for account-specific failures, quota or billing questions, paid
customer escalation, production outages and service-side incidents.

## Issue Triage

Provider maintainers review public issues on a regular triage cycle, normally
at least weekly. Urgent security and production-impact reports should not wait
for that cycle; use the security advisory or Nebius support path above.

Triage usually follows these paths:

* Provider defect: the maintainers reproduce the Terraform behavior, identify
  the affected provider versions and queue a fix or workaround.
* Nebius service or API behavior: the issue is routed to the relevant Nebius
  service team. A provider change is made only if the provider can safely
  improve validation, diagnostics, retry behavior or documentation.
* Documentation or example issue: the maintainers update public docs, examples
  or generated provider reference where appropriate.
* Usage or configuration question: the issue may be answered directly, moved to
  Nebius support if it is account-specific or closed if it cannot be
  reproduced with enough detail.

Helpful issue details:

* Terraform version and Nebius provider version.
* Operating system and architecture.
* The exact command that failed, for example `terraform plan` or
  `terraform apply`.
* Minimal redacted Terraform configuration.
* Redacted diagnostic output and relevant `TF_LOG=DEBUG` excerpts.
* Resource IDs, project IDs, tenant IDs and request IDs when available.

Do not include secrets, IAM tokens, private keys, service-account credentials,
full Terraform state files or unredacted logs in public issues.

## Paid Customer Support

Paid customers should use Nebius support for urgent or account-specific
problems. Support tickets let Nebius apply the priority, response-time and
account-team escalation processes described in the Nebius support documentation.

For Terraform provider issues that also have a public GitHub issue, include the
GitHub issue link in the support ticket. For support tickets that reveal a
provider bug, Nebius support may escalate the case to the provider maintainers
and affected service owners.

## Urgent Patch Scenarios

Nebius may publish an out-of-cycle patch release for:

* Confirmed vulnerabilities or release-artifact integrity issues.
* Regressions that prevent safe create, update, read, import or destroy
  operations for production resources.
* Provider behavior that can unexpectedly expose credentials in Terraform
  state, logs, diagnostics or generated configuration.
* Compatibility breaks caused by Nebius public API changes.
* Terraform Registry publishing or installation failures affecting the latest
  supported release.

Urgent patches follow the same release pipeline as regular releases: versioned
source, changelog entry, signed multi-platform release assets, checksums and
Terraform Registry manifest.

## Release Cadence

Provider release numbers use Semantic Versioning syntax. While the provider is
pre-1.0, changelog entries call out notable improvements, new resources and
data sources, deprecations and breaking changes.

Regular releases are published when provider fixes or Nebius public API/schema
changes are ready. While the provider is under active development, the target
cadence is at least monthly when user-visible changes exist. Releases may be
more frequent when generated resources change or when important fixes are
available and may be skipped when there are no user-visible changes.

Each release is expected to include:

* A `vX.Y.Z` Git tag matching `provider/version/version.go`.
* Changelog or release-note content for the version.
* Signed checksums and multi-platform archives for Terraform Registry
  ingestion.
* The `terraform-registry-manifest.json` release artifact.

## Credentials

Use service-account authentication for CI, automation and shared Terraform
workflows. User tokens and CLI profiles are better suited to local development.
Prefer environment variables or credential files managed outside source
control and grant only the permissions needed by the Terraform configuration.

For details, see the [authentication guide](docs/guides/authentication.md) and
the provider [configuration reference](https://docs.nebius.com/terraform-provider/reference/provider).

## Terraform State and Secrets

Terraform state can contain resource attributes and other sensitive
infrastructure data. Store shared state in a protected remote backend, restrict
access to the state bucket and enable locking where available. See the Nebius
guide for [storing Terraform state in Object Storage](https://docs.nebius.com/object-storage/store-terraform-state).

For secrets, prefer provider `sensitive` blocks, Terraform ephemeral values
and the `nebius_hash` helper pattern so raw secret values do not need to be
stored in state. See the [sensitive values guide](docs/guides/sensitive-values.md).

## Logs and Diagnostics

Terraform diagnostics are the main source of provider validation errors,
service errors and warnings. Server-side warnings may appear during `plan` or
`apply` without failing the operation.

When maintainers or Nebius support ask for debug data, run with
`TF_LOG=DEBUG` only long enough to reproduce the issue. The provider redacts
the credentials and sensitive fields it handles before emitting logs, including
debug logs. Still review logs before sharing them: secrets can appear if they
were placed in non-sensitive Terraform fields, user-controlled variables,
external command output, shell traces, or other data outside provider-controlled
redaction.

## Resource Import

Remote Nebius resources generated from public APIs generally support Terraform
import by resource ID unless the resource documentation says otherwise:

```bash
terraform import nebius_compute_v1_disk.example <resource-id>
```

With Terraform 1.5 or later, you can also use an import block and ask Terraform
to generate initial configuration:

```hcl
import {
  to = nebius_compute_v1_disk.example
  id = "<resource-id>"
}
```

```bash
terraform plan -generate-config-out=generated.tf
```

Review generated configuration before applying it. Import cannot recover
write-only or input-only values that the Nebius API does not return and those
values may need to be added manually through the `sensitive` object or other
configuration.

## Deprecation Policy

The provider follows Nebius public API deprecations. Deprecated fields,
resources, data sources or enum values may be marked in generated reference
documentation and surfaced as Terraform diagnostics or warnings during
`plan`/`apply`.

When a replacement exists, release notes or diagnostics should point to it.
Before provider version 1.0, breaking changes can still appear in minor or
patch releases when public APIs change or unsafe behavior must be removed; the
changelog calls these out under breaking changes. After 1.0, incompatible
provider-level changes should normally be reserved for major versions unless a
security or service-compatibility issue requires faster removal.
