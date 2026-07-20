# Changelog

All notable changes to this project will be documented in this file.

## 0.6.29 (July 20, 2026)

NOTES:

* deps: Update `google.golang.org/grpc` from `v1.82.0` to `v1.82.1`.

## 0.6.28 (July 16, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.39`.

FEATURES:

* Added new resources and data sources: [nebius_capacity_v1_capacity_allowance](./docs/resources/capacity_v1_capacity_allowance.md).

## 0.6.27 (July 14, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.38`.

IMPROVEMENTS:

* Changed attributes for resource [nebius_compute_v1_disk](./docs/resources/compute_v1_disk.md):
    * Added: `source_snapshot_id`
* Changed attributes for data source [nebius_compute_v1_disk](./docs/data-sources/compute_v1_disk.md):
    * Added: `source_snapshot_id`
* Changed attributes for data source [nebius_compute_v1_instance](./docs/data-sources/compute_v1_instance.md):
    * Added: `boot_disk.managed_disk.spec.source_snapshot_id`, `secondary_disks.managed_disk.spec.source_snapshot_id`

BREAKING CHANGES:

* Changed attributes for resource [nebius_compute_v1_instance](./docs/resources/compute_v1_instance.md):
    * Added: `boot_disk.managed_disk.spec.source_snapshot_id`, `secondary_disks.managed_disk.spec.source_snapshot_id`
    * Became required: `boot_disk`

## 0.6.26 (July 8, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.37`.

IMPROVEMENTS:

* Changed attributes for data source [nebius_compute_v1_image](./docs/data-sources/compute_v1_image.md):
    * Added: `source_disk_snapshot_id`

## 0.6.25 (July 7, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.36`.
* deps: Update `google.golang.org/genproto/googleapis/rpc` from `v0.0.0-20260319201613-d00831a3d3e7` to `v0.0.0-20260414002931-afd174a4e478`.
* deps: Update `google.golang.org/grpc` from `v1.81.1` to `v1.82.0`.

## 0.6.24 (July 6, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.35`.

FEATURES:

* Added new resources and data sources: [nebius_compute_v1_disk_snapshot](./docs/resources/compute_v1_disk_snapshot.md).

IMPROVEMENTS:

* Changed attributes for resource [nebius_compute_v1_disk](./docs/resources/compute_v1_disk.md):
    * Added: `status.lock_state.snapshots`
* Changed attributes for data source [nebius_compute_v1_disk](./docs/data-sources/compute_v1_disk.md):
    * Added: `status.lock_state.snapshots`

## 0.6.23 (July 2, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.34`.

## 0.6.22 (June 29, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.33`.

IMPROVEMENTS:

* Changed attributes for resource [nebius_mk8s_v1_node_group](./docs/resources/mk8s_v1_node_group.md):
    * Added: `template.nvlink`
* Changed attributes for resource [nebius_vpc_v1_subnet](./docs/resources/vpc_v1_subnet.md):
    * Added: `status.ipv4_private_pools`, `status.ipv4_public_pools`
    * Deprecated: `status.ipv4_private_cidrs`, `status.ipv4_public_cidrs`
* Changed attributes for data source [nebius_mk8s_v1_node_group](./docs/data-sources/mk8s_v1_node_group.md):
    * Added: `template.nvlink`
* Changed attributes for data source [nebius_vpc_v1_subnet](./docs/data-sources/vpc_v1_subnet.md):
    * Added: `status.ipv4_private_pools`, `status.ipv4_public_pools`
    * Deprecated: `status.ipv4_private_cidrs`, `status.ipv4_public_cidrs`

## 0.6.21 (June 25, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.32`.

FEATURES:

* Added new resources and data sources: [nebius_tunnel_v1_tunnel](./docs/resources/tunnel_v1_tunnel.md).

## 0.6.20 (June 24, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.31`.

IMPROVEMENTS:

* Changed attributes for resource [nebius_storage_v1_transfer](./docs/resources/storage_v1_transfer.md):
    * Added: `enable_deletes_in_destination`, `status.last_iteration.objects_deleted_count`
* Changed attributes for data source [nebius_storage_v1_transfer](./docs/data-sources/storage_v1_transfer.md):
    * Added: `enable_deletes_in_destination`, `status.last_iteration.objects_deleted_count`

## 0.6.19 (June 23, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.30`.

IMPROVEMENTS:

* Changed attributes for resource [nebius_compute_v1_nvl_instance_group](./docs/resources/compute_v1_nvl_instance_group.md):
    * Added: `size`
* Changed attributes for data source [nebius_compute_v1_nvl_instance_group](./docs/data-sources/compute_v1_nvl_instance_group.md):
    * Added: `size`

## 0.6.18 (June 22, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.29`.

IMPROVEMENTS:

* Changed [provider](docs/index.md) attributes:
    * Added: `impersonate_service_account_id`
* Changed attributes for resource [nebius_kms_v1_symmetric_key](./docs/resources/kms_v1_symmetric_key.md):
    * Became computed: `rotation_period`

## 0.6.17 (June 18, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.28`.

## 0.6.16 (June 16, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.27`.

IMPROVEMENTS:

* Changed attributes for resource [nebius_mk8s_v1_cluster](./docs/resources/mk8s_v1_cluster.md):
    * Added: `control_plane.endpoints.public_endpoint.allowed_cidrs`
* Changed attributes for resource [nebius_mk8s_v1_node_group](./docs/resources/mk8s_v1_node_group.md):
    * Added: `template.network_interfaces.security_groups`
* Changed attributes for data source [nebius_mk8s_v1_cluster](./docs/data-sources/mk8s_v1_cluster.md):
    * Added: `control_plane.endpoints.public_endpoint.allowed_cidrs`
* Changed attributes for data source [nebius_mk8s_v1_node_group](./docs/data-sources/mk8s_v1_node_group.md):
    * Added: `template.network_interfaces.security_groups`

## 0.6.15 (June 15, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.26`.

FEATURES:

* Added new resources and data sources: [nebius_kms_v1_asymmetric_key](./docs/resources/kms_v1_asymmetric_key.md), [nebius_kms_v1_symmetric_key](./docs/resources/kms_v1_symmetric_key.md).

## 0.6.14 (June 11, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.25`.

FEATURES:

* Added new resources: [nebius_dns_v1_record](./docs/resources/dns_v1_record.md), [nebius_dns_v1_zone](./docs/resources/dns_v1_zone.md).

IMPROVEMENTS:

* Changed attributes for resource [nebius_vpc_v1_route](./docs/resources/vpc_v1_route.md):
    * Added: `status.priority`
* Changed attributes for data source [nebius_vpc_v1_route](./docs/data-sources/vpc_v1_route.md):
    * Added: `status.priority`

## 0.6.13 (June 9, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.24`.

IMPROVEMENTS:

* Changed attributes for resource [nebius_vpc_v1_allocation](./docs/resources/vpc_v1_allocation.md):
    * Added: `status.assignment.network_interface.type`
* Changed attributes for resource [nebius_vpc_v1alpha1_allocation](./docs/resources/vpc_v1alpha1_allocation.md):
    * Added: `status.assignment.network_interface.type`
* Changed attributes for data source [nebius_vpc_v1_allocation](./docs/data-sources/vpc_v1_allocation.md):
    * Added: `status.assignment.network_interface.type`
* Changed attributes for data source [nebius_vpc_v1alpha1_allocation](./docs/data-sources/vpc_v1alpha1_allocation.md):
    * Added: `status.assignment.network_interface.type`

## 0.6.12 (June 4, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.22`.

IMPROVEMENTS:

* Changed attributes for resource [nebius_compute_v1_disk](./docs/resources/compute_v1_disk.md):
    * Added: `status.managed_by`
* Changed attributes for resource [nebius_mk8s_v1_node_group](./docs/resources/mk8s_v1_node_group.md):
    * Added: `template.max_pods`
* Changed attributes for data source [nebius_compute_v1_disk](./docs/data-sources/compute_v1_disk.md):
    * Added: `status.managed_by`
* Changed attributes for data source [nebius_mk8s_v1_node_group](./docs/data-sources/mk8s_v1_node_group.md):
    * Added: `template.max_pods`

## 0.6.11 (June 3, 2026)

NOTES:

* Internal improvements.

## 0.6.10 (June 2, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.21`.

## 0.6.9 (June 1, 2026)

NOTES:

* Internal improvements.

## 0.6.8 (May 28, 2026)

NOTES:

* changelog: Add versioned changelog entries for recent Terraform Registry releases.
* release: Added release-notes extraction and appending.
* provider: Update Nebius Go SDK to `v0.2.20`.
* deps: Update `google.golang.org/grpc` from `v1.81.0` to `v1.81.1`.

IMPROVEMENTS:


* Changed attributes for resource [nebius_mk8s_v1_node_group](./docs/resources/mk8s_v1_node_group.md):
    * Added: `status.strategy`
* Changed attributes for resource [nebius_storage_v1_bucket](./docs/resources/storage_v1_bucket.md):
    * Added: `lifecycle_configuration.rules.filter.tags`
* Changed attributes for data source [nebius_mk8s_v1_node_group](./docs/data-sources/mk8s_v1_node_group.md):
    * Added: `status.strategy`
* Changed attributes for data source [nebius_storage_v1_bucket](./docs/data-sources/storage_v1_bucket.md):
    * Added: `lifecycle_configuration.rules.filter.tags`

BREAKING CHANGES:

* Removed resources and data sources: `nebius_compute_v1alpha1_disk`, `nebius_compute_v1alpha1_filesystem`, `nebius_compute_v1alpha1_gpu_cluster`, `nebius_compute_v1alpha1_instance`.
* Removed data sources: `nebius_compute_v1alpha1_image`.

## 0.6.7 (May 20, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.17` [GH-26]
* docs: Remove the completed post-publication checklist from the public repository [GH-24]
* ci: Add workflow failure notifications for release, test, and sync auto-approval workflows [GH-25]
* ci: Run the main test workflow nightly and limit Terraform prerelease matrix checks to pull requests [GH-25]

## 0.6.6 (May 19, 2026)

NOTES:

* provider: Publish signed release assets for HashiCorp Terraform Registry ingestion [GH-23]

FEATURES:

* **New Resource:** `nebius_compute_v1_nvl_instance_group` [GH-23]
* **New Data Source:** `nebius_compute_v1_nvl_instance_group` [GH-23]

IMPROVEMENTS:

* resource/compute_v1_instance: Add `nvl_instance_group_id` for associating VMs with NVLink instance groups [GH-23]
* resource/compute_v1_instance: Mark preemptible `priority` as deprecated [GH-23]
* resource/mk8s_v1_cluster: Expose aggregated status events [GH-23]
* resource/storage_v1_bucket: Add the `FILESYSTEM` storage class [GH-23]
* resource/vpc_v1_route: Expose route type information for static and redistributed routes [GH-23]
* ci: Add Go tests, generated documentation checks, Terraform Registry documentation validation, and E2E test workflows [GH-17]
* ci: Enable auto-approval workflow for generated sync pull requests [GH-18]
* release: Enable the signed GoReleaser publishing workflow and manual release dispatch [GH-22]
