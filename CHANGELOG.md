# Changelog

All notable changes to this project will be documented in this file.

## Unreleased

NOTES:

* changelog: Add versioned changelog entries for recent Terraform Registry releases.
* release: Added release-notes extraction and appending.

## 0.6.7 (May 20, 2026)

NOTES:

* provider: Update Nebius Go SDK to `v0.2.17` [GH-26]
* docs: Remove the completed post-publication checklist from the public repository [GH-24]
* ci: Add Slack failure notifications for release, test, and sync auto-approval workflows [GH-25]
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
