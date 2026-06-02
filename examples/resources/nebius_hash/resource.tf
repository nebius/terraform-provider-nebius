variable "secret_value" {
  type      = string
  ephemeral = true
}

provider "nebius" {
  versioned_ephemeral_values = {
    app_secret = var.secret_value
  }
}

resource "nebius_hash" "app_secret" {
  name = "app_secret"
}
