resource "nebius_compute_v1_instance" "vm" {
  parent_id = "project-id"
  name      = "example-vm"

  resources = {
    platform = "cpu-e2"
    preset   = "2vcpu-8gb"
  }

  boot_disk = {
    attach_mode = "READ_WRITE"
    managed_disk = {
      name = "example-vm-boot"
      spec = {
        type           = "NETWORK_SSD"
        size_gibibytes = 64
        source_image_family = {
          parent_id    = "project-id"
          image_family = "ubuntu22.04"
        }
      }
    }
  }

  network_interfaces = [
    {
      name      = "eth0"
      subnet_id = "subnet-id"
      ip_address = {
        allocation_id = "private-allocation-id"
      }
    }
  ]
}
