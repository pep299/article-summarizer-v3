resource "google_storage_bucket" "processed_articles" {
  name          = var.bucket_name
  location      = var.region
  force_destroy = false

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  public_access_prevention = "enforced"

  logging {
    log_bucket = google_storage_bucket.access_logs.name
  }
}

resource "google_storage_bucket" "access_logs" {
  # checkov:skip=CKV_GCP_62:Log bucket does not need its own access logging
  # checkov:skip=CKV_GCP_78:Log bucket does not need versioning
  name          = "${var.bucket_name}-access-logs"
  location      = var.region
  force_destroy = false

  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type = "Delete"
    }
  }
}