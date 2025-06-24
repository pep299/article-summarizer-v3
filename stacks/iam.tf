resource "google_service_account" "article_summarizer" {
  account_id = var.service_account_name
}

# Project-level IAM
resource "google_project_iam_member" "secret_manager_accessor" {
  project = var.project_id
  role    = "roles/secretmanager.secretAccessor"
  member  = "serviceAccount:${google_service_account.article_summarizer.email}"
}

# Cloud Run IAM
resource "google_cloud_run_service_iam_member" "public_access" {
  # checkov:skip=CKV_GCP_102:Public access required, API implements bearer token authentication
  service  = google_cloud_run_v2_service.article_summarizer.name
  location = google_cloud_run_v2_service.article_summarizer.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# GCS IAM
resource "google_storage_bucket_iam_member" "service_account_storage_admin" {
  bucket = google_storage_bucket.processed_articles.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${google_service_account.article_summarizer.email}"
}

resource "google_storage_bucket_iam_member" "developer_access" {
  bucket = google_storage_bucket.processed_articles.name
  role   = "roles/storage.admin"
  member = "user:eniarup@gmail.com"
}


