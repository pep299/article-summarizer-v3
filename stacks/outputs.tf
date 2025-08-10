output "cloud_run_url" {
  description = "Cloud Run service URL"
  value       = google_cloud_run_v2_service.article_summarizer.uri
}

output "service_account_email" {
  description = "Service account email"
  value       = google_service_account.article_summarizer.email
}

output "storage_bucket_name" {
  description = "GCS bucket name"
  value       = google_storage_bucket.processed_articles.name
}

output "storage_bucket_url" {
  description = "GCS bucket URL"
  value       = google_storage_bucket.processed_articles.url
}


output "artifact_registry_repository" {
  description = "Artifact Registry repository name (main)"
  value       = google_artifact_registry_repository.article_summarizer.name
}

output "legacy_artifact_registry_repository" {
  description = "Legacy Artifact Registry repository name"
  value       = google_artifact_registry_repository.cloud_run_source_deploy.name
}

output "current_deployed_image" {
  description = "Currently deployed Docker image"
  value       = google_cloud_run_v2_service.article_summarizer.template[0].containers[0].image
}

output "project_number" {
  description = "GCP Project Number"
  value       = data.google_project.project.number
}