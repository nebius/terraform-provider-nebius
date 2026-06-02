resource "nebius_vpc_v1_allocation" "vm_private_ip" {
  parent_id = "project-id"
  name      = "vm-private-ip"

  ipv4_private = {
    subnet_id = "subnet-id"
    cidr      = "/32"
  }
}
