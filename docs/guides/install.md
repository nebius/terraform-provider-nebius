---
page_title: "Nebius Provider: Install"
description: |-
  How to install and initialize the Nebius Terraform provider from the public Terraform Registry.
---

# Install the Nebius Provider

To use Nebius resources and data sources in Terraform, declare the provider in your working directory and run `terraform init`.

## Create a working directory

```bash
mkdir nebius-terraform-quickstart
cd nebius-terraform-quickstart
```

## Declare the provider

Create `terraform.tf`:

```hcl
terraform {
  required_providers {
    nebius = {
      source  = "nebius/nebius"
      version = ">= 0.6.8"
    }
  }
}
```

## Initialize the directory

```bash
terraform init
```

After initialization, you can add Nebius resources and data sources to the directory and apply them with Terraform.

## Move from the custom registry to the HashiCorp registry

If your project uses the provider from a Nebius custom registry, move it to the HashiCorp registry.

> **Note**
> The custom registry provider version `0.5.217` corresponds to the HashiCorp registry provider version `0.6.8`.

To move the provider:

1. Before changing the provider source, upgrade to the most recent custom registry provider version. Keep the custom registry source that the project already uses, update the version constraint and run `terraform init -upgrade`.

   For the current custom registry hostname:

   ```hcl
   terraform {
     required_providers {
       nebius = {
         source  = "terraform-provider.storage.eu-north1.nebius.cloud/nebius/nebius"
         version = ">= 0.5.217"
       }
     }
   }
   ```

   For the older custom registry hostname:

   ```hcl
   terraform {
     required_providers {
       nebius = {
         source  = "terraform-provider-nebius.storage.ai.nebius.cloud/nebius/nebius"
         version = ">= 0.5.217"
       }
     }
   }
   ```

   ```bash
   terraform init -upgrade
   ```

1. In `required_providers`, replace the custom registry source with the HashiCorp registry source and the corresponding most recent HashiCorp registry provider version:

   ```hcl
   terraform {
     required_providers {
       nebius = {
         source  = "nebius/nebius"
         version = ">= 0.6.8"
       }
     }
   }
   ```

   Use the `0.6` provider version that corresponds to the `0.5` version currently used in the project. For example, after upgrading to `0.5.217` in the custom registry, use `0.6.8` in the HashiCorp registry. Prefer an explicit constraint, such as `>= 0.6.8`.

1. From the root module or workspace that owns the Terraform state, replace the provider address in the state:

   ```bash
   terraform state replace-provider \
     terraform-provider.storage.eu-north1.nebius.cloud/nebius/nebius \
     registry.terraform.io/nebius/nebius
   ```

   If the project uses the older custom registry hostname, use it as the source provider address:

   ```bash
   terraform state replace-provider \
     terraform-provider-nebius.storage.ai.nebius.cloud/nebius/nebius \
     registry.terraform.io/nebius/nebius
   ```

   The `terraform state replace-provider` command updates the provider source address for all matching resources in the state and creates a state backup before saving changes.

   After this command, all resources in the state that used the custom registry provider are managed through the HashiCorp registry provider.

1. Refresh the provider version selection and lock file:

   ```bash
   terraform init -upgrade
   ```

   Expect changes in `.terraform.lock.hcl` because the provider source address and signing key are different. Terraform also prints the signing key fingerprint during initialization. Review and commit the updated `.terraform.lock.hcl` file.

1. Check that Terraform does not use the old provider addresses:

   ```bash
   terraform providers
   terraform plan
   ```

   The output must reference only `registry.terraform.io/nebius/nebius` and must not reference:

   * `terraform-provider.storage.eu-north1.nebius.cloud/nebius/nebius`
   * `terraform-provider-nebius.storage.ai.nebius.cloud/nebius/nebius`

   If old addresses are still present, update nested modules or module versions that still declare the old provider source.

## See also

* [Quickstart](quickstart.md)
* [Authentication](authentication.md)
