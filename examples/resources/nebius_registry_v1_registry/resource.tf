resource "nebius_registry_v1_registry" "app" {
  parent_id   = "project-id"
  name        = "app-registry"
  description = "Container images for the application"
}
