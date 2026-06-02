resource "nebius_mk8s_v1_node_group" "default" {
  parent_id        = "mk8s-cluster-id"
  name             = "default"
  fixed_node_count = 2
  version          = "1.31"

  template = {
    resources = {
      platform = "cpu-e2"
      preset   = "2vcpu-8gb"
    }

    boot_disk = {
      type           = "NETWORK_SSD"
      size_gibibytes = 64
    }

    network_interfaces = [
      {
        subnet_id = "subnet-id"
      }
    ]
  }
}
