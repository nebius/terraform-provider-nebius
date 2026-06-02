resource "nebius_iam_v1_service_account" "terraform" {
  parent_id = "project-id"
  name      = "terraform-sa"
}

resource "nebius_iam_v1_auth_public_key" "terraform" {
  parent_id = "project-id"
  name      = "terraform-auth-key"

  account = {
    service_account = {
      id = nebius_iam_v1_service_account.terraform.id
    }
  }

  data = file("${path.module}/public.pem")
}
