resource "nebius_vpc_v1_pool" "private" {
  parent_id  = "project-id"
  name       = "private-ipv4-pool"
  version    = "IPV4"
  visibility = "PRIVATE"

  cidrs = [
    {
      cidr            = "10.10.0.0/16"
      max_mask_length = 32
    }
  ]
}
