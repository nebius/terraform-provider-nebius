variable "mlflow_admin_password" {
  type      = string
  ephemeral = true
  sensitive = true
}

resource "nebius_iam_v1_service_account" "mlflow" {
  parent_id = "project-id"
  name      = "mlflow-sa"
}

resource "nebius_msp_mlflow_v1alpha1_cluster" "mlflow" {
  parent_id          = "project-id"
  name               = "example-mlflow"
  network_id         = "network-id"
  service_account_id = nebius_iam_v1_service_account.mlflow.id
  admin_username     = "admin"
  public_access      = false

  sensitive = {
    version        = "1"
    admin_password = var.mlflow_admin_password
  }
}
