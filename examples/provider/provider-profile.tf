terraform {
  required_providers {
    nebius = {
      source  = "nebius/nebius"
      version = ">= 0.6.8"
    }
  }
}

provider "nebius" {
  profile = {
    name = "default"
  }
}
