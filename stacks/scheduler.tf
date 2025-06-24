# Cloud Scheduler jobs for periodic article processing

# Use the old URL format for compatibility with existing schedulers
locals {
  cloud_run_base_url = "https://article-summarizer-v3-740100231389.asia-northeast1.run.app"
}

resource "google_cloud_scheduler_job" "hatena_summarizer_scheduler" {
  name             = "hatena-summarizer-scheduler"
  schedule         = "0 */3 * * *" # Every 3 hours
  time_zone        = "Asia/Tokyo"
  region           = var.region
  attempt_deadline = "180s"

  http_target {
    http_method = "POST"
    uri         = "${local.cloud_run_base_url}/process/hatena"

    headers = {
      "Authorization" = "Bearer 5PFLwep1sk6PkBb0PVZtDaAJ0mub8rrzbkJ/cJofHpk="
    }
  }

  retry_config {
    retry_count          = 2
    max_retry_duration   = "0s"
    min_backoff_duration = "5s"
    max_backoff_duration = "3600s"
    max_doublings        = 5
  }
}

resource "google_cloud_scheduler_job" "lobsters_summarizer_scheduler" {
  name             = "lobsters-summarizer-scheduler"
  schedule         = "10 */6 * * *" # Every 6 hours at :10
  time_zone        = "Asia/Tokyo"
  region           = var.region
  attempt_deadline = "180s"

  http_target {
    http_method = "POST"
    uri         = "${local.cloud_run_base_url}/process/lobsters"

    headers = {
      "Authorization" = "Bearer 5PFLwep1sk6PkBb0PVZtDaAJ0mub8rrzbkJ/cJofHpk="
    }
  }

  retry_config {
    retry_count          = 2
    max_retry_duration   = "0s"
    min_backoff_duration = "5s"
    max_backoff_duration = "3600s"
    max_doublings        = 5
  }
}

resource "google_cloud_scheduler_job" "reddit_summarizer_scheduler" {
  name             = "reddit-summarizer-scheduler"
  schedule         = "20 */6 * * *" # Every 6 hours at :20
  time_zone        = "Asia/Tokyo"
  region           = var.region
  attempt_deadline = "180s"

  http_target {
    http_method = "POST"
    uri         = "${local.cloud_run_base_url}/process/reddit"

    headers = {
      "Authorization" = "Bearer 5PFLwep1sk6PkBb0PVZtDaAJ0mub8rrzbkJ/cJofHpk="
    }
  }

  retry_config {
    retry_count          = 2
    max_retry_duration   = "0s"
    min_backoff_duration = "5s"
    max_backoff_duration = "3600s"
    max_doublings        = 5
  }
}