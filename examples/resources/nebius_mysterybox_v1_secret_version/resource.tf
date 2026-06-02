variable "secret_value" {
  type      = string
  ephemeral = true
  sensitive = true
}

resource "nebius_mysterybox_v1_secret" "app" {
  parent_id = "project-id"
  name      = "app-secret"
}

resource "nebius_mysterybox_v1_secret_version" "current" {
  parent_id   = nebius_mysterybox_v1_secret.app.id
  name        = "current"
  set_primary = true

  sensitive = {
    version = "1"
    payload = [
      {
        key          = "password"
        string_value = var.secret_value
      }
    ]
  }
}
