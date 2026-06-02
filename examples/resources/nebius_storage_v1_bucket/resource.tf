resource "nebius_storage_v1_bucket" "state" {
  parent_id = "project-id"
  name      = "terraform-state-bucket"

  default_storage_class = "STANDARD"
  versioning_policy     = "ENABLED"
}
