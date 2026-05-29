---
page_title: "Nebius Provider: Store Terraform State"
description: |-
  How to store Terraform state for Nebius configurations in an Object Storage bucket.
---

# Store Terraform State in Object Storage

When you create Nebius AI Cloud infrastructure with Terraform, Terraform stores the current infrastructure state in a `.tfstate` file. For shared environments or automated workflows, store the state in a protected remote backend instead of a local file.

Nebius Object Storage is S3-compatible, so it can be used with Terraform's `s3` backend.

## Prerequisites

Before you configure the backend:

* Create an Object Storage bucket for the state.
* Create a service account that can access the bucket.
* Create an access key for that service account and save the AWS-like ID and secret.
* Install and configure the Nebius provider.

Terraform recommends enabling bucket versioning for state buckets. Versioning can increase storage costs.

## Configure the backend

Create `terraform.tf`:

```hcl
terraform {
  required_providers {
    nebius = {
      source  = "nebius/nebius"
      version = ">= 0.6.8"
    }
  }

  backend "s3" {
    bucket = "<bucket_name>"
    key    = "<path_to_store_tfstate>"
    region = "eu-north1"

    access_key = "<AWS-like_key_ID>"
    secret_key = "<AWS-like_key_secret>"

    endpoints = {
      s3 = "https://storage.eu-north1.nebius.cloud"
    }

    use_lockfile = true

    skip_credentials_validation = true
    skip_region_validation      = true
    skip_requesting_account_id  = true
    skip_metadata_api_check     = true
  }
}
```

In this file:

* `bucket`: Bucket name.
* `key`: Path where Terraform stores the state, relative to the bucket root.
* `access_key` and `secret_key`: AWS-like ID and secret used to access the bucket.
* `use_lockfile`: Enables state locking to prevent concurrent modifications.

Then initialize and apply the configuration:

```bash
terraform init
terraform validate
terraform apply
```

After that, Terraform pulls the latest remote state from the bucket whenever you run Terraform commands in this working directory.

For the full Nebius guide, see [How to store a Terraform state in an Object Storage bucket](https://docs.nebius.com/object-storage/store-terraform-state).
