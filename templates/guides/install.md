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

## Move from the Nebius registry to the Terraform Registry

Before the provider was published in the [Terraform Registry](https://registry.terraform.io/providers/nebius/nebius/latest), Nebius distributed provider versions `0.5.x` through custom registries at these source addresses:

> **Note**
> Moving to the Terraform Registry does not affect the existing resources that your Terraform configuration manages.

* `terraform-provider.storage.eu-north1.nebius.cloud/nebius/nebius`
* `terraform-provider-nebius.storage.ai.nebius.cloud/nebius/nebius`

If your configuration uses one of these source addresses, migrate it to the corresponding `0.6.y` provider version in the Terraform Registry.

To check and migrate your configuration, perform these steps in the working directory:

1. Find `terraform.tf` or another file that contains the top-level `terraform` block.

1. In the `required_providers` block, find the `nebius` provider and check its `source` value:

   ```hcl
   terraform {
     required_providers {
       nebius = {
         source  = "terraform-provider.storage.eu-north1.nebius.cloud/nebius/nebius"
         version = ">= 0.5.210"
       }
     }
   }
   ```

   * If `source` is `terraform-provider.storage.eu-north1.nebius.cloud/nebius/nebius` or `terraform-provider-nebius.storage.ai.nebius.cloud/nebius/nebius`, continue with the migration.
   * If `source` is `nebius/nebius`, the configuration is already migrated.

1. Before changing the provider source, upgrade to the most recent `0.5.x` provider version. Keep the custom registry source that the project already uses, update the version constraint and run `terraform init -upgrade`.

   For example:

   ```hcl
   terraform {
     required_providers {
       nebius = {
         source  = "terraform-provider.storage.eu-north1.nebius.cloud/nebius/nebius"
         version = ">= 0.5.218"
       }
     }
   }
   ```

   ```bash
   terraform init -upgrade
   ```

1. Replace the custom registry source with the Terraform Registry source and upgrade from `0.5.x` to the corresponding `0.6.y` version:

   ```hcl
   terraform {
     required_providers {
       nebius = {
         source  = "nebius/nebius"
         version = ">= 0.6.9"
       }
     }
   }
   ```

   To find the corresponding `0.6.y` version, use this mapping:

   * `0.5.218` -> `0.6.9`
   * `0.5.217` -> `0.6.8`

   Although Terraform allows omitting the version constraint, prefer an explicit constraint such as `>= 0.6.9`.

1. From the root module or workspace that owns the Terraform state, replace the provider address in the state:

   ```bash
   terraform state replace-provider \
     terraform-provider.storage.eu-north1.nebius.cloud/nebius/nebius \
     registry.terraform.io/nebius/nebius
   ```

   If the project uses the older custom registry hostname, use that hostname as the source provider address:

   ```bash
   terraform state replace-provider \
     terraform-provider-nebius.storage.ai.nebius.cloud/nebius/nebius \
     registry.terraform.io/nebius/nebius
   ```

   The `terraform state replace-provider` commands create a state backup before saving changes.

   After this command, all resources in the state that used the custom registry provider are managed through the Terraform Registry provider.

1. Refresh the provider version selection and lock file:

   ```bash
   terraform init -upgrade
   ```

   Expect changes in `.terraform.lock.hcl` because the provider source address and signing key are different. Terraform also prints the signing key fingerprint during initialization.

1. Check that Terraform uses the new provider address:

   ```bash
   terraform providers
   terraform plan
   ```

   The output must reference only `registry.terraform.io/nebius/nebius`.

   If old addresses are still present, update nested modules or module versions that still declare the old provider source.

1. If your configuration is under version control, commit the changes that you made to it, including the updated `.terraform.lock.hcl` file.

## See also

* [Quickstart](quickstart.md)
* [Authentication](authentication.md)
