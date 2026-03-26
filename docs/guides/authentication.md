---
page_title: "Nebius Provider: Authentication"
description: |-
  Authentication patterns supported by the Nebius Terraform provider.
---

# Authentication

To manage Nebius resources with Terraform, provide credentials through the provider configuration. You can authenticate as either a service account or a user account.

If both a service-account configuration and a user token are present, the token-based user authentication takes precedence.

Service accounts are the recommended choice for CI, automation, and shared Terraform workflows.

## Service Account Authentication

Use a service account when Terraform is running outside an interactive user session.

You can specify service-account credentials directly:

```hcl
provider "nebius" {
  service_account = {
    account_id       = "serviceaccount-e00a0b1c**********"
    public_key_id    = "publickey-e00z9y8x**********"
    private_key_file = "~/.nebius/authkey/private.pem"
  }
}
```

Or indirectly through environment variables:

```hcl
provider "nebius" {
  service_account = {
    account_id_env       = "NB_SA_ID"
    public_key_id_env    = "NB_AUTHKEY_PUBLIC_ID"
    private_key_file_env = "NB_AUTHKEY_PRIVATE_PATH"
  }
}
```

```bash
export NB_SA_ID=serviceaccount-e00a0b1c**********
export NB_AUTHKEY_PUBLIC_ID=publickey-e00z9y8x**********
export NB_AUTHKEY_PRIVATE_PATH=~/.nebius/authkey/private.pem
```

The provider also supports `credentials_file` and `credentials_file_env` if you prefer to point at a credentials file instead of specifying individual fields.

## User Account Authentication

Use user-account authentication mainly for local development.

### IAM Token

Use an IAM token when you already have a token issuance flow outside Terraform.

```terraform
provider "nebius" {
  token = var.nebius_iam_token
}
```

You can also set the token through the `NEBIUS_IAM_TOKEN` environment variable.

### CLI Profile

Use a Nebius CLI profile when local operators already authenticate through the Nebius CLI and want Terraform to reuse that context:

```terraform
provider "nebius" {
  profile = {
    name = "default"
  }
}
```

The profile block reads Nebius CLI configuration and is most useful on developer machines.

## Choosing a Method

- Prefer service accounts for CI and production automation.
- Prefer CLI profiles for local development.
- Prefer short-lived IAM tokens over long-lived credentials when integrating with external secret management.
