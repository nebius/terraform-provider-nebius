resource "nebius_iam_v1_service_account" "storage" {
  parent_id = "project-id"
  name      = "storage-sa"
}

resource "nebius_iam_v2_access_key" "storage" {
  parent_id = "project-id"
  name      = "storage-access-key"

  account = {
    service_account = {
      id = nebius_iam_v1_service_account.storage.id
    }
  }

  secret_delivery_mode = "MYSTERY_BOX"
}
