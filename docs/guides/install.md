---
page_title: "Nebius Provider: Install"
description: |-
  How to install and initialize the Nebius Terraform provider from the public Terraform Registry.
---

# Install the Nebius Provider

To use Nebius resources and data sources in Terraform, declare the provider in your working directory and run `terraform init`.

## 1. Create a working directory

```bash
mkdir nebius-terraform
cd nebius-terraform
```

## 2. Declare the provider

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

If you want reproducible builds, add an explicit version constraint once you have selected a released provider version.

## 3. Configure authentication

For automation, prefer service-account authentication:

```hcl
provider "nebius" {
  service_account = {
    account_id_env       = "NB_SA_ID"
    public_key_id_env    = "NB_AUTHKEY_PUBLIC_ID"
    private_key_file_env = "NB_AUTHKEY_PRIVATE_PATH"
  }
}
```

See the [authentication guide](authentication.md) for other supported methods.

## 4. Initialize the directory

```bash
terraform init
```

After initialization, you can add Nebius resources and data sources to the directory and apply them with Terraform.
