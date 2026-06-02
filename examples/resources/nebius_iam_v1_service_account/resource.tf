resource "nebius_iam_v1_service_account" "terraform" {
  parent_id   = "project-id"
  name        = "terraform-sa"
  description = "Service account for Terraform automation"
}
