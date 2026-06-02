resource "nebius_mk8s_v1_cluster" "cluster" {
  parent_id = "project-id"
  name      = "example-cluster"

  control_plane = {
    subnet_id         = "subnet-id"
    version           = "1.31"
    etcd_cluster_size = 1
    endpoints = {
      public_endpoint = {}
    }
  }
}
