resource "nebius_compute_v1_disk" "boot" {
  parent_id      = "project-id"
  name           = "ubuntu-boot-disk"
  type           = "NETWORK_SSD"
  size_gibibytes = 64

  source_image_family = {
    parent_id    = "project-id"
    image_family = "ubuntu22.04"
  }
}
