terraform {
  required_providers {
    nebius = {
      source = "nebius/nebius"
    }
  }
}

provider "nebius" {
  profile = {
    name = "default"
  }
}
