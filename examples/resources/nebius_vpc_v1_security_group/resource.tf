resource "nebius_vpc_v1_security_group" "web" {
  parent_id  = "project-id"
  name       = "web-security-group"
  network_id = "network-id"
}
