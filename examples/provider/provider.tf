terraform {
  required_providers {
    nebius = {
      source  = "nebius/nebius"
      version = ">= 0.6.8"
    }
  }
}

provider "nebius" {
  service_account = {
    account_id_env       = "SA_ID"
    public_key_id_env    = "AUTHKEY_PUBLIC_ID"
    private_key_file_env = "AUTHKEY_PRIVATE_PATH"
  }
}
