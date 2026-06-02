resource "nebius_vpc_v1_security_group" "web" {
  parent_id  = "project-id"
  name       = "web-security-group"
  network_id = "network-id"
}

resource "nebius_vpc_v1_security_rule" "allow_https" {
  parent_id = nebius_vpc_v1_security_group.web.id
  name      = "allow-https"
  access    = "ALLOW"
  protocol  = "TCP"
  priority  = 100
  type      = "STATEFUL"

  ingress = {
    source_cidrs      = ["0.0.0.0/0"]
    destination_ports = [443]
  }
}
