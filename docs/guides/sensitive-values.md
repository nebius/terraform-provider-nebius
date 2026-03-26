---
page_title: "Nebius Provider: Sensitive Values"
description: |-
  Guidance for handling secrets with write-only values, ephemeral values, and the nebius_hash helper resource.
---

# Working With Sensitive Values

Terraform stores infrastructure state in a `.tfstate` file. Because that file can contain sensitive information, the Nebius provider supports write-only values and ephemeral values so secrets do not need to be stored in state.

## Write-Only Arguments

In the Nebius provider, write-only arguments are stored inside a resource's `sensitive` object. Each write-only argument usually has a corresponding first-level managed argument.

For example:

```hcl
resource "nebius_msp_mlflow_v1alpha1_cluster" "example" {
  name           = "example"
  parent_id      = "your-project-id"
  admin_username = "user"
  admin_password = "password"

  sensitive = {
    admin_password = "password"
  }
}
```

Set only one of the two forms. Prefer the value inside `sensitive` when you do not want the secret stored in Terraform state.

## Tracking Updates to Write-Only Values

Terraform cannot detect changes to write-only values during planning because they are intentionally excluded from state. To signal that a write-only value changed, update `sensitive.version`.

```hcl
resource "nebius_msp_mlflow_v1alpha1_cluster" "example" {
  name           = "example"
  parent_id      = "your-project-id"
  admin_username = "user"

  sensitive = {
    version        = "2"
    admin_password = "password"
  }
}
```

If a resource is updated and the required write-only value is omitted, Terraform will prompt for it again during `terraform apply`.

## Ephemeral Variables

Terraform ephemeral variables can feed write-only arguments without storing the values in state.

```hcl
variable "secret" {
  type      = string
  ephemeral = true
}

resource "nebius_msp_mlflow_v1alpha1_cluster" "example" {
  name           = "example"
  parent_id      = "your-project-id"
  admin_username = "user"

  sensitive = {
    version        = "1"
    admin_password = var.secret
  }
}
```

To avoid prompts during `terraform apply`, provide the ephemeral value through an environment variable such as `TF_VAR_secret`.

## Detecting Secret Changes With `nebius_hash`

If an ephemeral value changes, Terraform may need a deterministic value to notice that change during planning. The provider exposes `versioned_ephemeral_values` and the `nebius_hash` resource for this case.

```hcl
variable "secret" {
  type      = string
  ephemeral = true
}

provider "nebius" {
  versioned_ephemeral_values = {
    "secret_to_hash" = var.secret
  }
}

resource "nebius_hash" "secret_hash" {
  name = "secret_to_hash"
}

resource "nebius_msp_mlflow_v1alpha1_cluster" "example" {
  name           = "example"
  parent_id      = "your-project-id"
  admin_username = "user"

  sensitive = {
    version        = nebius_hash.secret_hash.hash
    admin_password = var.secret
  }
}
```

## Ephemeral Resources

Write-only arguments can also consume values from ephemeral resources:

```hcl
ephemeral "tls_private_key" "example" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "nebius_msp_mlflow_v1alpha1_cluster" "example" {
  name           = "example"
  parent_id      = "your-project-id"
  admin_username = "user"

  sensitive = {
    version        = "1"
    admin_password = ephemeral.tls_private_key.example.private_key_pem
  }
}
```

Ephemeral resources require Terraform 1.10.0 or later.
