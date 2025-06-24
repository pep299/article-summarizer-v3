# Enable required APIs
resource "google_project_service" "artifactregistry" {
  service = "artifactregistry.googleapis.com"
  project = var.project_id
}

resource "google_project_service" "cloudbuild" {
  service = "cloudbuild.googleapis.com"
  project = var.project_id
}

resource "google_project_service" "run" {
  service = "run.googleapis.com"
  project = var.project_id
}

resource "google_project_service" "secretmanager" {
  service = "secretmanager.googleapis.com"
  project = var.project_id
}