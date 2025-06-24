variable "project_id" {
  description = "Google Cloud Project ID"
  type        = string
  default     = "gen-lang-client-0715048106"
}

variable "region" {
  description = "Google Cloud Region"
  type        = string
  default     = "asia-northeast1"
}

variable "service_name" {
  description = "Cloud Run service name"
  type        = string
  default     = "article-summarizer-v3"
}

variable "bucket_name" {
  description = "GCS bucket name for processed articles"
  type        = string
  default     = "article-summarizer-processed-articles"
}

variable "index_file_name" {
  description = "Index file name in GCS bucket"
  type        = string
  default     = "index-v2.json"
}

variable "service_account_name" {
  description = "Service account name"
  type        = string
  default     = "article-summarizer-sa"
}