---
page_title: "Nebius Provider: Authentication"
description: |-
  Authentication patterns supported by the Nebius Terraform provider.
---

# Authentication

To manage Nebius AI Cloud resources with Terraform, provide credentials through the provider configuration. You can authenticate as either a service account or a user account.

Service accounts are the recommended choice for CI, automation and shared Terraform workflows. If both service-account credentials and user-account credentials are present in a Terraform configuration, the provider authenticates as the user account.

## Authenticate with a service account

Use a service account when Terraform is running outside an interactive user session.

Before using a service account in the provider:

1. Create a service account if you have not already.
1. Add the service account to a group with the permissions needed for the resources you want to manage. In most cases, a group with the `editor` role is enough. Use the `admin` role only if Terraform must manage other accounts' group memberships.
1. Create an authorized key for the service account.

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
    account_id_env       = "SA_ID"
    public_key_id_env    = "AUTHKEY_ID"
    private_key_file_env = "AUTHKEY_PRIV_PATH"
  }
}
```

```bash
export SA_ID=serviceaccount-e00a0b1c**********
export AUTHKEY_ID=publickey-e00z9y8x**********
export AUTHKEY_PRIV_PATH=~/.nebius/authkey/private.pem
```

The following credentials are required in `service_account`:

* Service account ID, in `account_id` or `account_id_env`. You can get it with:

  ```bash
  nebius iam service-account get-by-name \
    --name <service_account_name> \
    --format json | jq -r '.metadata.id'
  ```

  Alternatively, use `nebius iam service-account list` and get the ID from `.items[*].metadata.id`.

* Authorized key ID, in `public_key_id` or `public_key_id_env`. You can list authorized keys created for the service account with:

  ```bash
  nebius iam auth-public-key list-by-account \
    --account-service-account-id <service_account_ID> \
    --format json
  ```

* Path to the private key that you used to create the authorized key, in `private_key_file` or `private_key_file_env`.

The provider also supports `credentials_file` and `credentials_file_env` if you prefer to point at a credentials file instead of specifying individual fields.

## Authenticate with a user account

Use user-account authentication mainly for local development. User account authentication uses access tokens. The lifetime of an access token is 12 hours.

To get an access token:

1. Install and configure the Nebius AI Cloud CLI.
1. Run:

   ```bash
   nebius iam get-access-token
   ```

You can specify the token in the `NEBIUS_IAM_TOKEN` environment variable:

```bash
NEBIUS_IAM_TOKEN=<access_token> terraform apply
```

Or in the provider configuration:

```hcl
provider "nebius" {
  token = "<access_token>"
}
```

## See also

* [Quickstart](quickstart.md)
* [Provider configuration](../index.md#schema)
