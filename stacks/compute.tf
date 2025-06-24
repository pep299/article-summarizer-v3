# Artifact Registry repository for Cloud Run source deployments
resource "google_artifact_registry_repository" "cloud_run_source_deploy" {
  # checkov:skip=CKV_GCP_84:Using default encryption is sufficient for this use case
  repository_id = "cloud-run-source-deploy"
  format        = "DOCKER"
  location      = var.region
  description   = "Cloud Run Source Deployments"


}

resource "google_cloud_run_v2_service" "article_summarizer" {
  name     = var.service_name
  location = var.region

  depends_on = [
    google_artifact_registry_repository.cloud_run_source_deploy
  ]

  lifecycle {
    ignore_changes = [
      template[0].containers[0].image,
      template[0].containers[0].resources[0].cpu_idle,
      template[0].containers[0].resources[0].startup_cpu_boost,
      client,
      client_version
    ]
  }

  template {
    service_account = google_service_account.article_summarizer.email

    containers {
      # Image will be managed by gcloud run deploy --source
      # This will be ignored on subsequent applies
      image = "gcr.io/cloudrun/hello"

      resources {
        limits = {
          memory = "512Mi"
          cpu    = "1000m"
        }
      }


      env {
        name = "GEMINI_API_KEY"
        value_source {
          secret_key_ref {
            secret  = "GEMINI_API_KEY"
            version = "latest"
          }
        }
      }

      env {
        name = "SLACK_BOT_TOKEN"
        value_source {
          secret_key_ref {
            secret  = "SLACK_BOT_TOKEN"
            version = "latest"
          }
        }
      }

      env {
        name = "WEBHOOK_AUTH_TOKEN"
        value_source {
          secret_key_ref {
            secret  = "WEBHOOK_AUTH_TOKEN"
            version = "latest"
          }
        }
      }
    }

    timeout = "540s"

    scaling {
      min_instance_count = 0
      max_instance_count = 40
    }
  }

  traffic {
    percent = 100
    type    = "TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST"
  }

}