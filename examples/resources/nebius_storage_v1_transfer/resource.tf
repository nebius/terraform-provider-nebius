variable "transfer_secret_access_key" {
  type      = string
  ephemeral = true
  sensitive = true
}

resource "nebius_storage_v1_transfer" "copy" {
  parent_id          = "project-id"
  name               = "copy-between-buckets"
  overwrite_strategy = "IF_NEWER"
  touch_unmanaged    = false

  source = {
    nebius = {
      bucket_name = "source-bucket"
      region      = "eu-north1"
      access_key = {
        access_key_id = "source-access-key-id"
      }
    }
  }

  destination = {
    nebius = {
      bucket_name = "destination-bucket"
      region      = "eu-north1"
      access_key = {
        access_key_id = "destination-access-key-id"
      }
    }
  }

  after_one_iteration = {}

  sensitive = {
    version = "1"
    source = {
      nebius = {
        access_key = {
          secret_access_key = var.transfer_secret_access_key
        }
      }
    }
    destination = {
      nebius = {
        access_key = {
          secret_access_key = var.transfer_secret_access_key
        }
      }
    }
  }
}
