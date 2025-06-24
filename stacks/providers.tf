terraform {
  required_version = ">= 1.0"
  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }

  backend "gcs" {
    bucket = "gen-lang-client-terraform-state"
    prefix = "article-summarizer/state"
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Data source for project information
data "google_project" "project" {
  project_id = var.project_id
}