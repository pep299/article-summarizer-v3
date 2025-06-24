resource "google_storage_bucket" "processed_articles" {
  name          = var.bucket_name
  location      = var.region
  force_destroy = false

  uniform_bucket_level_access = false

}