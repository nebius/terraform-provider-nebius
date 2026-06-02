variable "postgres_password" {
  type      = string
  ephemeral = true
  sensitive = true
}

resource "nebius_msp_postgresql_v1alpha1_cluster" "postgres" {
  parent_id  = "project-id"
  name       = "example-postgres"
  network_id = "network-id"

  bootstrap = {
    db_name   = "app"
    user_name = "app_user"
  }

  config = {
    version = "16"
    template = {
      resources = {
        platform = "cpu-e2"
        preset   = "2vcpu-8gb"
      }
      disk = {
        type           = "NETWORK_SSD"
        size_gibibytes = 32
      }
      hosts = {
        count = 1
      }
    }
  }

  sensitive = {
    version = "1"
    bootstrap = {
      user_password = var.postgres_password
    }
  }
}
