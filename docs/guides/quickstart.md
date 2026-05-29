---
page_title: "Nebius Provider: Quickstart"
description: |-
  A minimal end-to-end example for installing, authenticating and applying the Nebius Terraform provider.
---

# Quickstart

Nebius AI Cloud provides several interfaces to manage cloud resources. In addition to the web console and CLI commands, you can use the Terraform provider by Nebius AI Cloud.

Terraform is most useful when you need to create and maintain multiple resources simultaneously. The example below shows how to get started with the provider by creating a Container Registry registry.

## Install required tools

Before you start:

* Install Terraform.
* Install `jq`.
* Install and configure the Nebius AI Cloud CLI.

The CLI and `jq` are only required for the setup commands in this example.

## Configure access and credentials

In this example, Terraform applies configurations on behalf of a Nebius AI Cloud service account. Alternatively, you can authenticate with your user account. For details, see [authentication](authentication.md).

To configure access for the service account:

1. Create a service account and save its ID to an environment variable:

   ```bash
   export SA_ID=$(nebius iam service-account create \
     --name terraform-sa --format json \
     | jq -r '.metadata.id')
   ```

1. Grant edit access to the service account:

   1. Get the tenant ID from the web console or with the Nebius AI Cloud CLI.

   1. Get the ID of the default `editors` group:

      ```bash
      export EDITORS_GROUP_ID=$(nebius iam group get-by-name \
        --name editors --parent-id <tenant_ID> --format json \
        | jq -r '.metadata.id')
      ```

      If Terraform must manage users and group memberships, use the `admins` group instead of `editors`.

   1. Add the service account to the group:

      ```bash
      nebius iam group-membership create \
        --parent-id $EDITORS_GROUP_ID \
        --member-id $SA_ID
      ```

1. Create an authorized key:

   1. Generate a key pair:

      ```bash
      mkdir -p ~/.nebius/authkey
      export AUTHKEY_PRIVATE_PATH=~/.nebius/authkey/private.pem
      export AUTHKEY_PUBLIC_PATH=~/.nebius/authkey/public.pem
      openssl genrsa -out $AUTHKEY_PRIVATE_PATH 4096
      openssl rsa -in $AUTHKEY_PRIVATE_PATH \
        -outform PEM -pubout -out $AUTHKEY_PUBLIC_PATH
      ```

   1. Upload the public key to create the authorized key and save its ID to an environment variable:

      ```bash
      export AUTHKEY_PUBLIC_ID=$(nebius iam auth-public-key create \
        --account-service-account-id $SA_ID \
        --data "$(cat $AUTHKEY_PUBLIC_PATH)" \
        --format json | jq -r '.metadata.id')
      ```

## Initialize a working directory

The configuration files for each infrastructure that you deploy with Terraform should be in their own working directory. This is where you run the Terraform CLI commands.

1. Create the working directory:

   ```bash
   mkdir nebius-terraform-quickstart
   cd nebius-terraform-quickstart
   ```

1. Create `terraform.tf`:

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

1. Create `providers.tf`:

   ```hcl
   provider "nebius" {
     service_account = {
       private_key_file_env = "AUTHKEY_PRIVATE_PATH"
       public_key_id_env    = "AUTHKEY_PUBLIC_ID"
       account_id_env       = "SA_ID"
     }
   }
   ```

1. Initialize Terraform:

   ```bash
   terraform init
   ```

## Create resources

After your working directory is initialized, define and build your infrastructure:

1. Create `main.tf`:

   ```hcl
   resource "nebius_registry_v1_registry" "my-registry" {
     name        = "my-registry"
     parent_id   = "<project_ID>"
     description = "My registry"
   }
   ```

   `parent_id` is the target project ID.

1. Validate the configuration:

   ```bash
   terraform validate
   ```

1. If the configuration is valid, apply it:

   ```bash
   terraform apply
   ```

## See also

* [Install the provider](install.md)
* [Authentication](authentication.md)
* [Working with sensitive values](sensitive-values.md)
