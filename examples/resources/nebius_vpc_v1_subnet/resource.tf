resource "nebius_vpc_v1_network" "default" {
  parent_id = "project-id"
  name      = "default-network"
}

resource "nebius_vpc_v1_subnet" "default" {
  parent_id  = "project-id"
  name       = "default-subnet"
  network_id = nebius_vpc_v1_network.default.id

  ipv4_private_pools = {
    use_network_pools = true
  }
}
