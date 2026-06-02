# Security Policy

## Supported Versions

Security fixes are normally released for the latest published minor series of
the Nebius Terraform provider. Older minor series may receive fixes only when
Nebius determines that the risk and upgrade impact require it.

| Version | Security support |
| ------- | ---------------- |
| Latest minor series | Supported |
| Older minor series | Best effort |

## Reporting a Vulnerability

The Nebius team takes security bugs seriously. We appreciate responsible
disclosure and will make every effort to acknowledge your report.

To report a security issue, please use the GitHub Security Advisory [Report a Vulnerability](https://github.com/nebius/terraform-provider-nebius/security/advisories/new) tab.

Do not open a public GitHub issue for suspected vulnerabilities. Do not include
IAM tokens, private keys, service-account credentials, Terraform state files,
or unredacted debug logs in the report. If those details are needed, the
maintainers will arrange a safer exchange path.

## Response and Escalation

The Terraform provider is an open-source integration and does not have a
standalone service-availability SLA. Availability commitments for Nebius cloud
services are governed by the Nebius AI Cloud [Service Level Agreement](https://docs.nebius.com/legal/sla)
and the service-specific terms that apply to the affected service.

Security reports are triaged by provider maintainers and routed to Nebius teams
when they may affect Nebius customer data, credentials, service availability
or released provider binary integrity.

Expected handling:

* Initial acknowledgement: normally within two business days.
* Triage: determine affected provider versions, exploitability, severity,
  workarounds and whether Nebius cloud services or customer credentials are
  involved.
* Escalation: high-impact reports are escalated to Nebius security, support
  and affected service owners. Paid customers with an active production
  impact should also create a Nebius support ticket so the documented
  [support priorities and response-time process](https://docs.nebius.com/overview/support)
  and account-team escalation can be applied.
* Fix: confirmed security issues are fixed through a non-public maintainer
  workflow when embargo is required, covered by focused tests where practical,
  then released as a patch as soon as the fix is ready.
* Disclosure: Nebius publishes a GitHub Security Advisory, release note, CVE
  or other customer notice when appropriate. Public details may be delayed
  until a fixed provider version is available.

Urgent patch releases may be used for credential exposure, provider behavior
that can write secrets to state or logs unexpectedly, release-artifact
integrity problems or severe regressions that block safe management of
production resources.

For non-security support and production-impact cases, see [SUPPORT.md](SUPPORT.md).

## Learning More About Security in Nebius

To learn more about security in Nebius, see [Security in Nebius AI Cloud](https://docs.nebius.com/security).
