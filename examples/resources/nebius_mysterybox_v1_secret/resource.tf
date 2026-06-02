resource "nebius_mysterybox_v1_secret" "app" {
  parent_id   = "project-id"
  name        = "app-secret"
  description = "Application secret"
}
