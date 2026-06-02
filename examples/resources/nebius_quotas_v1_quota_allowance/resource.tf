resource "nebius_quotas_v1_quota_allowance" "bucket_count" {
  parent_id = "project-id"
  name      = "storage.bucket.count"
  limit     = 20
}
