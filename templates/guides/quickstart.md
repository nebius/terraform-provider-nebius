---
page_title: "Nebius Provider: Quickstart"
description: |-
  A minimal end-to-end example for installing, authenticating, and applying the Nebius Terraform provider.
---

# Quickstart

This quickstart shows the smallest practical flow for creating a Nebius resource with Terraform.

## Before you start

- Install Terraform.
- Create a Nebius service account with the permissions needed for the resources you want to manage.
- Create an authorized key for that service account.
- Export the service-account credentials as environment variables.

For local development you can also authenticate with a user token or CLI profile, but service accounts are the recommended default.

## 1. Create the Terraform working directory

```bash
mkdir nebius-terraform-quickstart
cd nebius-terraform-quickstart
```

## 2. Add provider configuration

Create `terraform.tf`:

```hcl
terraform {
  required_providers {
    nebius = {
      source = "nebius/nebius"
    }
  }
}
```

Create `providers.tf`:

```hcl
provider "nebius" {
  service_account = {
    account_id_env       = "NB_SA_ID"
    public_key_id_env    = "NB_AUTHKEY_PUBLIC_ID"
    private_key_file_env = "NB_AUTHKEY_PRIVATE_PATH"
  }
}
```

## 3. Initialize Terraform

```bash
terraform init
```

## 4. Add a resource

Create `main.tf`:

```hcl
resource "nebius_registry_v1_registry" "example" {
  name        = "example-registry"
  parent_id   = "<project-id>"
  description = "Registry managed by Terraform"
}
```

Replace `<project-id>` with the target project ID.

## 5. Validate and apply

```bash
terraform validate
terraform apply
```

## Next steps

- Review [authentication options](authentication.md).
- Review [sensitive-value handling](sensitive-values.md) before managing secrets.
- Add version constraints for the provider before using it in shared environments.
